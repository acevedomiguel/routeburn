package openrouter

import (
	"errors"
	"fmt"
)

// Sentinel errors let callers classify failures without parsing messages.
var (
	ErrBadRequest           = errors.New("openrouter: bad request")
	ErrUnauthorized         = errors.New("openrouter: credential is invalid or missing")
	ErrForbidden            = errors.New("openrouter: credential lacks permission for this endpoint")
	ErrNotFound             = errors.New("openrouter: endpoint not found")
	ErrRateLimited          = errors.New("openrouter: rate limited")
	ErrServer               = errors.New("openrouter: server error")
	ErrTransport            = errors.New("openrouter: transport failure")
	ErrMalformedResponse    = errors.New("openrouter: malformed response")
	ErrResponseTooLarge     = errors.New("openrouter: response too large")
	ErrMissingManagementKey = errors.New("openrouter: a management API key is required")
	ErrMissingAPIKey        = errors.New("openrouter: a regular API key is required")
)

// APIError is a typed HTTP failure. It carries the endpoint and status so the
// UI can explain what failed, and never carries credentials or response bodies.
type APIError struct {
	StatusCode int
	Endpoint   string
	Message    string
	kind       error
}

func (e *APIError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("openrouter: %s: %s", e.Endpoint, e.Message)
	}
	return fmt.Sprintf("openrouter: %s: %s (%d %s)", e.Endpoint, e.Message, e.StatusCode, httpStatusText(e.StatusCode))
}

// Unwrap makes errors.Is(apiErr, ErrRateLimited) work.
func (e *APIError) Unwrap() error { return e.kind }

// RateLimited reports whether the failure means a later retry may succeed.
func (e *APIError) RateLimited() bool { return e.StatusCode == 429 }

func kindForStatus(status int) error {
	switch {
	case status == 400:
		return ErrBadRequest
	case status == 401:
		return ErrUnauthorized
	case status == 403:
		return ErrForbidden
	case status == 404:
		return ErrNotFound
	case status == 429:
		return ErrRateLimited
	case status >= 500:
		return ErrServer
	default:
		return ErrTransport
	}
}

func httpStatusText(status int) string {
	if text := httpStatusTexts[status]; text != "" {
		return text
	}
	return "HTTP error"
}

var httpStatusTexts = map[int]string{
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	429: "Too Many Requests",
	500: "Internal Server Error",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
}
