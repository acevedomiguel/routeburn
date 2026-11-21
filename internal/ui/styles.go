package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette colors, chosen to stay legible on both dark and light terminals.
const (
	colorAccent  = lipgloss.Color("6")
	colorMuted   = lipgloss.Color("8")
	colorSuccess = lipgloss.Color("2")
	colorWarning = lipgloss.Color("3")
	colorError   = lipgloss.Color("1")
)

// Styles bundles every Lip Gloss style used by the dashboard.
type Styles struct {
	Enabled bool

	Base     lipgloss.Style
	Title    lipgloss.Style
	Muted    lipgloss.Style
	Accent   lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Card     lipgloss.Style
	CardHead lipgloss.Style
	Label    lipgloss.Style
	Value    lipgloss.Style
	Banner   lipgloss.Style
	Help     lipgloss.Style
	Key      lipgloss.Style
	Bar      lipgloss.Style
}

// NewStyles builds the style set. When enabled is false every style renders
// plain text so --no-color and NO_COLOR produce identical output.
func NewStyles(enabled bool) Styles {
	build := func(fg lipgloss.Color, bold bool) lipgloss.Style {
		if !enabled {
			return lipgloss.NewStyle()
		}
		s := lipgloss.NewStyle()
		if fg != "" {
			s = s.Foreground(fg)
		}
		if bold {
			s = s.Bold(true)
		}
		return s
	}

	return Styles{
		Enabled:  enabled,
		Base:     build("", false),
		Title:    build(colorAccent, true),
		Muted:    build(colorMuted, false),
		Accent:   build(colorAccent, false),
		Success:  build(colorSuccess, false),
		Warning:  build(colorWarning, false),
		Error:    build(colorError, false),
		Card:     build("", false),
		CardHead: build(colorAccent, true),
		Label:    build(colorMuted, false),
		Value:    build("", true),
		Banner:   build(colorError, false),
		Help:     build(colorMuted, false),
		Key:      build(colorAccent, false),
		Bar:      build(colorSuccess, false),
	}
}

// WithWidth sets a width on a style when color is enabled so padding stays
// consistent regardless of escape sequences.
func (s Styles) WithWidth(style lipgloss.Style, width int) lipgloss.Style {
	return style.Width(width)
}
