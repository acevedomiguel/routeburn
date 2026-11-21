package credentials

import (
	"regexp"
	"strings"
)

// redactionMarker is what replaces anything that looks like a secret.
const redactionMarker = "[redacted]"

// secretPattern matches OpenRouter key shapes, Authorization headers, and
// token query parameters so log and error lines can be scrubbed.
var secretPattern = regexp.MustCompile(`(?i)(sk-or-v1-[A-Za-z0-9_-]+|bearer\s+[A-Za-z0-9._-]+|token=[A-Za-z0-9._-]+)`)

// Redact removes anything that looks like a credential from s.
func Redact(s string) string {
	if s == "" {
		return s
	}
	return secretPattern.ReplaceAllString(s, redactionMarker)
}

// RedactAll applies Redact to every value.
func RedactAll(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = Redact(v)
	}
	return out
}

// LooksLikeSecret reports whether s contains anything Redact would remove.
func LooksLikeSecret(s string) bool {
	return secretPattern.MatchString(s)
}

// bearerPrefix mirrors the header scheme so tests can assert case matching.
var bearerPrefix = strings.ToLower("Bearer ")

// IsBearer reports whether a header value uses the bearer scheme.
func IsBearer(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), bearerPrefix)
}
