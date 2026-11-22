package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Binding describes one key hint. It is derived from KeyMap so the footer, the
// overlay, and the handlers cannot drift apart.
type Binding struct {
	Keys   []string
	Hint   string
	Action string
}

// Footer renders the bottom menu bar. It is a real bubbles composition: the
// help bubble supplies the key hints and the paginator bubble supplies the
// scroll position, both inset in a full-width status bar so the bar sits flush
// along the bottom edge of the screen.
func Footer(opts ViewOptions) string {
	width := opts.Width
	if width <= 0 {
		width = 80
	}

	right := ""
	if opts.Pages > 1 {
		right = NewPaginator(opts.Pages, opts.Page).View()
	}

	// Give the help bubble the room the dots need, one column for the gap and
	// two for the status bar padding.
	helpWidth := width - 2
	if right != "" {
		helpWidth -= lipgloss.Width(right) + 1
	}
	h := NewHelp(opts.Styles, helpWidth)
	h.ShowAll = false
	left := h.View(NewKeyMap())

	bar := left
	if right != "" {
		gap := width - 2 - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
		bar = left + strings.Repeat(" ", gap) + right
	}
	if bar == "" {
		bar = " "
	}

	out := opts.Styles.StatusBar.Render(bar)
	if got := lipgloss.Width(out); got < width {
		// Pad so the bar reaches the right edge on every terminal.
		out += strings.Repeat(" ", width-got)
	}
	return out
}

// HelpBar renders the full grouped help, still inside the bubbles bubble.
func HelpBar(opts ViewOptions) string {
	width := opts.Width
	if width <= 0 {
		width = 80
	}
	h := NewHelp(opts.Styles, width)
	h.ShowAll = true
	return h.View(NewKeyMap())
}

// HelpText is the full-screen help overlay. It renders the bubbles help bubble
// in its full (grouped) mode so the overlay and the bottom bar share one source
// of truth.
func HelpText(s Styles) string {
	return HelpTextAt(s, 100)
}

// HelpTextAt renders the overlay for a given terminal width.
func HelpTextAt(s Styles, width int) string {
	if width <= 0 {
		width = 100
	}

	h := NewHelp(s, width)
	h.ShowAll = true

	var b strings.Builder
	b.WriteString(s.Title.Render("RouteBurn help"))
	b.WriteString("\n\n")
	b.WriteString(s.Muted.Render("RouteBurn reads your OpenRouter credits and usage. It never sends inference requests."))
	b.WriteString("\n\n")
	b.WriteString(h.View(NewKeyMap()))
	b.WriteString("\n\n")
	b.WriteString(s.Muted.Render("Press Esc or ? to close."))
	return b.String()
}
