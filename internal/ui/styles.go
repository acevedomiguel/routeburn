package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette colors, chosen to stay legible on both dark and light terminals.
// lipgloss v2 turned Color into a function returning color.Color, so these
// are variables, not constants.
var (
	colorAccent  color.Color = lipgloss.Color("6")
	colorMuted   color.Color = lipgloss.Color("8")
	colorSuccess color.Color = lipgloss.Color("2")
	colorWarning color.Color = lipgloss.Color("3")
	colorError   color.Color = lipgloss.Color("1")
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

	StatusBar lipgloss.Style
}

// NewStyles builds the style set. When enabled is false every style renders
// plain text so --no-color and NO_COLOR produce identical output.
func NewStyles(enabled bool) Styles {
	build := func(fg color.Color, bold bool) lipgloss.Style {
		if !enabled {
			return lipgloss.NewStyle()
		}
		s := lipgloss.NewStyle()
		if fg != nil {
			s = s.Foreground(fg)
		}
		if bold {
			s = s.Bold(true)
		}
		return s
	}

	return Styles{
		Enabled:   enabled,
		Base:      build(nil, false),
		Title:     build(colorAccent, true),
		Muted:     build(colorMuted, false),
		Accent:    build(colorAccent, false),
		Success:   build(colorSuccess, false),
		Warning:   build(colorWarning, false),
		Error:     build(colorError, false),
		Card:      build(nil, false),
		CardHead:  build(colorAccent, true),
		Label:     build(colorMuted, false),
		Value:     build(nil, true),
		Banner:    build(colorError, false),
		Help:      build(colorMuted, false),
		Key:       build(colorAccent, false),
		Bar:       build(colorSuccess, false),
		StatusBar: statusBarStyle(enabled),
	}
}

// WithWidth sets a width on a style when color is enabled so padding stays
// consistent regardless of escape sequences.
// statusBarStyle is the full-width bottom bar background.
func statusBarStyle(enabled bool) lipgloss.Style {
	if !enabled {
		return lipgloss.NewStyle().Padding(0, 1)
	}
	return lipgloss.NewStyle().Background(lipgloss.Color("236")).Padding(0, 1)
}
