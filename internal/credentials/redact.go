package credentials

import (
	"net/url"
	"regexp"
	"sort"
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

// Redactor scrubs a set of known secrets in addition to the generic secret
// shapes. A key that does not match the documented OpenRouter prefix is still
// hidden, and the value is matched in its URL-escaped forms too.
type Redactor struct {
	secrets []string
}

// NewRedactor returns a redactor aware of the given secrets. Empty values are
// ignored, so an unset credential never becomes a match-all pattern, and the
// longest secret is matched first so overlapping values are fully removed.
func NewRedactor(secrets ...string) *Redactor {
	r := &Redactor{}
	seen := map[string]bool{}
	for _, s := range secrets {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		for _, form := range []string{s, url.QueryEscape(s), url.PathEscape(s)} {
			if form == "" || seen[form] {
				continue
			}
			seen[form] = true
			r.secrets = append(r.secrets, form)
		}
	}
	sort.Slice(r.secrets, func(i, j int) bool { return len(r.secrets[i]) > len(r.secrets[j]) })
	return r
}

// Redact scrubs the receiver's secrets and the generic secret shapes from s. A
// nil receiver still scrubs the generic shapes, so a missing configuration
// never disables redaction.
func (r *Redactor) Redact(s string) string {
	if s == "" {
		return s
	}
	if r != nil {
		for _, secret := range r.secrets {
			s = strings.ReplaceAll(s, secret, redactionMarker)
		}
	}
	return Redact(s)
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
