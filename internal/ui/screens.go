package ui

import (
	"strings"
)

// SetupOptions configures the first-run setup screen.
type SetupOptions struct {
	Width      int
	Height     int
	Styles     Styles
	Banner     string
	KeyMasked  string
	KeyLen     int
	Validating bool
	Notice     string
}

// SetupView renders the first-run credential panel. It never shows the
// entered secret, only its length.
func SetupView(opts SetupOptions) string {
	t := opts.Styles
	width := opts.Width
	if width < 40 {
		width = 40
	}
	if width > 78 {
		width = 78
	}

	var b strings.Builder
	b.WriteString(t.Title.Render("RouteBurn setup"))
	b.WriteString("\n\n")
	b.WriteString(t.Muted.Render("RouteBurn is a read-only dashboard. It never sends inference requests."))
	b.WriteString("\n\n")
	b.WriteString("A Management API key is required for account balance and analytics.\n")
	b.WriteString(t.Muted.Render("Create one at https://openrouter.ai/settings/management_keys"))
	b.WriteString("\n\n")

	if opts.Banner != "" {
		b.WriteString(t.Error.Render(opts.Banner))
		b.WriteString("\n\n")
	}

	label := "Management key: "
	masked := strings.Repeat("*", opts.KeyLen)
	if opts.KeyLen == 0 {
		masked = t.Muted.Render("(not entered)")
	}
	b.WriteString(label + masked)
	b.WriteString("\n")

	if opts.Validating {
		b.WriteString(t.Accent.Render("Validating..."))
		b.WriteString("\n")
	}
	if opts.Notice != "" {
		b.WriteString(t.Warning.Render(opts.Notice))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(t.Help.Render("Enter to continue   Esc to quit   Ctrl+C to quit"))

	return b.String()
}

// FatalOptions configures the fatal-error screen.
type FatalOptions struct {
	Width  int
	Height int
	Styles Styles
	Reason string
}

// FatalView renders a fatal error with a next action.
func FatalView(opts FatalOptions) string {
	t := opts.Styles
	var b strings.Builder
	b.WriteString(t.Error.Render("RouteBurn cannot start"))
	b.WriteString("\n\n")
	reason := opts.Reason
	if reason == "" {
		reason = "No usable data was available."
	}
	b.WriteString(reason)
	b.WriteString("\n\n")
	b.WriteString(t.Muted.Render("Check your connection and credentials, then press r to retry or q to quit."))
	return b.String()
}
