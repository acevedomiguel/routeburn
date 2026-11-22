package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the only production endpoint. Insecure HTTP is
	// rejected unless the host is loopback, which tests rely on.
	DefaultBaseURL = "https://openrouter.ai/api/v1"

	defaultTimeout   = 15 * time.Second
	maxResponseBytes = 4 << 20
	maxRetries       = 2
	backoffBase      = 250 * time.Millisecond
)

// Client talks to the OpenRouter REST API. It is safe for concurrent use.
type Client struct {
	baseURL       string
	managementKey string
	apiKey        string
	http          *http.Client
	userAgent     string
	sleep         func(time.Duration)
	now           func() time.Time

	mu            sync.Mutex
	nextRefreshAt time.Time
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL. Non-loopback HTTP is refused.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithManagementKey sets the Management API key used for credits and analytics.
func WithManagementKey(k string) Option {
	return func(c *Client) { c.managementKey = k }
}

// WithAPIKey sets the regular API key used for the /key endpoint.
func WithAPIKey(k string) Option {
	return func(c *Client) { c.apiKey = k }
}

// WithHTTPClient injects an HTTP client; tests use it for transports.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithVersion sets the User-Agent version component.
func WithVersion(v string) Option {
	return func(c *Client) { c.userAgent = "RouteBurn/" + v }
}

// WithSleep replaces the backoff sleeper so tests stay fast.
func WithSleep(f func(time.Duration)) Option {
	return func(c *Client) { c.sleep = f }
}

// WithClock replaces the clock used for rate-limit deadlines.
func WithClock(f func() time.Time) Option {
	return func(c *Client) { c.now = f }
}

// NewClient validates configuration and returns a ready client.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:   DefaultBaseURL,
		http:      &http.Client{Timeout: defaultTimeout},
		userAgent: "RouteBurn/dev",
		sleep:     time.Sleep,
		now:       time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("openrouter: invalid base URL: %w", err)
	}
	if u.Scheme != "https" && !isLoopback(u.Hostname()) {
		return nil, fmt.Errorf("openrouter: refusing non-TLS base URL %q", c.baseURL)
	}
	return c, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// NextRefreshAllowedAt is the earliest time the scheduler may refresh, derived
// from server rate-limit headers. The zero value means no restriction.
func (c *Client) NextRefreshAllowedAt() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nextRefreshAt
}

// CreditSummary is the credits endpoint decoded into currency values.
type CreditSummary struct {
	TotalPurchased float64
	TotalUsed      float64
	Remaining      float64
}

// Credits fetches account-wide credits with the management key.
func (c *Client) Credits(ctx context.Context) (CreditSummary, error) {
	if c.managementKey == "" {
		return CreditSummary{}, ErrMissingManagementKey
	}

	var payload struct {
		Data struct {
			TotalCredits float64 `json:"total_credits"`
			TotalUsage   float64 `json:"total_usage"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/credits", c.managementKey, nil, &payload); err != nil {
		return CreditSummary{}, err
	}

	return CreditSummary{
		TotalPurchased: payload.Data.TotalCredits,
		TotalUsed:      payload.Data.TotalUsage,
		Remaining:      payload.Data.TotalCredits - payload.Data.TotalUsage,
	}, nil
}

// KeyInfo is the /key endpoint, keeping nullable fields as pointers so
// "unlimited" never renders as zero.
type KeyInfo struct {
	Label          string
	Limit          *float64
	LimitRemaining *float64
	LimitReset     string
	UsageDaily     float64
	UsageWeekly    float64
	UsageMonthly   float64
}

// Key fetches the current regular API key's details.
func (c *Client) Key(ctx context.Context) (KeyInfo, error) {
	if c.apiKey == "" {
		return KeyInfo{}, ErrMissingAPIKey
	}

	var payload struct {
		Data struct {
			Label          string   `json:"label"`
			Limit          *float64 `json:"limit"`
			LimitRemaining *float64 `json:"limit_remaining"`
			LimitReset     *string  `json:"limit_reset"`
			UsageDaily     float64  `json:"usage_daily"`
			UsageWeekly    float64  `json:"usage_weekly"`
			UsageMonthly   float64  `json:"usage_monthly"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/key", c.apiKey, nil, &payload); err != nil {
		return KeyInfo{}, err
	}

	info := KeyInfo{
		Label:          payload.Data.Label,
		Limit:          payload.Data.Limit,
		LimitRemaining: payload.Data.LimitRemaining,
		UsageDaily:     payload.Data.UsageDaily,
		UsageWeekly:    payload.Data.UsageWeekly,
		UsageMonthly:   payload.Data.UsageMonthly,
	}
	if payload.Data.LimitReset != nil {
		info.LimitReset = *payload.Data.LimitReset
	}
	return info, nil
}

// AnalyticsMeta is the discovered shape of the analytics API.
type AnalyticsMeta struct {
	Metrics    []string `json:"metrics"`
	Dimensions []string `json:"dimensions"`
	Operators  []string `json:"operators"`
}

// AnalyticsMeta fetches supported metrics and dimensions so optional fields are
// discovered rather than guessed.
func (c *Client) AnalyticsMeta(ctx context.Context) (AnalyticsMeta, error) {
	if c.managementKey == "" {
		return AnalyticsMeta{}, ErrMissingManagementKey
	}
	// The envelope may be flat or wrapped in "data"; both are accepted.
	var payload struct {
		AnalyticsMeta
		Data *AnalyticsMeta `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/analytics/meta", c.managementKey, nil, &payload); err != nil {
		return AnalyticsMeta{}, err
	}
	if payload.Data != nil {
		return *payload.Data, nil
	}
	return payload.AnalyticsMeta, nil
}

// AnalyticsQuery is the documented POST body for /analytics/query.
type AnalyticsQuery struct {
	Metrics     []string
	Dimensions  []string
	Granularity string
	Start       time.Time
	End         time.Time
	Limit       int
}

func (q AnalyticsQuery) marshal() ([]byte, error) {
	body := map[string]any{
		"metrics": q.Metrics,
		"time_range": map[string]string{
			"start": q.Start.UTC().Format(time.RFC3339),
			"end":   q.End.UTC().Format(time.RFC3339),
		},
	}
	// An empty granularity is sent as omitted, not as "": the API rejects the
	// empty string but aggregates the whole range when the field is absent.
	if q.Granularity != "" {
		body["granularity"] = q.Granularity
	}
	if len(q.Dimensions) > 0 {
		body["dimensions"] = q.Dimensions
	}
	if q.Limit > 0 {
		body["limit"] = q.Limit
	}
	return json.Marshal(body)
}

// AnalyticsRow is one aggregation row mapped to domain values.
//
// Every field below total_usage/tokens_total is optional: the API only returns
// the metrics the query asked for, and older responses omit the newer ones. A
// zero value therefore means "not requested or not reported", never "zero
// measured" — callers that need to tell the difference check the matching
// *Set flag.
type AnalyticsRow struct {
	Day         time.Time
	Model       string
	CreditSpend float64
	Tokens      int64

	// RequestCount is the number of inference requests in the row.
	RequestCount int64
	// TokensPrompt and TokensCompletion split Tokens into input and output.
	TokensPrompt     int64
	TokensCompletion int64
	// CachedTokens is the prompt tokens served from the provider cache.
	CachedTokens int64
	// ReasoningTokens is the subset of completion tokens spent reasoning.
	ReasoningTokens int64
	// CacheHitRate is the provider-reported cache hit ratio in 0..1.
	CacheHitRate float64

	// HasCacheHitRate reports whether the row carried a cache hit rate, so a
	// genuine 0.0 is distinguishable from a missing metric.
	HasCacheHitRate bool
}

// QueryAnalytics runs an aggregation query.
func (c *Client) QueryAnalytics(ctx context.Context, q AnalyticsQuery) ([]AnalyticsRow, error) {
	if c.managementKey == "" {
		return nil, ErrMissingManagementKey
	}

	body, err := q.marshal()
	if err != nil {
		return nil, fmt.Errorf("openrouter: encode analytics query: %w", err)
	}

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/analytics/query", c.managementKey, body, &raw); err != nil {
		return nil, err
	}
	return decodeAnalyticsRows(raw)
}

// doJSON performs one request with bounded retries and typed errors.
func (c *Client) doJSON(ctx context.Context, method, endpoint, key string, body []byte, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		retryAfter, err := c.attempt(ctx, method, endpoint, key, body, out)
		if err == nil {
			return nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !retryable(err) || attempt == maxRetries {
			return err
		}

		delay := retryAfter
		if delay <= 0 {
			delay = backoff(attempt)
		}
		c.sleep(delay)
	}
	return lastErr
}

func (c *Client) attempt(ctx context.Context, method, endpoint, key string, body []byte, out any) (time.Duration, error) {
	var reader io.Reader
	if body != nil {
		reader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reader)
	if err != nil {
		return 0, &APIError{Endpoint: endpoint, Message: "could not build request", kind: ErrTransport}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, &APIError{Endpoint: endpoint, Message: "request failed", kind: ErrTransport}
	}
	defer resp.Body.Close()

	c.recordRateLimit(resp.Header)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return retryAfterDelay(resp.Header, c.now()), &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   endpoint,
			Message:    messageFor(resp.StatusCode),
			kind:       kindForStatus(resp.StatusCode),
		}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return 0, &APIError{Endpoint: endpoint, Message: "could not read response", kind: ErrTransport}
	}
	if len(data) > maxResponseBytes {
		return 0, &APIError{Endpoint: endpoint, Message: "response exceeded the size limit", kind: ErrResponseTooLarge}
	}

	if out == nil {
		return 0, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return 0, &APIError{Endpoint: endpoint, Message: "response was not valid JSON for this endpoint", kind: ErrMalformedResponse}
	}
	return 0, nil
}

func (c *Client) recordRateLimit(h http.Header) {
	reset := strings.TrimSpace(h.Get("X-RateLimit-Reset"))
	if reset == "" {
		return
	}
	t, ok := parseRateLimitReset(reset, c.now())
	if !ok {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if t.After(c.nextRefreshAt) {
		c.nextRefreshAt = t
	}
}

func parseRateLimitReset(v string, now time.Time) (time.Time, bool) {
	if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
		if secs > 1e12 {
			return time.Unix(secs/1000, 0), true
		}
		return time.Unix(secs, 0), true
	}
	if t, err := http.ParseTime(v); err == nil {
		return t, true
	}
	if d, err := time.ParseDuration(v); err == nil {
		return now.Add(d), true
	}
	return time.Time{}, false
}

func retryAfterDelay(h http.Header, now time.Time) time.Duration {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

func retryable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.kind == ErrTransport {
		return true
	}
	switch apiErr.StatusCode {
	case 429, 502, 503, 504:
		return true
	}
	return false
}

func backoff(attempt int) time.Duration {
	d := backoffBase << attempt
	if d > 4*time.Second {
		d = 4 * time.Second
	}
	return d/2 + time.Duration(rand.Int63n(int64(d/2)+1))
}

func messageFor(status int) string {
	switch status {
	case 401:
		return "credential is invalid or expired"
	case 403:
		return "credential lacks permission; use a management API key"
	case 404:
		return "endpoint unavailable; the API may have changed"
	case 429:
		return "rate limited; backing off"
	}
	if status >= 500 {
		return "OpenRouter reported a server error"
	}
	return "request rejected"
}
