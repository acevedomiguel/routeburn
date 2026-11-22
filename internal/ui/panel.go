package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Box-drawing characters. The dashboard draws bordered panels like btop: every
// section is a box whose title is inlaid in the top border, so the frame does
// the grouping work and no section needs a background colour.
const (
	boxTopLeft     = "╭"
	boxTopRight    = "╮"
	boxBottomLeft  = "╰"
	boxBottomRight = "╯"
	boxHorizontal  = "─"
	boxVertical    = "│"
	boxDot         = "·"
)

// panel draws one bordered box exactly width columns wide and height rows tall,
// with title inlaid in the top border. rows are the body lines; they are padded
// and vertically padded to fill the box. Callers must size rows to the panel's
// text width (width-4); a row that still overflows is clipped only when it is
// plain text, so escape sequences are never cut in half.
func panel(title string, rows []string, width, height int, st Styles) []string {
	return panelWithCaption(title, "", rows, width, height, st)
}

// panelWithCaption draws a panel whose title sits in the top border and whose
// caption sits flush right in the same border, like btop's panel headers.
func panelWithCaption(title, caption string, rows []string, width, height int, st Styles) []string {
	if width < 4 {
		width = 4
	}
	if height < 2 {
		height = 2
	}
	inner := width - 2
	textWidth := inner - 2
	if textWidth < 1 {
		textWidth = 1
	}

	title = truncate(title, maxInt(inner-5, 1))
	head := boxHorizontal + "┤ " + st.CardHead.Render(title) + " ├"
	tail := ""
	if caption != "" {
		if room := inner - lipgloss.Width(head) - 5; room > 0 {
			tail = " " + st.Muted.Render(truncate(caption, room)) + " " + boxHorizontal
		}
	}
	fill := inner - lipgloss.Width(head) - lipgloss.Width(tail)
	if fill < 0 {
		fill = 0
	}

	out := make([]string, 0, height)
	out = append(out, boxTopLeft+st.Accent.Render(head)+strings.Repeat(boxHorizontal, fill)+st.Accent.Render(tail)+boxTopRight)

	for i := 0; i < height-2; i++ {
		row := ""
		if i < len(rows) {
			row = rows[i]
		}
		if lipgloss.Width(row) > textWidth {
			if !strings.ContainsRune(row, '\x1b') {
				row = truncate(row, textWidth)
			}
		}
		row = padRight(row, textWidth)
		out = append(out, st.Accent.Render(boxVertical)+" "+row+" "+st.Accent.Render(boxVertical))
	}

	out = append(out, boxBottomLeft+st.Accent.Render(strings.Repeat(boxHorizontal, inner))+boxBottomRight)
	return out
}

// leader renders "label ······ value" filling exactly width columns with the
// value flush right. It is the btop-style row that replaced the stacked
// label-above-value cards. Both inputs are plain text so the geometry is exact.
func leader(label, value string, width int, st Styles) string {
	return leaderValue(label, value, width, st, st.Value)
}

// leaderValue is leader with an explicit style for the value, used for warning
// rows such as a low-confidence runway.
func leaderValue(label, value string, width int, st Styles, valueStyle lipgloss.Style) string {
	label = truncate(label, maxInt(width, 1))
	value = truncate(value, maxInt(width, 1))
	lw, vw := lipgloss.Width(label), lipgloss.Width(value)
	dots := width - lw - vw - 2
	if dots < 1 {
		return st.Muted.Render(truncate(label+" "+value, width))
	}
	return st.Label.Render(label) + " " + st.Muted.Render(strings.Repeat(boxDot, dots)) + " " + valueStyle.Render(value)
}

// meter renders a proportional bar of width cells: filled blocks for the ratio,
// dim blocks for the remainder. It sits inline with the number it describes.
func meter(ratio float64, width int, st Styles) string {
	if width < 1 {
		width = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	fill := int(math.Round(ratio * float64(width)))
	if fill < 0 {
		fill = 0
	}
	if fill > width {
		fill = width
	}
	return st.Bar.Render(strings.Repeat("▇", fill)) + st.Muted.Render(strings.Repeat("░", width-fill))
}

// joinColumns places rendered blocks side by side, one space apart, padding
// shorter blocks with blank lines so every row lines up.
func joinColumns(blocks ...[]string) []string {
	height := 0
	widths := make([]int, len(blocks))
	for i, b := range blocks {
		if len(b) > height {
			height = len(b)
		}
		if len(b) > 0 {
			widths[i] = lipgloss.Width(b[0])
		}
	}

	out := make([]string, 0, height)
	for row := 0; row < height; row++ {
		var b strings.Builder
		for i, blk := range blocks {
			if i > 0 {
				b.WriteString(" ")
			}
			line := ""
			if row < len(blk) {
				line = blk[row]
			}
			b.WriteString(padRight(line, widths[i]))
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}
	return out
}

// frameTop draws the outer frame's top border with the wordmark inlaid and the
// freshness status flush right.
func frameTop(title, right string, width int, st Styles) string {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	head := boxHorizontal + "┤ " + st.CardHead.Render(title) + " ├"
	tail := boxHorizontal
	if right != "" {
		tail = " " + st.Muted.Render(right) + " " + boxHorizontal
	}
	fill := inner - lipgloss.Width(head) - lipgloss.Width(tail)
	if fill < 1 {
		tail = boxHorizontal
		fill = inner - lipgloss.Width(head) - 1
	}
	if fill < 0 {
		fill = 0
	}
	return boxTopLeft + st.Muted.Render(head) + strings.Repeat(boxHorizontal, fill) + st.Muted.Render(tail) + boxTopRight
}

// frameBottom draws the outer frame's bottom border.
func frameBottom(width int, st Styles) string {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	return boxBottomLeft + st.Muted.Render(strings.Repeat(boxHorizontal, inner)) + boxBottomRight
}

// frameRow wraps one body line in the outer frame's side borders.
func frameRow(content string, width int, st Styles) string {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	if lipgloss.Width(content) > inner {
		if !strings.ContainsRune(content, '\x1b') {
			content = truncate(content, inner)
		}
	}
	content = padRight(content, inner)
	return st.Muted.Render(boxVertical) + content + st.Muted.Render(boxVertical)
}

// fitLinesSlice pads or trims the body so the frame and footer always land on
// the expected rows.
func fitLinesSlice(lines []string, height int) []string {
	if height < 1 {
		height = 1
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}
