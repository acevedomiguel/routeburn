package app

import (
	"github.com/acevedomiguel/routeburn/internal/credentials"
)

// credRedact scrubs anything that looks like a credential from a message.
func credRedact(s string) string {
	return credentials.Redact(s)
}
