package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/charmbracelet/bubbles/table"
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
// The layout fills the alternate screen exactly: one header row, a scrolling
// body, and the bubbles help bar pinned to the last row, so the bar never
// floats in the middle of a tall terminal.
func Render(snap *domain.Snapshot, opts ViewOptions) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	lay := Classify(opts.Width, opts.Height)
	if lay.Class == ClassTooSmall {
		return opts.Styles.Warning.Render(lay.Message) + "\n"
	}

	var b strings.Builder
	if opts.Banner != "" {
		b.WriteString(opts.Styles.Banner.Render(truncate(opts.Banner, lay.ContentWidth)))
		b.WriteString("\n\n")
	}

	cards := []string{
		creditsCard(snap, opts, lay),
		runwayCard(snap, opts, lay),
		last30Card(snap, opts, lay),
	}

	if lay.TwoColumn {
		left := cards[0] + "\n" + cards[1]
		b.WriteString(twoColumns(left, cards[2], lay.ContentWidth))
	} else {
		for i, c := range cards {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(c)
		}
	}

	b.WriteString("\n\n")
	if snap != nil {
		b.WriteString(Sparkline(snap.Daily, ChartOptions{Width: lay.ContentWidth, Styles: opts.Styles}))
	}
	b.WriteString(modelsCard(snap, opts, lay))

	header := headerLine(snap, opts, lay)
	footer := Footer(opts)

	// Two rows are reserved: the header and the help bar. Everything else
	// scrolls inside the viewport bubble.
	bodyHeight := opts.Height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := b.String()
	lines := len(strings.Split(body, "\n"))
	maxOffset := maxInt(lines-bodyHeight, 0)

	// The viewport bubble takes over whenever the content is taller than the
	// space between the header and the menu bar, regardless of width class.
	if maxOffset > 0 {
		pageSize := maxInt(bodyHeight, 1)
		opts.Pages = (maxOffset + pageSize - 1) / pageSize
		if opts.Pages < 2 {
			opts.Pages = 2
		}

		vp := NewViewport(lay.ContentWidth, bodyHeight)
		vp.SetContent(body)
		offset := minInt(opts.Scroll, maxOffset)
		vp.SetYOffset(offset)
		body = vp.View()
		opts.Page = minInt(offset/pageSize, opts.Pages-1)

		// Recompute the menu bar now that the dot count is known.
		footer = Footer(opts)
	}
	return header + "\n" + fitLines(body, bodyHeight) + "\n" + footer
}

// fitLines pads or trims the body so the header and footer always land on the
// expected rows.
func fitLines(body string, height int) string {
	lines := strings.Split(body, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// headerLine renders the single top row: the wordmark on the left and the
// freshness status right-aligned.
func headerLine(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	title := t.Title.Render("RouteBurn")
	status := headerStatus(snap, opts)
	if strings.TrimSpace(status) == "" {
		return title
	}

	detail := t.Muted.Render(status)
	gap := lay.Width - lipgloss.Width(title) - lipgloss.Width(status) - 2*lay.Padding
	if gap < 1 {
		return truncate(title+" "+status, lay.Width)
	}
	line := strings.Repeat(" ", lay.Padding) + title + strings.Repeat(" ", gap) + detail
	return truncate(line, lay.Width)
}

func headerStatus(snap *domain.Snapshot, opts ViewOptions) string {
	when := ""
	if snap != nil && !snap.GeneratedAt.IsZero() {
		when = snap.GeneratedAt.Format("15:04")
	}
	switch {
	case opts.Cached:
		return CachedLabel(opts.CacheAge, true)
	case opts.Refreshing:
		frame := opts.Spinner
		if frame == "" {
			frame = ""
		}
		return frame + " Refreshing…"
	case when != "":
		return "Updated " + when
	}
	return ""
}

func creditsCard(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	if snap == nil {
		return card(t, lay, "Credits remaining", t.Muted.Render("loading"))
	}
	acct := snap.Account

	lines := []string{t.Value.Render(domain.FormatCredits(acct.Remaining))}
	if acct.Remaining < 0 {
		lines = append(lines, t.Warning.Render("account is over its purchased credit"))
	}
	return card(t, lay, "Credits remaining", strings.Join(lines, "\n"))
}

func runwayCard(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	if snap == nil {
		return card(t, lay, "Estimated runway", t.Muted.Render("loading"))
	}
	rw := snap.Runway
	label := domain.FormatDaysRemaining(rw)
	if d := domain.FormatDepletionDate(rw); d != "" {
		label += " - " + d
	}
	value := t.Value.Render(label)
	if rw.LowConfidence {
		value = t.Warning.Render(label)
	}
	lines := []string{value}
	if rw.Basis == domain.BasisUnavailable {
		lines = append(lines, t.Muted.Render("analytics unavailable"))
	} else if rw.DailyBurn > 0 {
		lines = append(lines, t.Muted.Render("average "+domain.FormatCredits(rw.DailyBurn)+"/day"))
	}
	return card(t, lay, "Estimated runway", strings.Join(lines, "\n"))
}

func last30Card(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	if snap == nil {
		return card(t, lay, "Last 30 days", t.Muted.Render("loading"))
	}
	spend := 0.0
	for _, d := range snap.Daily {
		spend += d.CreditSpend
	}
	rows := []string{
		t.Label.Render("Spend: ") + t.Value.Render(domain.FormatCredits(spend)),
		t.Label.Render("Tokens: ") + t.Value.Render(domain.FormatTokens(snap.TotalTokens)),
	}
	return card(t, lay, "Last 30 days", strings.Join(rows, "\n"))
}

func modelsCard(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	head := t.CardHead.Render("Cost by model")
	if snap == nil || len(snap.Models) == 0 {
		return head + "\n" + t.Muted.Render("No model spend recorded.")
	}

	models := make([]domain.ModelUsage, len(snap.Models))
	copy(models, snap.Models)
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].CreditSpend == models[j].CreditSpend {
			return models[i].Model < models[j].Model
		}
		return models[i].CreditSpend > models[j].CreditSpend
	})

	show := models
	if len(show) > 5 {
		top := show[:4]
		other := 0.0
		otherTokens := int64(0)
		for _, m := range show[4:] {
			other += m.CreditSpend
			otherTokens += m.Tokens
		}
		show = append(append([]domain.ModelUsage{}, top...), domain.ModelUsage{
			Model:       "Other (remaining)",
			CreditSpend: other,
			Tokens:      otherTokens,
		})
	}

	leftWidth := lay.ContentWidth
	if lay.TwoColumn {
		leftWidth = (lay.ContentWidth - 2) / 2
	}

	// The table bubble owns alignment and truncation. Reserve the spend column
	// so money is never the thing that gets clipped.
	const spendWidth = 9
	nameWidth := leftWidth - spendWidth
	if nameWidth < 12 {
		nameWidth = 12
	}

	rows := make([]table.Row, 0, len(show))
	for _, m := range show {
		rows = append(rows, table.Row{m.Model, domain.FormatCredits(m.CreditSpend)})
	}
	tbl := NewTable([]table.Column{
		{Title: "Model", Width: nameWidth},
		{Title: "Spend", Width: spendWidth},
	}, rows, t)
	left := tbl.View()

	tok := ""
	if snap.CacheHitRate != nil {
		tok = t.Label.Render("Cache hit  ") + t.Value.Render(fmt.Sprintf("%.1f%%", *snap.CacheHitRate))
	}
	right := t.Label.Render("Total       ") + t.Value.Render(domain.FormatTokens(snap.TotalTokens))
	if tok != "" {
		right += "\n" + tok
	}

	if lay.TwoColumn {
		return head + "\n" + twoColumns(left, right, lay.ContentWidth)
	}
	return head + "\n" + left + "\n\n" + t.CardHead.Render("Token usage") + "\n" + right
}

func card(t Styles, lay Layout, title, body string) string {
	width := lay.ContentWidth
	var b strings.Builder
	b.WriteString(t.CardHead.Render(title))
	b.WriteString("\n")
	for _, line := range strings.Split(body, "\n") {
		b.WriteString(truncate(line, width))
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
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

func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// truncate cuts plain text to width, appending an ellipsis when it is cut. It
// is escape-aware: it measures the rendered width.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if len(runes) > width-1 {
		runes = runes[:width-1]
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
