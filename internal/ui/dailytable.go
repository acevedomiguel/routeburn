package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// minTableHeight is the smallest daily table panel worth drawing: two borders,
// a column header, one day, a separator, and the pinned total row.
const minTableHeight = 6

// DailyTableOptions configures the daily spend table.
type DailyTableOptions struct {
	Width   int
	Height  int
	Offset  int
	Styles  Styles
	Focused bool
}

// dailyColumn is one column of the daily table. Width is the content width;
// the table style adds one padding column on each side.
type dailyColumn struct {
	title string
	width int
	right bool
}

// DailyTable renders the daily credit spend as a bordered table panel, newest
// day first, with a pinned total row. It returns exactly opts.Height lines
// whenever opts.Height is at least minTableHeight.
//
// The frame is drawn by panelWithCaption; bubbles/table supplies the header,
// the cell rendering, and the row windowing, so the panel's own border and the
// app's scroll plumbing stay in charge of the geometry.
func DailyTable(daily []domain.DailyUsage, opts DailyTableOptions) []string {
	height := maxInt(opts.Height, minTableHeight)
	width := maxInt(opts.Width, 24)
	st := opts.Styles

	if len(daily) == 0 {
		return panel("Daily spend", []string{st.Muted.Render("No daily data available.")}, width, height, st)
	}

	cols := dailyColumns(daily, width)

	// Newest day first: the series runs oldest to newest, the table reads the
	// other way, the same direction as the sparkline it replaces. Cells are cut
	// to the visible columns so a dropped column cannot overrun the row.
	rows := make([][]string, 0, len(daily))
	for i := len(daily) - 1; i >= 0; i-- {
		rows = append(rows, dayCells(daily[i])[:len(cols)])
	}

	dataRows := tableDayRows(height)
	maxOffset := maxInt(len(rows)-dataRows, 0)
	offset := minInt(maxInt(opts.Offset, 0), maxOffset)

	tbl := table.New(
		table.WithColumns(colSpecs(cols)),
		table.WithRows(windowRows(rows, cols, offset, dataRows)),
		// The header costs one of the requested lines, so ask for one more
		// than the data rows we want.
		table.WithHeight(dataRows+1),
		table.WithWidth(width-4),
		table.WithFocused(false),
		table.WithStyles(tblStyles(st)),
	)
	// The bubble derives its window from the cursor, so park the cursor on the
	// last visible row: SetCursor(offset+dataRows-1) yields exactly
	// [offset, offset+dataRows).
	tbl.SetCursor(len(tbl.Rows()) - 1)

	body := strings.Split(tbl.View(), "\n")
	rule := strings.Repeat("─", minInt(colBudget(cols), width-4))
	body = append(body, st.Muted.Render(rule))
	body = append(body, totalRow(daily, cols, st))

	caption := ""
	if len(rows) > dataRows {
		caption = fmt.Sprintf("%d-%d/%d", offset+1, offset+dataRows, len(rows))
	}
	return panelWithCaption("Daily spend", caption, body, width, height, st)
}

// dayCells renders one day's values in column order.
func dayCells(d domain.DailyUsage) []string {
	return []string{
		d.Day.Format("Jan 02"),
		domain.FormatCredits(d.CreditSpend),
		domain.FormatTokens(d.Tokens),
		domain.FormatCount(d.Requests),
		formatCacheRate(d.CachedTokens, d.TokensPrompt),
	}
}

// formatCacheRate derives a day's cache hit rate. A day that reported no prompt
// tokens reads "-", never "0.0%", because a zero in this API means "not
// reported" rather than "measured zero".
func formatCacheRate(cached, prompt int64) string {
	if prompt <= 0 {
		return "-"
	}
	rate := float64(cached) / float64(prompt)
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	return fmt.Sprintf("%.1f%%", rate*100)
}

// dailyColumns picks the visible columns and their widths. Dropped columns are
// the ones the API never reported, so a fresh account does not get three
// columns of zeros. Widths plus the two padding columns per cell fill a body
// row exactly: sum(width) + 2*len(cols) == panelWidth-4.
func dailyColumns(daily []domain.DailyUsage, panelWidth int) []dailyColumn {
	hasTokens, hasReqs, hasCache := false, false, false
	for _, d := range daily {
		if d.Tokens > 0 {
			hasTokens = true
		}
		if d.Requests > 0 {
			hasReqs = true
		}
		if d.TokensPrompt > 0 {
			hasCache = true
		}
	}

	cols := []dailyColumn{{title: "DATE", width: 11}}
	if len(daily) > 0 {
		cols = append(cols, dailyColumn{title: "SPEND", width: 7, right: true})
	}
	if hasTokens {
		cols = append(cols, dailyColumn{title: "TOKENS", width: 7, right: true})
	}
	if hasReqs {
		cols = append(cols, dailyColumn{title: "REQS", width: 6, right: true})
	}
	if hasCache {
		cols = append(cols, dailyColumn{title: "CACHE", width: 6, right: true})
	}

	// Drop optional columns from the right until the row fits, then shave the
	// widest column by one at a time. DATE keeps at least 6 so "Jan 02"
	// survives; everything else keeps at least its header.
	for len(cols) > 1 && colBudget(cols) > panelWidth-4 {
		cols = cols[:len(cols)-1]
	}
	for colBudget(cols) > panelWidth-4 && widest(cols) > 5 {
		i := widestIndex(cols)
		cols[i].width--
	}
	// Surplus is spread over the columns rather than dropped in one place: the
	// numbers are right-aligned inside their cells, so widening every column
	// pushes each figure to its own right edge and the table reads as a single
	// block instead of a cluster on the left with a margin to the right.
	// All surplus goes to DATE. The bubble left-aligns every cell, so a wide
	// numeric column would strand its figure mid-row; a wide date column pushes
	// the numbers together against the right edge, which is how the rest of the
	// dashboard (leader rows with dot leaders) already reads.
	cols[0].width += (panelWidth - 4) - colBudget(cols)
	return cols
}

// colBudget is the row width the columns occupy, padding included.
func colBudget(cols []dailyColumn) int {
	sum := 0
	for _, c := range cols {
		sum += c.width + 2
	}
	return sum
}

func widest(cols []dailyColumn) int {
	w := 0
	for _, c := range cols {
		w = maxInt(w, c.width)
	}
	return w
}

func widestIndex(cols []dailyColumn) int {
	idx, w := 0, -1
	for i, c := range cols {
		if c.width > w {
			idx, w = i, c.width
		}
	}
	return idx
}

// colSpecs maps our columns onto the bubble's.
func colSpecs(cols []dailyColumn) []table.Column {
	specs := make([]table.Column, 0, len(cols))
	for _, c := range cols {
		// The bubble left-aligns every title, so a right-aligned column gets one
		// padded here. Without it the header sits at the far side of its numbers.
		title := c.title
		if c.right {
			title = padCell(title, c.width, true)
		}
		specs = append(specs, table.Column{Title: title, Width: c.width})
	}
	return specs
}

// windowRows slices the rows the table should hold. The table is given only the
// visible window so its own cursor arithmetic cannot drift from ours.
func windowRows(rows [][]string, cols []dailyColumn, offset, dataRows int) []table.Row {
	end := minInt(offset+dataRows, len(rows))
	window := make([]table.Row, 0, end-offset)
	for _, r := range rows[offset:end] {
		// Cells are padded here, not by the bubble: it renders each value
		// left-aligned inside a fixed width, so right-aligned figures have to
		// arrive already padded to their column.
		row := make(table.Row, len(r))
		for i, cell := range r {
			row[i] = cell
			if i < len(cols) && cols[i].right {
				row[i] = padCell(cell, cols[i].width, true)
			}
		}
		window = append(window, row)
	}
	return window
}

// tblStyles gives the bubble the palette without its default padding on the
// selected row: padded selection renders the cursor row two columns left of
// the others, which breaks column alignment.
func tblStyles(st Styles) table.Styles {
	// Padding is geometry, not colour, so it stays under --no-color: the cells
	// need their two columns either way for the frame width to add up.
	return table.Styles{
		Header:   st.CardHead.Padding(0, 1),
		Cell:     st.Base.Padding(0, 1),
		Selected: st.Base,
	}
}

// totalRow pins the all-days totals under the visible window, so scrolling
// never hides the number the panel exists to show.
func totalRow(daily []domain.DailyUsage, cols []dailyColumn, st Styles) string {
	var spend float64
	var tokens, reqs, cached, prompt int64
	for _, d := range daily {
		spend += d.CreditSpend
		tokens += d.Tokens
		reqs += d.Requests
		cached += d.CachedTokens
		prompt += d.TokensPrompt
	}
	cells := []string{
		fmt.Sprintf("%d days", len(daily)),
		domain.FormatCredits(spend),
		domain.FormatTokens(tokens),
		domain.FormatCount(reqs),
		formatCacheRate(cached, prompt),
	}
	if len(cols) < len(cells) {
		cells = cells[:len(cols)]
	}
	parts := make([]string, 0, len(cols))
	for i, c := range cols {
		parts = append(parts, " "+padCell(cells[i], c.width, c.right)+" ")
	}
	return st.Value.Render(strings.Join(parts, ""))
}

// padCell pads a cell to its column width. Right-aligned columns are padded on
// the left, which keeps every number on a shared decimal edge.
func padCell(s string, width int, right bool) string {
	if w := lipglossWidth(s); w > width {
		s = truncate(s, width)
	}
	pad := width - lipglossWidth(s)
	if pad <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", pad) + s
	}
	return s + strings.Repeat(" ", pad)
}

// tableDayRows is the number of day rows a panel of this height holds: two
// borders, the column header, the separator, and the total row take the rest.
func tableDayRows(height int) int {
	return maxInt(height-5, 1)
}
