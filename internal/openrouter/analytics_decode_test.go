package openrouter

import (
	"encoding/json"
	"testing"
)

// The live /analytics/query response nests the row array twice:
//
//	{"data":{"data":[{...}],"metadata":{...},"cachedAt":...}}
//
// and buckets are keyed "date__day", not "day". Getting this wrong silently
// empties the chart while /credits keeps working.
func TestDecodeAnalyticsLiveEnvelope(t *testing.T) {
	raw := json.RawMessage(`{
	  "data": {
	    "data": [
	      {"date__day": "2026-11-21", "total_usage": 1.729425, "tokens_total": 80447889},
	      {"date__day": "2026-11-20", "total_usage": 1.480667, "tokens_total": 20118539}
	    ],
	    "metadata": {"query_time_ms": 14, "row_count": 2, "truncated": false},
	    "cachedAt": 1795256182527
	  }
	}`)

	rows, err := decodeAnalyticsRows(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if got := rows[0].Day.Format("2006-01-02"); got != "2026-11-21" {
		t.Errorf("day = %q, want 2026-11-21", got)
	}
	if rows[0].CreditSpend != 1.729425 {
		t.Errorf("spend = %v", rows[0].CreditSpend)
	}
	if rows[0].Tokens != 80447889 {
		t.Errorf("tokens = %d", rows[0].Tokens)
	}
}

// Model rows carry "date__day" plus "model" in the same flat object.
func TestDecodeAnalyticsLiveModelRows(t *testing.T) {
	raw := json.RawMessage(`{
	  "data": {
	    "data": [
	      {"date__day": "2026-11-21", "model": "moonshotai/kimi-k3.1-pro-20261023", "total_usage": 1.160576},
	      {"date__day": "2026-11-21", "model": "deepseek/deepseek-v4.1-flash-20260910", "total_usage": 0.568849}
	    ],
	    "metadata": {"row_count": 2}
	  }
	}`)

	rows, err := decodeAnalyticsRows(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Model != "moonshotai/kimi-k3.1-pro-20261023" {
		t.Errorf("model = %q", rows[0].Model)
	}
	if rows[0].Day.Format("2006-01-02") != "2026-11-21" {
		t.Errorf("day = %s", rows[0].Day)
	}
}

// An empty result set is a valid answer, not a shape error.
func TestDecodeAnalyticsEmptyRows(t *testing.T) {
	raw := json.RawMessage(`{"data":{"data":[],"metadata":{"row_count":0}}}`)
	rows, err := decodeAnalyticsRows(raw)
	if err != nil {
		t.Fatalf("empty result must not error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %d", len(rows))
	}
}

// The previously assumed shapes must keep working.
func TestDecodeAnalyticsLegacyShapes(t *testing.T) {
	cases := map[string]string{
		"bare array":  `[{"day":"2026-11-21","total_usage":1.5}]`,
		"rows key":    `{"rows":[{"day":"2026-11-21","total_usage":1.5}]}`,
		"results key": `{"results":[{"day":"2026-11-21","total_usage":1.5}]}`,
		"items key":   `{"items":[{"day":"2026-11-21","total_usage":1.5}]}`,
		"nested rows": `{"data":{"rows":[{"day":"2026-11-21","total_usage":1.5}]}}`,
		"nested data": `{"data":{"data":[{"date__day":"2026-11-21","total_usage":1.5}]}}`,
	}

	for name, body := range cases {
		rows, err := decodeAnalyticsRows(json.RawMessage(body))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(rows) != 1 {
			t.Errorf("%s: rows = %d", name, len(rows))
		}
	}
}

// Aliases the API may use for the same metric must all decode.
func TestDecodeAnalyticsMetricAliases(t *testing.T) {
	for _, field := range []string{"total_usage", "usage", "credit_spend", "credits"} {
		body := `{"data":{"data":[{"date__day":"2026-11-21","` + field + `":2.5}]}}`
		rows, err := decodeAnalyticsRows(json.RawMessage(body))
		if err != nil {
			t.Fatalf("%s: %v", field, err)
		}
		if rows[0].CreditSpend != 2.5 {
			t.Errorf("%s: spend = %v", field, rows[0].CreditSpend)
		}
	}
}
