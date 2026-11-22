package openrouter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.Handler, opts ...Option) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	base := []Option{
		WithBaseURL(srv.URL),
		WithManagementKey("mgmt-test-key"),
		WithAPIKey("api-test-key"),
		WithSleep(func(time.Duration) {}),
	}
	c, err := NewClient(append(base, opts...)...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestNewClientRejectsInsecureBaseURL(t *testing.T) {
	if _, err := NewClient(WithBaseURL("http://openrouter.ai/api/v1")); err == nil {
		t.Fatal("expected an error for a non-TLS production base URL")
	}
	if _, err := NewClient(WithBaseURL("http://127.0.0.1:8080")); err != nil {
		t.Fatalf("loopback base URL must be allowed for tests: %v", err)
	}
}

func TestFetchCreditsMapsResponse(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/credits" {
			t.Errorf("path = %q, want /credits", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer mgmt-test-key" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "RouteBurn/") {
			t.Errorf("user agent = %q", got)
		}
		io.WriteString(w, `{"data":{"total_credits":100.5,"total_usage":25.75}}`)
	}))

	got, err := c.Credits(context.Background())
	if err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if got.TotalPurchased != 100.5 || got.TotalUsed != 25.75 {
		t.Errorf("credits = %+v", got)
	}
	if got.Remaining < 74.7499 || got.Remaining > 74.7501 {
		t.Errorf("remaining = %v, want 74.75", got.Remaining)
	}
}

func TestFetchCreditsPreservesNegativeBalance(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"total_credits":10,"total_usage":12.5}}`)
	}))

	got, err := c.Credits(context.Background())
	if err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if got.Remaining > -2.4999 || got.Remaining < -2.5001 {
		t.Errorf("remaining = %v, want -2.5 (never clamped)", got.Remaining)
	}
}

func TestFetchCreditsRequiresManagementKey(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be sent without a management key")
	}))
	c.managementKey = ""

	_, err := c.Credits(context.Background())
	if !errors.Is(err, ErrMissingManagementKey) {
		t.Fatalf("err = %v, want ErrMissingManagementKey", err)
	}
}

func TestTypedErrorsCarryStatusAndEndpoint(t *testing.T) {
	tests := []struct {
		status  int
		wantErr error
	}{
		{401, ErrUnauthorized},
		{403, ErrForbidden},
		{404, ErrNotFound},
		{429, ErrRateLimited},
		{500, ErrServer},
		{503, ErrServer},
	}

	for _, tt := range tests {
		c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
			io.WriteString(w, `{"error":{"message":"nope"}}`)
		}))

		_, err := c.Credits(context.Background())
		if err == nil {
			t.Fatalf("status %d: expected an error", tt.status)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("status %d: err = %T %v, want *APIError", tt.status, err, err)
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("status %d: errors.Is(%v, %v) = false", tt.status, err, tt.wantErr)
		}
		if apiErr.StatusCode != tt.status {
			t.Errorf("status = %d, want %d", apiErr.StatusCode, tt.status)
		}
		if apiErr.Endpoint != "/credits" {
			t.Errorf("endpoint = %q, want /credits", apiErr.Endpoint)
		}
		if strings.Contains(err.Error(), "mgmt-test-key") || strings.Contains(err.Error(), "Bearer") {
			t.Errorf("error leaks credentials: %v", err)
		}
	}
}

func TestMalformedJSONIsTypedAndDoesNotLeakBody(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html>secret-ish body</html>`)
	}))

	_, err := c.Credits(context.Background())
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("err = %v, want ErrMalformedResponse", err)
	}
	if strings.Contains(err.Error(), "secret-ish") {
		t.Errorf("error must not echo the response body: %v", err)
	}
}

func TestRetrySucceedsAfterTransientFailures(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, `{}`)
			return
		}
		io.WriteString(w, `{"data":{"total_credits":1,"total_usage":0}}`)
	}))

	if _, err := c.Credits(context.Background()); err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3 (initial + two retries)", got)
	}
}

func TestRetryStopsAfterTwoRetries(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))

	if _, err := c.Credits(context.Background()); err == nil {
		t.Fatal("expected failure")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestRetryHonorsRetryAfterSeconds(t *testing.T) {
	var calls atomic.Int32
	var slept []time.Duration
	var mu sync.Mutex

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		io.WriteString(w, `{"data":{"total_credits":1,"total_usage":0}}`)
	}), WithSleep(func(d time.Duration) { mu.Lock(); slept = append(slept, d); mu.Unlock() }))

	if _, err := c.Credits(context.Background()); err != nil {
		t.Fatalf("Credits: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(slept) != 1 || slept[0] != 2*time.Second {
		t.Errorf("slept = %v, want [2s]", slept)
	}
}

func TestRateLimitResetHeaderIsExposedForScheduling(t *testing.T) {
	reset := time.Now().Add(45 * time.Second).UTC().Format(http.TimeFormat)
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Reset", reset)
		io.WriteString(w, `{"data":{"total_credits":1,"total_usage":0}}`)
	}))

	if _, err := c.Credits(context.Background()); err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if c.NextRefreshAllowedAt().IsZero() {
		t.Error("next refresh allowed time must be recorded from X-RateLimit-Reset")
	}
}

func TestNoRetryOnClientErrors(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))

	if _, err := c.Credits(context.Background()); err == nil {
		t.Fatal("expected failure")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls = %d, want 1 (400 must not be retried)", got)
	}
}

func TestContextCancellationStopsRetries(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}), WithSleep(func(time.Duration) {}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Credits(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("calls = %d, want 0", got)
	}
}

func TestResponseBodyIsSizeLimited(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"total_credits":1,"total_usage":0},"padding":"`)
		chunk := strings.Repeat("x", 4096)
		for i := 0; i < 2048; i++ {
			io.WriteString(w, chunk)
		}
		io.WriteString(w, `"}`)
	}))

	_, err := c.Credits(context.Background())
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("err = %v, want ErrResponseTooLarge", err)
	}
}
