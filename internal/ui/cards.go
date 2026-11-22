package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/acevedomiguel/routeburn/internal/domain"
)

// ViewOptions carries everything the renderer needs that is not in the
// snapshot, keeping rendering pure and deterministic.
type ViewOptions struct {
	Width       int
	Height      int
	Styles      Styles
	Now         time.Time
	Cached      bool
	CacheAge    time.Duration
	Banner      string
	HelpVisible bool
	Scroll      int
	Page        int
	Pages       int
	Refreshing  bool
	Spinner     string

	// TableOffset is the first visible day when the table has focus. It is
	// ignored while the body owns the scroll keys.
	TableOffset  int
	TableFocused bool
}

// Render draws the full dashboard. It never performs IO.
//
// The layout fills the alternate screen exactly: the panels, and the bubbles
// help bar pinned to the last row, so the bar never floats mid-screen. There
// is no outer frame: the panels are the frame, and the app name is not
// repeated inside the app. The freshness status lives in the Credits panel's
// border caption, not on a floating line.
func Render(snap *domain.Snapshot, opts ViewOptions) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	lay := Classify(opts.Width, opts.Height)
	if lay.Class == ClassTooSmall {
		return opts.Styles.Warning.Render(lay.Message) + "\n"
	}

	// The footer occupies the last row. A banner is an extra row above the
	// body, so it comes out of the body's budget rather than pushing the
	// footer off the bottom of the screen.
	bannerRows := 0
	if opts.Banner != "" {
		bannerRows = 1
	}
	bodyHeight := opts.Height - 1 - bannerRows
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	lines := dashboardBody(snap, opts, lay, bodyHeight)

	// Scrolling: when the composed body is taller than the space, the viewport
	// bubble takes over and the footer gains pagination dots.
	if len(lines) > bodyHeight {
		maxOffset := len(lines) - bodyHeight
		opts.Pages = (maxOffset + bodyHeight - 1) / bodyHeight
		if opts.Pages < 2 {
			opts.Pages = 2
		}
		offset := minInt(opts.Scroll, maxOffset)
		offset = (offset / bodyHeight) * bodyHeight
		opts.Page = minInt(offset/bodyHeight, opts.Pages-1)
		lines = lines[offset : offset+bodyHeight]
	} else {
		lines = fitLinesSlice(lines, bodyHeight)
	}

	header := ""
	if opts.Banner != "" {
		header = opts.Styles.Banner.Render(truncate(opts.Banner, lay.Width)) + "\n"
	}

	footer := Footer(opts)
	return header + strings.Join(lines, "\n") + "\n" + footer
}

// dashboardBody composes the interior of the frame: the credit panels, the
// daily table, and the model panel. It returns exactly height rows when the
// content fits, stretching the last panel to absorb the slack.
func dashboardBody(snap *domain.Snapshot, opts ViewOptions, lay Layout, height int) []string {
	inner := lay.Width

	// Wide terminals get a sidebar: the small panels stack down the left and
	// the daily table takes a full-height right column, so a 200-column run no
	// longer hands the table the whole middle of the screen. Below the height
	// floor the stacked arrangement fits more panels, so it keeps the short
	// terminal.
	if lay.Class == ClassWide && lay.TwoColumn && height >= minSidebarHeight {
		return sidebarBody(snap, opts, inner, height)
	}

	// Vertical budget: two panel borders per panel, plus the gap row between
	// panels. The token split only appears once the terminal can afford it: it
	// costs four extra rows in the top block, which the minimum 60x16 cannot pay.
	detail := height >= 26

	rows := make([]string, 0, height)

	// Top row: credits and last-30 panels side by side. Both are three leader
	// rows tall. Stacking them costs six rows, which at the 60x16 minimum leaves
	// no room for the model breakdown, so the two panels always share a row.
	if lay.TwoColumn {
		half := (inner - 1) / 2
		leftWidth := half
		rightWidth := inner - 1 - leftWidth
		// The credits panel carries the freshness caption; widen it when the
		// caption would truncate, as long as the last-30 panel stays readable.
		if status := headerStatus(snap, opts); status != "" {
			if need := lipgloss.Width(status) + 19; need > leftWidth && inner-1-need >= minLast30Width {
				leftWidth, rightWidth = need, inner-1-need
			}
		}
		rightRows := last30PanelWidth(snap, opts, rightWidth)
		// The credits panel is padded to the right panel's height so both boxes
		// end on the same row. Without it the shorter column leaves a notch that
		// reads as a layout bug rather than as deliberate whitespace.
		leftRows := creditsPanelWidth(snap, opts, leftWidth, len(rightRows))
		rows = append(rows, joinColumns(leftRows, rightRows)...)
	} else {
		rows = append(rows, creditsPanelWidth(snap, opts, inner, 0)...)
		rows = append(rows, "")
		rows = append(rows, last30PanelWidth(snap, opts, inner)...)
	}

	// The model breakdown and the token split share a row of their own, between
	// the credit panels and the sparkline: model on the left, token split on the
	// right. Side by side they cost the height of the taller panel instead of the
	// sum of both, which is what keeps the model list and the sparkline on a
	// 26-row terminal. The split has no place on a short terminal, so there the
	// stacked arrangement below runs instead and the model panel keeps the whole
	// width.
	if lay.TwoColumn && detail {
		leftWidth := (inner - 1) / 2
		rightWidth := inner - 1 - leftWidth
		remaining := height - len(rows)

		// The model panel sizes to its content and the token split is fixed, so
		// the shared row is as tall as whichever needs more. The model count is
		// capped so the sparkline always keeps its minimum below.
		tokens := tokenPanelHeight(snap, opts, rightWidth, 0)
		modelCount := modelPanelCount(snap, remaining-minTableHeight-4, leftWidth-4)
		rowHeight := modelCount + 2
		if len(tokens) > rowHeight {
			rowHeight = len(tokens)
		}

		// Both panels are padded to the row height so their bottom borders end on
		// the same row, matching the credits/last-30 pattern. A split that cannot
		// fit beside the sparkline's minimum is left to the stacked arrangement
		// below rather than clipped.
		if len(tokens) > 0 && rowHeight <= remaining-2-minTableHeight {
			rows = append(rows, "")
			leftRows := modelPanelWidth(snap, opts, leftWidth, modelCount, rowHeight)
			rightRows := tokenPanelHeight(snap, opts, rightWidth, rowHeight)
			rows = append(rows, joinColumns(leftRows, rightRows)...)
			rows = append(rows, "")

			tableHeight := fitTableHeight(height-len(rows), dayCount(snap))
			rows = append(rows, dailyTablePanel(snap, opts, inner, tableHeight)...)
			return rows
		}
	}

	// The token split is a full-width panel of its own rather than extra rows in
	// the last-30 panel. Stacking it inside one column would leave the other
	// column short and punch a hole in the layout; as its own row both columns
	// end together.
	if detail {
		if tokens := tokenPanel(snap, opts, inner); len(tokens) > 0 {
			rows = append(rows, "")
			rows = append(rows, tokens...)
		}
	}

	rows = append(rows, "")

	// The model panel sizes to its content: its rows are the models plus the two
	// borders. Everything left over goes to the sparkline, so a tall terminal
	// gets a taller chart rather than a panel full of blank rows.
	//
	// A short terminal cannot fit both. Rather than clip a panel in half and
	// break the frame, the model panel is dropped and the sparkline takes the
	// whole remainder. minModelHeight is the smallest model panel worth drawing:
	// one row plus its two borders.
	remaining := height - len(rows)
	if remaining < minTableHeight+1+minModelHeight {
		tableHeight := fitTableHeight(remaining, dayCount(snap))
		rows = append(rows, dailyTablePanel(snap, opts, inner, tableHeight)...)
		return rows
	}

	modelCount := modelPanelCount(snap, remaining-minTableHeight-1-2, inner-4)
	tableHeight := fitTableHeight(remaining-1-(modelCount+2), dayCount(snap))
	rows = append(rows, dailyTablePanel(snap, opts, inner, tableHeight)...)

	rows = append(rows, "")
	rows = append(rows, modelPanel(snap, opts, inner, modelCount)...)
	return rows
}

// minModelHeight is the smallest model panel worth drawing: one row plus its
// two borders.
const minModelHeight = 3

// minLast30Width is the narrowest last-30 panel that still reads: a leader
// row, its dots, and a value.
const minLast30Width = 28

// minSidebarHeight is the shortest body that can hold the sidebar: credits,
// last-30, and the token split stacked need most of it before the model panel
// or the table get anything useful.
const minSidebarHeight = 20

// minTableWidth is the narrowest right-hand column worth giving the daily
// table: below it the table drops so many columns the sidebar stops paying
// for itself.
const minTableWidth = 44

// sidebarBody is the wide-terminal arrangement: the small panels stack down a
// left column and the daily table takes a full-height right column. The model
// panel is last on the left and absorbs the leftover rows, so both columns
// end on the same row.
func sidebarBody(snap *domain.Snapshot, opts ViewOptions, inner, height int) []string {
	// The left column defaults to half, but widens when the freshness caption
	// would otherwise truncate: the caption needs its own width plus the title
	// head, the caption slack panelWithCaption reserves, and the two borders.
	leftWidth := (inner - 1) / 2
	if status := headerStatus(snap, opts); status != "" {
		if need := lipgloss.Width(status) + 19; need > leftWidth && inner-1-need >= minTableWidth {
			leftWidth = need
		}
	}
	rightWidth := inner - 1 - leftWidth

	left := creditsPanelWidth(snap, opts, leftWidth, 0)
	left = append(left, "")
	left = append(left, last30PanelWidth(snap, opts, leftWidth)...)
	if tokens := tokenPanelHeight(snap, opts, leftWidth, 0); len(tokens) > 0 {
		left = append(left, "")
		left = append(left, tokens...)
	}
	if remaining := height - len(left) - 1; remaining >= minModelHeight {
		left = append(left, "")
		left = append(left, modelPanel(snap, opts, leftWidth, modelPanelCount(snap, remaining-2, leftWidth-4))...)
	}

	right := dailyTablePanel(snap, opts, rightWidth, height)
	return joinColumns(left, right)
}

// minTableHeight lives in dailytable.go: a top border, a column header, one
// day, a separator, and the total row.

// minModelNameWidth is the narrowest model name worth rendering. Shorter than
// this and the row is a column of ellipses that tells the reader nothing.
const minModelNameWidth = 8

// minModelTailWidth is the room the fixed trailing columns need: the spend and
// the percentage, plus their separating spaces.
const minModelTailWidth = 12

// modelPanelCount is how many model rows fit in the space left over after the
// sparkline takes its minimum. It never returns zero: one row is always shown so
// the panel never degrades into an empty box.
//
// textWidth is the panel's text width. On the shared row the model panel is
// half the content width, so a column too narrow to show a name beside the
// spend collapses to the single top spender instead of stacking ellipses.
func modelPanelCount(snap *domain.Snapshot, available, textWidth int) int {
	n := 1
	if snap != nil && len(snap.Models) > 0 {
		n = len(domain.TopModels(domain.SortModels(snap.Models), 5))
	}
	if textWidth > 0 && textWidth-minModelTailWidth < minModelNameWidth {
		return 1
	}
	if available < 1 {
		return 1
	}
	if n > available {
		n = available
	}
	return n
}

// creditsPanelWidth renders the credits panel at an explicit width. A height of
// zero sizes the panel to its content; a positive height pads it with blank rows
// so it can sit flush beside a taller neighbour. The freshness status rides in
// the border's caption slot, so the dashboard needs no floating status line.
func creditsPanelWidth(snap *domain.Snapshot, opts ViewOptions, width, height int) []string {
	t := opts.Styles
	caption := statusCaption(snap, opts, width)
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	if snap == nil {
		return panelWithCaption("Credits", caption, []string{t.Muted.Render("loading")}, width, 3, t)
	}
	acct := snap.Account
	rw := snap.Runway

	runway := domain.FormatDaysRemaining(rw)
	if d := domain.FormatDepletionDate(rw); d != "" {
		runway += " " + d
	}
	runwayStyle := t.Value
	if rw.LowConfidence {
		runwayStyle = t.Warning
	}

	rows := []string{
		leader("Remaining", domain.FormatCredits(acct.Remaining), inner, t),
		leaderValue("Runway", runway, inner, t, runwayStyle),
	}
	if rw.Basis == domain.BasisUnavailable {
		rows = append(rows, leader("Daily burn", "analytics unavailable", inner, t))
	} else if rw.DailyBurn > 0 {
		rows = append(rows, leader("Daily burn", domain.FormatCredits(rw.DailyBurn)+"/day", inner, t))
	}
	if height <= 0 {
		return panelWithCaption("Credits", caption, rows, width, len(rows)+2, t)
	}
	// Pad with empty rows rather than inventing data: the panel keeps its exact
	// height contract and the neighbouring box's bottom edge lines up.
	for len(rows) < height-2 {
		rows = append(rows, "")
	}
	if len(rows) > height-2 {
		rows = rows[:height-2]
	}
	return panelWithCaption("Credits", caption, rows, width, height, t)
}

// last30PanelWidth renders the 30-day summary panel at an explicit width.
func last30PanelWidth(snap *domain.Snapshot, opts ViewOptions, width int) []string {
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	if snap == nil {
		return panel("Last 30 days", []string{t.Muted.Render("loading")}, width, 3, t)
	}
	spend := 0.0
	for _, d := range snap.Daily {
		spend += d.CreditSpend
	}
	rows := []string{
		leader("Spend", domain.FormatCredits(spend), inner, t),
	}
	// A zero request count means the API did not report it, so the row is
	// omitted rather than shown as a misleading "0".
	if snap.Requests > 0 {
		rows = append(rows, leader("Requests", domain.FormatCount(snap.Requests), inner, t))
	}
	rows = append(rows, leader("Tokens", domain.FormatTokens(snap.TotalTokens), inner, t))
	if snap.CacheHitRate != nil {
		rows = append(rows, leader("Cache hit", fmt.Sprintf("%.1f%%", *snap.CacheHitRate), inner, t))
	}
	return panel("Last 30 days", rows, width, len(rows)+2, t)
}

// tokenPair is one label/value row of the token split.
type tokenPair struct{ label, value string }

// tokenPairs is the prompt/completion/cached/reasoning split in render order.
// Every value comes from the same analytics query the dashboard already runs, so
// the extra detail costs no additional request and no additional API key. It is
// empty when the API reported no split, so the panel never appears as a box of
// zeros.
func tokenPairs(snap *domain.Snapshot) []tokenPair {
	if snap == nil {
		return nil
	}
	pairs := make([]tokenPair, 0, 4)
	if snap.TokensPrompt > 0 {
		pairs = append(pairs, tokenPair{"Input", domain.FormatTokens(snap.TokensPrompt)})
	}
	if snap.TokensCompletion > 0 {
		pairs = append(pairs, tokenPair{"Output", domain.FormatTokens(snap.TokensCompletion)})
	}
	if snap.CachedTokens > 0 {
		pairs = append(pairs, tokenPair{"Cached", domain.FormatTokens(snap.CachedTokens)})
	}
	if snap.ReasoningTokens > 0 {
		pairs = append(pairs, tokenPair{"Reasoning", domain.FormatTokens(snap.ReasoningTokens)})
	}
	return pairs
}

// tokenPairsFitTwoUp reports whether every pair fits in a column of the given
// width with a dot leader: a label, at least one dot, and the value. When any
// pair does not fit, the two-up arrangement would clip its value, so the split
// renders one pair per row instead.
func tokenPairsFitTwoUp(pairs []tokenPair, colWidth int) bool {
	for _, p := range pairs {
		if lipgloss.Width(p.label)+lipgloss.Width(p.value)+3 > colWidth {
			return false
		}
	}
	return true
}

// tokenPanel renders the split as a full-width panel sized to its content.
func tokenPanel(snap *domain.Snapshot, opts ViewOptions, width int) []string {
	return tokenPanelHeight(snap, opts, width, 0)
}

// tokenPanelHeight renders the split as a panel, two leader pairs per row. A
// height of zero sizes the panel to its content; a positive height pads it with
// blank rows so it can sit flush beside a taller neighbour, exactly like the
// credits panel. It returns nil when there is no split to show.
func tokenPanelHeight(snap *domain.Snapshot, opts ViewOptions, width, height int) []string {
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	pairs := tokenPairs(snap)
	if len(pairs) == 0 {
		return nil
	}

	// Two pairs per row keeps the panel short, but only when the column is wide
	// enough to hold a label and its value side by side. In the narrow half-width
	// column of the shared row it often is not: a two-up row would run past the
	// panel and have its value clipped, so the split falls back to one pair per
	// row, where every value the API reported stays readable.
	colWidth := (inner - 2) / 2
	perRow := 2
	if colWidth < 12 || !tokenPairsFitTwoUp(pairs, colWidth) {
		perRow, colWidth = 1, inner
	}
	rows := make([]string, 0, (len(pairs)+perRow-1)/perRow)
	for i := 0; i < len(pairs); i += perRow {
		if perRow == 2 && i+1 < len(pairs) {
			left := leader(pairs[i].label, pairs[i].value, colWidth, t)
			right := leader(pairs[i+1].label, pairs[i+1].value, colWidth, t)
			rows = append(rows, left+"  "+right)
		} else {
			rows = append(rows, leader(pairs[i].label, pairs[i].value, inner, t))
		}
	}
	if height <= 0 {
		return panel("Tokens", rows, width, len(rows)+2, t)
	}
	for len(rows) < height-2 {
		rows = append(rows, "")
	}
	if len(rows) > height-2 {
		rows = rows[:height-2]
	}
	return panel("Tokens", rows, width, height, t)
}

// dailyTablePanel renders the daily spend table as a bordered panel. The
// offset is clamped here so a stale scroll from the app can never ask for a
// window past the end of the series.
func dailyTablePanel(snap *domain.Snapshot, opts ViewOptions, width, height int) []string {
	var daily []domain.DailyUsage
	if snap != nil {
		daily = snap.Daily
	}
	return DailyTable(daily, DailyTableOptions{
		Width:   width,
		Height:  height,
		Offset:  opts.TableOffset,
		Styles:  opts.Styles,
		Focused: opts.TableFocused,
	})
}

// dayCount is the number of days the snapshot carries, or 0 when there is no
// snapshot yet.
func dayCount(snap *domain.Snapshot) int {
	if snap == nil {
		return 0
	}
	return len(snap.Daily)
}

// TableWindow is how many day rows the table shows at the current view size.
// The app needs it to page the offset by a screen at a time.
//
// It counts the rows out of a real render rather than recomputing the panel's
// share of the height: the election in dashboardBody depends on the series
// length, the model count, and the layout branch, and a second copy of that
// arithmetic would silently disagree with what is on screen. Rendering here is
// pure string work, and this runs once per keypress.
func TableWindow(snap *domain.Snapshot, opts ViewOptions) int {
	opts.TableOffset = 0
	lines := strings.Split(Render(snap, opts), "\n")
	for i, line := range lines {
		if !strings.Contains(line, boxTopLeft+boxHorizontal+boxTick+" Daily spend") {
			continue
		}
		// Rows after the header run until the separator rule that sits above
		// the pinned total.
		count := 0
		for _, row := range lines[i+2:] {
			if strings.Contains(row, strings.Repeat(boxHorizontal, 8)) {
				return maxInt(count, 1)
			}
			if strings.Contains(row, boxBottomLeft) {
				return maxInt(count, 1)
			}
			count++
		}
	}
	return 1
}

// neededTableHeight is the tallest the panel can usefully be: a border, a
// header, every day, a separator, a total row, and the closing border.
func neededTableHeight(days int) int { return days + 5 }

// fitTableHeight clamps the panel between its minimum and what the series can
// fill. It absorbs slack only while there are days to show, so a tall terminal
// shows more data rather than a panel full of blank rows; any surplus is left
// to Render's tail padding.
func fitTableHeight(available, days int) int {
	return minInt(maxInt(available, minTableHeight), maxInt(neededTableHeight(days), minTableHeight))
}

// modelPanel renders the model spend breakdown as a bordered panel sized to its
// rows. Rows are short names, a dot leader, and an inline meter with the share
// at the row end.
func modelPanel(snap *domain.Snapshot, opts ViewOptions, width, limit int) []string {
	return modelPanelWidth(snap, opts, width, limit, 0)
}

// modelPanelWidth renders the model spend breakdown at an explicit width. A
// height of zero sizes the panel to its rows; a positive height pads it with
// blank rows so it can sit flush beside a taller neighbour on the shared row.
// The width is honoured here, so on a half-width column a long model name is
// truncated by modelRow rather than overflowing into the token panel.
func modelPanelWidth(snap *domain.Snapshot, opts ViewOptions, width, limit, height int) []string {
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	if snap == nil || len(snap.Models) == 0 {
		msg := []string{t.Muted.Render("No model spend recorded.")}
		if height <= 0 {
			return panel("Cost by model", msg, width, 3, t)
		}
		for len(msg) < height-2 {
			msg = append(msg, "")
		}
		return panel("Cost by model", msg[:maxInt(height-2, 1)], width, height, t)
	}
	if limit < 1 {
		limit = 1
	}

	// A single row goes to the top spender. TopModels would fold everything into
	// "Other (remaining)" at limit 1, which hides the biggest model behind a
	// useless 100% row.
	sorted := domain.SortModels(snap.Models)
	models := sorted
	if len(sorted) > limit {
		if limit == 1 {
			models = sorted[:1]
		} else {
			models = domain.TopModels(sorted, limit)
		}
	}
	total := 0.0
	for _, m := range models {
		total += m.CreditSpend
	}

	rows := make([]string, 0, len(models))
	for _, m := range models {
		rows = append(rows, modelRow(m, total, inner, t))
	}
	if height <= 0 {
		return panel("Cost by model", rows, width, len(rows)+2, t)
	}
	for len(rows) < height-2 {
		rows = append(rows, "")
	}
	if len(rows) > height-2 {
		rows = rows[:height-2]
	}
	return panel("Cost by model", rows, width, height, t)
}

// modelRow renders one model as "name ······· $2.61 ▇▇▇▇ 52%".
func modelRow(m domain.ModelUsage, total float64, width int, t Styles) string {
	name := domain.ShortModelName(m.Model)
	spend := domain.FormatCredits(m.CreditSpend)
	ratio := 0.0
	if total > 0 {
		ratio = m.CreditSpend / total
	}
	pct := fmt.Sprintf("%.0f%%", ratio*100)

	// Fixed trailing columns: percent, meter, spend.
	meterWidth := 10
	if width < 40 {
		meterWidth = 6
	}
	if width < 30 {
		meterWidth = 4
	}
	tail := spend + " " + meter(ratio, meterWidth, t) + " " + pct
	tailWidth := lipgloss.Width(tail)
	if tailWidth+12 > width {
		// Too narrow for the meter: keep the number and the share.
		tail = spend + " " + pct
		tailWidth = lipgloss.Width(tail)
	}

	nameWidth := width - tailWidth - 3
	if nameWidth < 6 {
		nameWidth = 6
	}
	name = truncate(name, nameWidth)
	dots := width - lipgloss.Width(name) - tailWidth - 2
	if dots < 1 {
		return t.Label.Render(truncate(name+" "+spend, width))
	}
	return t.Label.Render(name) + " " + t.Muted.Render(strings.Repeat(boxDot, dots)) + " " + tail
}

// headerStatus renders the freshness text shown in the frame's top border.
//
// The snapshot stores GeneratedAt in UTC so the JSON contract and the cache
// stay timezone-independent, but the dashboard is read by a person: the clock
// is rendered in the machine's local timezone so "Updated 17:42" matches the
// wall clock next to the terminal.
func headerStatus(snap *domain.Snapshot, opts ViewOptions) string {
	when := ""
	if snap != nil && !snap.GeneratedAt.IsZero() {
		when = snap.GeneratedAt.In(time.Local).Format("15:04")
	}
	switch {
	case opts.Cached:
		return CachedLabel(opts.CacheAge, true)
	case opts.Refreshing:
		frame := opts.Spinner
		if frame == "" {
			frame = "·"
		}
		return frame + " Refreshing…"
	case when != "":
		return "Updated " + when
	}
	return ""
}

// statusCaption picks the freshness text for a panel border of the given
// width: the full headerStatus string when it fits the caption slot, the
// compact "Cached 2h 14m ago" form when only that fits. The slot's room is
// the panel width minus the title head and the slack panelWithCaption
// reserves; anything longer is left to the panel's own truncation.
func statusCaption(snap *domain.Snapshot, opts ViewOptions, width int) string {
	full := headerStatus(snap, opts)
	if lipgloss.Width(full) <= width-19 {
		return full
	}
	if opts.Cached {
		if compact := "Cached " + FormatAge(opts.CacheAge); lipgloss.Width(compact) <= width-19 {
			return compact
		}
	}
	return full
}

// twoColumns places two blocks side by side, padding both to half the width.
func twoColumns(left, right string, width int) string {
	leftWidth := (width - 2) / 2
	if leftWidth < 10 {
		leftWidth = 10
	}
	rightWidth := width - leftWidth - 2
	if rightWidth < 10 {
		rightWidth = 10
	}

	l := strings.Split(left, "\n")
	r := strings.Split(right, "\n")
	n := maxInt(len(l), len(r))

	var b strings.Builder
	for i := 0; i < n; i++ {
		var lv, rv string
		if i < len(l) {
			lv = l[i]
		}
		if i < len(r) {
			rv = r[i]
		}
		line := padRight(truncate(lv, leftWidth), leftWidth) + "  " + truncate(rv, rightWidth)
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func scroll(body string, offset, height int) string {
	lines := strings.Split(body, "\n")
	if offset < 0 {
		offset = 0
	}
	if offset > len(lines) {
		offset = len(lines)
	}
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	if end <= offset {
		return ""
	}
	return strings.Join(lines[offset:end], "\n")
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// truncate cuts plain text to width, appending an ellipsis when it is cut. It
// is escape-aware: it measures the rendered width. The result is never wider
// than width, so callers can rely on it to keep the frame's geometry exact.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if width <= 3 {
		return string(runes[:width])
	}
	if len(runes) > width-3 {
		runes = runes[:width-3]
	}
	return string(runes) + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
