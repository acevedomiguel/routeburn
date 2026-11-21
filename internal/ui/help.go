package ui

import (
	"strings"
)

// Binding describes one key hint in the footer and the help overlay.
type Binding struct {
	Keys   []string
	Action string
}

// Bindings is the complete set of advertised keys. Only implemented keys are
// listed.
func Bindings() []Binding {
	return []Binding{
		{Keys: []string{"r"}, Action: "refresh"},
		{Keys: []string{"j/k", "up/down"}, Action: "scroll"},
		{Keys: []string{"?"}, Action: "help"},
		{Keys: []string{"q"}, Action: "quit"},
	}
}

// Footer renders the one-line key hints.
func Footer(opts ViewOptions) string {
	var parts []string
	for _, b := range Bindings() {
		key := b.Keys[0]
		if len(key) > 2 {
			key = key[:2]
		}
		parts = append(parts, opts.Styles.Key.Render("["+key+"]")+opts.Styles.Help.Render(b.Action))
	}
	return strings.Join(parts, "  ")
}

// HelpText is the full-screen help content.
func HelpText(s Styles) string {
	var b strings.Builder
	b.WriteString(s.Title.Render("RouteBurn help"))
	b.WriteString("\n\n")
	b.WriteString(s.Muted.Render("RouteBurn reads your OpenRouter credits and usage. It never sends inference requests."))
	b.WriteString("\n\n")
	for _, bind := range Bindings() {
		b.WriteString("  ")
		b.WriteString(s.Key.Render(strings.Join(bind.Keys, ", ")))
		b.WriteString(strings.Repeat(" ", maxInt(1, 16-len(strings.Join(bind.Keys, ", ")))))
		b.WriteString(bind.Action)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(s.Muted.Render("Press Esc or ? to close."))
	return b.String()
}
