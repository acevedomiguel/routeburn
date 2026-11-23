package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	if opts.HelpVisible {
		left = h.View(NewOverlayKeyMap())
	}

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

// HelpText renders the complete Help document without popup framing. It uses
// the same grouped key bindings as the dashboard footer and Help popup.
func HelpText(s Styles) string {
	return helpTextAt(s, 100, "dev")
}

func helpTextAt(s Styles, width int, version string) string {
	return s.Title.Render("RouteBurn help") + "\n\n" +
		helpContentAt(s, width, version) + "\n\n" +
		s.Muted.Render("Press Esc or ? to close.")
}

func helpContentAt(s Styles, width int, version string) string {
	if width <= 0 {
		width = 100
	}
	if version == "" {
		version = "dev"
	}

	h := NewHelp(s, width)
	h.ShowAll = true

	var b strings.Builder
	b.WriteString(s.Muted.Render("RouteBurn reads your OpenRouter credits and usage. It never sends inference requests."))
	b.WriteString("\n\n")
	b.WriteString(h.View(NewKeyMap()))
	b.WriteString("\n\n")
	b.WriteString(s.Title.Render("About RouteBurn"))
	b.WriteString("\n")
	b.WriteString("RouteBurn is an independent, read-only OpenRouter dashboard, available under the MIT license.\n")
	b.WriteString("Version: " + version + "\n")
	b.WriteString("Repository: https://github.com/acevedomiguel/routeburn\n\n")
	b.WriteString("Created and maintained by Miguel Angel Acevedo, a Cloud Architect and DevOps Engineer based in Hong Kong.\n")
	b.WriteString("Contact: https://acevedomiguel.com/contact")
	return b.String()
}

type helpLayout struct {
	width          int
	height         int
	popupWidth     int
	popupHeight    int
	viewportHeight int
	maxScroll      int
	lines          []string
}

func resolveHelpLayout(opts ViewOptions) helpLayout {
	width := opts.Width
	if width <= 0 {
		width = 80
	}
	height := opts.Height
	if height <= 0 {
		height = 40
	}

	popupWidth := minInt(84, width-4)
	if popupWidth < 4 {
		popupWidth = 4
	}
	textWidth := maxInt(popupWidth-4, 1)
	content := helpContentAt(opts.Styles, textWidth, opts.Version)
	wrapped := lipgloss.Wrap(content, textWidth, " ")
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, textWidth, "")
	}

	maxHeight := height - 2
	if maxHeight < 3 {
		maxHeight = 3
	}
	popupHeight := minInt(len(lines)+3, maxHeight)
	viewportHeight := maxInt(popupHeight-3, 1)
	return helpLayout{
		width:          width,
		height:         height,
		popupWidth:     popupWidth,
		popupHeight:    popupHeight,
		viewportHeight: viewportHeight,
		maxScroll:      maxInt(len(lines)-viewportHeight, 0),
		lines:          lines,
	}
}

// HelpScrollLimit reports the largest useful popup offset for this terminal.
func HelpScrollLimit(opts ViewOptions) int {
	return resolveHelpLayout(opts).maxScroll
}

// HelpOverlay centers a rounded, scrollable Help popup above an existing frame.
// The canvas keeps the original terminal dimensions, so the dashboard and its
// pinned footer continue to occupy the same rows while Help is open.
func HelpOverlay(background string, opts ViewOptions) string {
	layout := resolveHelpLayout(opts)
	offset := minInt(maxInt(opts.HelpScroll, 0), layout.maxScroll)
	end := minInt(offset+layout.viewportHeight, len(layout.lines))
	visible := append([]string(nil), layout.lines[offset:end]...)

	closeHint := "Esc/? close"
	if layout.maxScroll > 0 {
		closeHint = "Up/down scroll - Esc/? close"
	}
	visible = append(visible, opts.Styles.Muted.Render(closeHint))
	popup := strings.Join(panel("RouteBurn help", visible, layout.popupWidth, layout.popupHeight, opts.Styles), "\n")

	x := maxInt((layout.width-layout.popupWidth)/2, 0)
	y := maxInt((layout.height-layout.popupHeight)/2, 0)
	compositor := lipgloss.NewCompositor(
		lipgloss.NewLayer(background),
		lipgloss.NewLayer(popup).X(x).Y(y).Z(1),
	)
	return lipgloss.NewCanvas(layout.width, layout.height).Compose(compositor).Render()
}
