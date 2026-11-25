package routeburn_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	routeburn "github.com/acevedomiguel/routeburn"
)

type fakeAPI struct{}

func (fakeAPI) Credits(context.Context) (routeburn.CreditSummary, error) {
	return routeburn.CreditSummary{TotalPurchased: 100, TotalUsed: 57.9, Remaining: 42.1}, nil
}

func (fakeAPI) Key(context.Context) (routeburn.KeyInfo, error) {
	return routeburn.KeyInfo{}, routeburn.ErrMissingAPIKey
}

func (fakeAPI) QueryAnalytics(context.Context, routeburn.AnalyticsQuery) ([]routeburn.AnalyticsRow, error) {
	return nil, nil
}

func (fakeAPI) NextRefreshAllowedAt() time.Time { return time.Time{} }

func fixedTime() time.Time {
	return time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
}

func TestFetchUsesPublicAPI(t *testing.T) {
	snap, err := routeburn.Fetch(context.Background(), fakeAPI{}, routeburn.FetchOptions{Now: fixedTime()})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := len(snap.Daily); got != 30 {
		t.Fatalf("daily buckets = %d, want 30", got)
	}
	if snap.Account.Remaining != 42.1 {
		t.Errorf("remaining = %v, want 42.1", snap.Account.Remaining)
	}
	if snap.Key != nil {
		t.Errorf("key = %+v, want nil for a missing regular key", snap.Key)
	}
}

func TestNewClientDelegatesOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/credits":
			if got := r.Header.Get("Authorization"); got != "Bearer synthetic-management" {
				t.Errorf("management authorization = %q", got)
			}
			if got := r.Header.Get("User-Agent"); got != "RouteBurn/test" {
				t.Errorf("user agent = %q", got)
			}
			fmt.Fprint(w, `{"data":{"total_credits":100,"total_usage":57.9}}`)
		case "/key":
			if got := r.Header.Get("Authorization"); got != "Bearer synthetic-regular" {
				t.Errorf("regular authorization = %q", got)
			}
			fmt.Fprint(w, `{"data":{"label":"synthetic","limit":null,"limit_remaining":null,"limit_reset":null,"usage_daily":1,"usage_weekly":2,"usage_monthly":3}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := routeburn.NewClient(
		routeburn.WithBaseURL(server.URL),
		routeburn.WithManagementKey("synthetic-management"),
		routeburn.WithAPIKey("synthetic-regular"),
		routeburn.WithHTTPClient(server.Client()),
		routeburn.WithVersion("test"),
		routeburn.WithSleep(func(time.Duration) {}),
		routeburn.WithClock(fixedTime),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	credits, err := client.Credits(context.Background())
	if err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if credits.Remaining != 42.1 {
		t.Errorf("remaining = %v, want 42.1", credits.Remaining)
	}
	key, err := client.Key(context.Background())
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if key.Label != "synthetic" || key.Limit != nil || key.LimitRemaining != nil {
		t.Errorf("key = %+v", key)
	}
}
