package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/charmbracelet/lipgloss"
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
}

// Render draws the full dashboard. It never performs IO.
//
// The layout fills the alternate screen exactly: a status row, the panels, and
// the bubbles help bar pinned to the last row, so the bar never floats
// mid-screen. There is no outer frame: the panels are the frame, and the app
// name is not repeated inside the app.
func Render(snap *domain.Snapshot, opts ViewOptions) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	lay := Classify(opts.Width, opts.Height)
	if lay.Class == ClassTooSmall {
		return opts.Styles.Warning.Render(lay.Message) + "\n"
	}

	// The status row occupies the top line and the footer the last one. A banner
	// is an extra row above the status, so it comes out of the body's budget
	// rather than pushing the footer off the bottom of the screen.
	bannerRows := 0
	if opts.Banner != "" {
		bannerRows = 1
	}
	bodyHeight := opts.Height - 2 - bannerRows
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

	status := StatusLine(snap, opts, lay.Width)
	footer := Footer(opts)
	return header + status + "\n" + strings.Join(lines, "\n") + "\n" + footer
}

// StatusLine renders the top row: just the freshness status, flush left. It
// replaces the outer frame's top border and deliberately carries no app name,
// since the app name inside the app is noise.
func StatusLine(snap *domain.Snapshot, opts ViewOptions, width int) string {
	status := headerStatus(snap, opts)
	if status == "" {
		return ""
	}
	return padRight(opts.Styles.Muted.Render(truncate(status, width)), width)
}

// dashboardBody composes the interior of the frame: the credit panels, the
// sparkline, and the model panel. It returns exactly height rows when the
// content fits, stretching the last panel to absorb the slack.
func dashboardBody(snap *domain.Snapshot, opts ViewOptions, lay Layout, height int) []string {
	inner := lay.Width

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
	if remaining < minSparkHeight+1+minModelHeight {
		rows = append(rows, sparkPanel(snap, opts, inner, maxInt(remaining, minSparkHeight))...)
		return rows
	}

	modelCount := modelPanelCount(snap, remaining-minSparkHeight-1-2)
	sparkHeight := remaining - 1 - (modelCount + 2)
	if sparkHeight < minSparkHeight {
		sparkHeight = minSparkHeight
	}
	rows = append(rows, sparkPanel(snap, opts, inner, sparkHeight)...)

	rows = append(rows, "")
	rows = append(rows, modelPanel(snap, opts, inner, modelCount)...)
	return rows
}

// minModelHeight is the smallest model panel worth drawing: one row plus its
// two borders.
const minModelHeight = 3

// minSparkHeight is the smallest usable sparkline panel: a top border, one row
// of blocks, and a bottom border.
const minSparkHeight = 3

// modelPanelCount is how many model rows fit in the space left over after the
// sparkline takes its minimum. It never returns zero: one row is always shown so
// the panel never degrades into an empty box.
func modelPanelCount(snap *domain.Snapshot, available int) int {
	n := 1
	if snap != nil && len(snap.Models) > 0 {
		n = len(domain.TopModels(domain.SortModels(snap.Models), 5))
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
// so it can sit flush beside a taller neighbour.
func creditsPanelWidth(snap *domain.Snapshot, opts ViewOptions, width, height int) []string {
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	if snap == nil {
		return panel("Credits", []string{t.Muted.Render("loading")}, width, 3, t)
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
		return panel("Credits", rows, width, len(rows)+2, t)
	}
	// Pad with empty rows rather than inventing data: the panel keeps its exact
	// height contract and the neighbouring box's bottom edge lines up.
	for len(rows) < height-2 {
		rows = append(rows, "")
	}
	if len(rows) > height-2 {
		rows = rows[:height-2]
	}
	return panel("Credits", rows, width, height, t)
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

// tokenPanel renders the prompt/completion/cached/reasoning split as a
// full-width panel, two leader pairs per row. Every value comes from the same
// analytics query the dashboard already runs, so the extra detail costs no
// additional request and no additional API key. It returns nil when the API
// reported no split, so the panel never appears as a box of zeros.
func tokenPanel(snap *domain.Snapshot, opts ViewOptions, width int) []string {
	if snap == nil {
		return nil
	}
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}

	type pair struct{ label, value string }
	pairs := make([]pair, 0, 4)
	if snap.TokensPrompt > 0 {
		pairs = append(pairs, pair{"Input", domain.FormatTokens(snap.TokensPrompt)})
	}
	if snap.TokensCompletion > 0 {
		pairs = append(pairs, pair{"Output", domain.FormatTokens(snap.TokensCompletion)})
	}
	if snap.CachedTokens > 0 {
		pairs = append(pairs, pair{"Cached", domain.FormatTokens(snap.CachedTokens)})
	}
	if snap.ReasoningTokens > 0 {
		pairs = append(pairs, pair{"Reasoning", domain.FormatTokens(snap.ReasoningTokens)})
	}
	if len(pairs) == 0 {
		return nil
	}

	// Two pairs per row keeps the panel short: the split costs two rows plus its
	// borders instead of four, leaving room for the chart and the model panel.
	colWidth := (inner - 2) / 2
	if colWidth < 12 {
		colWidth = inner
	}
	rows := make([]string, 0, (len(pairs)+1)/2)
	for i := 0; i < len(pairs); i += 2 {
		left := leader(pairs[i].label, pairs[i].value, colWidth, t)
		if i+1 < len(pairs) {
			right := leader(pairs[i+1].label, pairs[i+1].value, colWidth, t)
			rows = append(rows, left+"  "+right)
		} else {
			rows = append(rows, left)
		}
	}
	return panel("Tokens", rows, width, len(rows)+2, t)
}

// sparkPanel renders the daily spend sparkline as a bordered panel.
func sparkPanel(snap *domain.Snapshot, opts ViewOptions, width, height int) []string {
	var daily []domain.DailyUsage
	if snap != nil {
		daily = snap.Daily
	}
	rendered := Sparkline(daily, ChartOptions{Width: width, Height: height, Styles: opts.Styles})
	lines := strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
	return lines
}

// modelPanel renders the model spend breakdown as a bordered panel sized to its
// rows. Rows are short names, a dot leader, and an inline meter with the share
// at the row end.
func modelPanel(snap *domain.Snapshot, opts ViewOptions, width, limit int) []string {
	t := opts.Styles
	inner := width - 4
	if inner < 1 {
		inner = 1
	}
	if snap == nil || len(snap.Models) == 0 {
		return panel("Cost by model", []string{t.Muted.Render("No model spend recorded.")}, width, 3, t)
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
	return panel("Cost by model", rows, width, len(rows)+2, t)
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

	nameWidth := width - tailWidth - 2
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
