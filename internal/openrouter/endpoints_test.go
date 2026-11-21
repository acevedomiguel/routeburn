package openrouter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKeyMapsNullableFields(t *testing.T) {
	body := `{
	  "data": {
	    "label": "sk-or-v1-...",
	    "limit": null,
	    "limit_reset": null,
	    "limit_remaining": null,
	    "include_byok_in_limit": false,
	    "usage": 3.5,
	    "usage_daily": 0.5,
	    "usage_weekly": 1.5,
	    "usage_monthly": 3.5,
	    "byok_usage": 0,
	    "byok_usage_daily": 0,
	    "byok_usage_weekly": 0,
	    "byok_usage_monthly": 0,
	    "is_free_tier": false
	  }
	}`

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key" {
			t.Errorf("path = %q, want /key", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer api-test-key" {
			t.Errorf("authorization = %q, want the regular API key", got)
		}
		io.WriteString(w, body)
	}))

	got, err := c.Key(context.Background())
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if got.Limit != nil {
		t.Errorf("limit = %v, want nil for unlimited", *got.Limit)
	}
	if got.LimitRemaining != nil {
		t.Errorf("limit remaining = %v, want nil", *got.LimitRemaining)
	}
	if got.UsageDaily != 0.5 || got.UsageWeekly != 1.5 || got.UsageMonthly != 3.5 {
		t.Errorf("usage = %+v", got)
	}
}

func TestKeyWithLimitMapsValues(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"label":"prod","limit":20,"limit_remaining":5,"limit_reset":"monthly","usage_daily":1}}`)
	}))

	got, err := c.Key(context.Background())
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if got.Limit == nil || *got.Limit != 20 {
		t.Fatalf("limit = %v, want 20", got.Limit)
	}
	if got.LimitRemaining == nil || *got.LimitRemaining != 5 {
		t.Fatalf("limit remaining = %v, want 5", got.LimitRemaining)
	}
	if got.LimitReset != "monthly" {
		t.Errorf("limit reset = %q, want monthly", got.LimitReset)
	}
}

func TestKeyWithoutAPIKeyIsSkippable(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be sent without a regular API key")
	}))
	c.apiKey = ""

	_, err := c.Key(context.Background())
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("err = %v, want ErrMissingAPIKey", err)
	}
}

func TestAnalyticsMetaIsFetchedWithoutBody(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/meta" {
			t.Errorf("path = %q, want /analytics/meta", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		io.WriteString(w, `{"data":{"metrics":["total_usage","tokens_total"]}}`)
	}))

	meta, err := c.AnalyticsMeta(context.Background())
	if err != nil {
		t.Fatalf("AnalyticsMeta: %v", err)
	}
	if len(meta.Metrics) != 2 || meta.Metrics[0] != "total_usage" {
		t.Errorf("metrics = %v", meta.Metrics)
	}
}

func TestAnalyticsQuerySendsDocumentedShape(t *testing.T) {
	var seen string
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/query" {
			t.Errorf("path = %q, want /analytics/query", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content type = %q", ct)
		}
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		io.WriteString(w, `{"data":[{"day":"2026-11-20","total_usage":12.5,"tokens_total":4200}]}`)
	}))

	rows, err := c.QueryAnalytics(context.Background(), AnalyticsQuery{
		Metrics:     []string{"total_usage", "tokens_total"},
		Granularity: "day",
		Start:       utcTime(2026, 10, 23),
		End:         utcTime(2026, 11, 22),
	})
	if err != nil {
		t.Fatalf("QueryAnalytics: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].CreditSpend != 12.5 || rows[0].Tokens != 4200 {
		t.Errorf("row = %+v", rows[0])
	}
	for _, want := range []string{`"metrics":["total_usage","tokens_total"]`, `"granularity":"day"`, `"time_range"`} {
		if !strings.Contains(seen, want) {
			t.Errorf("request body missing %q: %s", want, seen)
		}
	}
	if !strings.Contains(seen, `"start":"2026-10-23T00:00:00Z"`) || !strings.Contains(seen, `"end":"2026-11-22T00:00:00Z"`) {
		t.Errorf("request body time range is wrong: %s", seen)
	}
}

func TestAnalyticsQueryToleratesAlternativeShapes(t *testing.T) {
	tests := map[string]string{
		"nested data rows": `{"data":{"rows":[{"date":"2026-11-20","usage":1.5,"tokens":10}]}}`,
		"top-level rows":   `{"rows":[{"day":"2026-11-20","total_usage":1.5,"tokens_total":10}]}`,
		"flat array":       `[{"day":"2026-11-20","total_usage":1.5,"tokens_total":10}]`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, body)
			}))
			rows, err := c.QueryAnalytics(context.Background(), AnalyticsQuery{Metrics: []string{"total_usage"}, Granularity: "day", Start: utcTime(2026, 11, 1), End: utcTime(2026, 11, 2)})
			if err != nil {
				t.Fatalf("QueryAnalytics: %v", err)
			}
			if len(rows) != 1 || rows[0].CreditSpend != 1.5 || rows[0].Tokens != 10 {
				t.Errorf("rows = %+v", rows)
			}
		})
	}
}

func TestAnalyticsUnsupportedMetricIsReportedNotGuessed(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"message":"unknown metric cache_hit_rate"}}`)
	}))

	_, err := c.QueryAnalytics(context.Background(), AnalyticsQuery{Metrics: []string{"cache_hit_rate"}, Granularity: "day", Start: utcTime(2026, 11, 1), End: utcTime(2026, 11, 2)})
	if err == nil {
		t.Fatal("expected an error for an unsupported metric")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("err = %v, want ErrBadRequest", err)
	}
}
