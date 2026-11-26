package app

import (
	"github.com/acevedomiguel/burnblocks/openrouter"
)

// credRedact scrubs anything that looks like a credential from a message.
func credRedact(s string) string {
	return openrouter.Redact(s)
}
