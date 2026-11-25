package routeburn

import (
	"net/http"
	"time"

	"github.com/acevedomiguel/routeburn/internal/openrouter"
)

// OpenRouter client and transport vocabulary.
type (
	Client         = openrouter.Client
	Option         = openrouter.Option
	CreditSummary  = openrouter.CreditSummary
	KeyInfo        = openrouter.KeyInfo
	AnalyticsQuery = openrouter.AnalyticsQuery
	AnalyticsRow   = openrouter.AnalyticsRow
	APIError       = openrouter.APIError
)

// DefaultBaseURL is the production OpenRouter API endpoint.
const DefaultBaseURL = openrouter.DefaultBaseURL

// Sentinel errors allow callers to classify client failures with errors.Is.
var (
	ErrBadRequest           = openrouter.ErrBadRequest
	ErrUnauthorized         = openrouter.ErrUnauthorized
	ErrForbidden            = openrouter.ErrForbidden
	ErrNotFound             = openrouter.ErrNotFound
	ErrRateLimited          = openrouter.ErrRateLimited
	ErrServer               = openrouter.ErrServer
	ErrTransport            = openrouter.ErrTransport
	ErrMalformedResponse    = openrouter.ErrMalformedResponse
	ErrResponseTooLarge     = openrouter.ErrResponseTooLarge
	ErrMissingManagementKey = openrouter.ErrMissingManagementKey
	ErrMissingAPIKey        = openrouter.ErrMissingAPIKey
)

// NewClient validates configuration and returns a read-only OpenRouter client.
func NewClient(opts ...Option) (*Client, error) {
	return openrouter.NewClient(opts...)
}

// WithBaseURL overrides the API base URL. Non-loopback HTTP is refused.
func WithBaseURL(url string) Option {
	return openrouter.WithBaseURL(url)
}

// WithManagementKey sets the key used for credits and analytics.
func WithManagementKey(key string) Option {
	return openrouter.WithManagementKey(key)
}

// WithAPIKey sets the regular key used for the key endpoint.
func WithAPIKey(key string) Option {
	return openrouter.WithAPIKey(key)
}

// WithHTTPClient injects an HTTP client and its transport.
func WithHTTPClient(client *http.Client) Option {
	return openrouter.WithHTTPClient(client)
}

// WithVersion sets the User-Agent version component.
func WithVersion(version string) Option {
	return openrouter.WithVersion(version)
}

// WithSleep replaces the backoff sleeper.
func WithSleep(sleep func(time.Duration)) Option {
	return openrouter.WithSleep(sleep)
}

// WithClock replaces the clock used for rate-limit deadlines.
func WithClock(now func() time.Time) Option {
	return openrouter.WithClock(now)
}
