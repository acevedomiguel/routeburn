package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// helpStyles adapts the terminal palette to the bubbles help bubble.
func helpStyles(s Styles) help.Styles {
	if !s.Enabled {
		empty := lipgloss.NewStyle()
		return help.Styles{
			Ellipsis:       empty,
			ShortKey:       empty,
			ShortDesc:      empty,
			ShortSeparator: empty,
			FullKey:        empty,
			FullDesc:       empty,
			FullSeparator:  empty,
		}
	}
	return help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(colorMuted),
		ShortKey:       lipgloss.NewStyle().Foreground(colorAccent),
		ShortDesc:      lipgloss.NewStyle().Foreground(colorMuted),
		ShortSeparator: lipgloss.NewStyle().Foreground(colorMuted),
		FullKey:        lipgloss.NewStyle().Foreground(colorAccent),
		FullDesc:       lipgloss.NewStyle().Foreground(colorMuted),
		FullSeparator:  lipgloss.NewStyle().Foreground(colorMuted),
	}
}

// NewHelp builds the bubbles help bubble used for the bottom bar and overlay.
func NewHelp(s Styles, width int) help.Model {
	h := help.New()
	h.Width = width
	h.ShortSeparator = "  "
	h.FullSeparator = "   "
	h.Styles = helpStyles(s)
	return h
}

// NewViewport builds the scrolling body bubble. It is always initialised so
// scroll offset and clamping are handled by bubbles, not by hand.
func NewViewport(width, height int) viewport.Model {
	v := viewport.New(width, height)
	v.MouseWheelEnabled = true
	return v
}

// NewPaginator builds the bubbles paginator used for the scroll dots in the
// bottom bar, mirroring the bubbles list example.
func NewPaginator(pages, page int) paginator.Model {
	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = "●"
	p.InactiveDot = "○"
	p.PerPage = 1
	p.TotalPages = pages
	p.Page = page
	return p
}

// NewTable builds the bubbles table used for the model breakdown. The table
// bubble owns column alignment, truncation, and header styling.
func NewTable(columns []table.Column, rows []table.Row, s Styles) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	styles := table.DefaultStyles()
	styles.Header = s.CardHead
	styles.Cell = s.Base
	styles.Selected = s.Base
	t.SetStyles(styles)

	width := 0
	for _, c := range columns {
		width += c.Width
	}
	t.SetWidth(width)
	t.SetHeight(len(rows) + 1)
	return t
}

// NewSpinner builds the refresh spinner shown in the header.
func NewSpinner(s Styles) spinner.Model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	if s.Enabled {
		sp.Style = lipgloss.NewStyle().Foreground(colorAccent)
	}
	return sp
}
