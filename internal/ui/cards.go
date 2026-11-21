package ui

import (
	"fmt"
	"sort"
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
}

// Render draws the full dashboard. It never performs IO.
func Render(snap *domain.Snapshot, opts ViewOptions) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	lay := Classify(opts.Width, opts.Height)
	if lay.Class == ClassTooSmall {
		return opts.Styles.Warning.Render(lay.Message) + "\n"
	}

	var b strings.Builder
	b.WriteString(header(snap, opts, lay))
	b.WriteString("\n\n")

	if opts.Banner != "" {
		b.WriteString(opts.Styles.Banner.Render(truncate(opts.Banner, lay.ContentWidth)))
		b.WriteString("\n\n")
	}

	cards := []string{
		creditsCard(snap, opts, lay),
		runwayCard(snap, opts, lay),
		last30Card(snap, opts, lay),
		keyCard(snap, opts, lay),
	}

	if lay.TwoColumn {
		left := cards[0] + "\n" + cards[1]
		right := cards[2] + "\n" + cards[3]
		b.WriteString(twoColumns(left, right, lay.ContentWidth))
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
		b.WriteString(DailyChart(snap.Daily, ChartOptions{Width: lay.ContentWidth, Height: 5, Styles: opts.Styles}))
		b.WriteString("\n")
	}
	b.WriteString(modelsCard(snap, opts, lay))

	body := b.String()
	if lay.Scrollable && opts.Height > 0 {
		body = scroll(body, opts.Scroll, opts.Height-2)
	}

	return body + "\n\n" + Footer(opts)
}

func header(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	title := opts.Styles.Title.Render("RouteBurn")
	when := ""
	if snap != nil && !snap.GeneratedAt.IsZero() {
		when = snap.GeneratedAt.Format("15:04")
	}
	status := "Updated " + when
	freshness := ""
	if opts.Cached {
		freshness = CachedLabel(opts.CacheAge, true)
		status = freshness
	}

	line := title + "  " + opts.Styles.Muted.Render(truncate(status, lay.ContentWidth-12))
	if strings.TrimSpace(headerPlain(status)) == "" {
		return title
	}
	return line
}

func headerPlain(s string) string { return s }

func creditsCard(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	if snap == nil {
		return card(t, lay, "Credits remaining", t.Muted.Render("loading"))
	}
	acct := snap.Account
	pct, clamped := Percent(acct.Remaining, acct.TotalPurchased)

	var lines []string
	lines = append(lines, t.Value.Render(domain.FormatCredits(acct.Remaining))+" / "+t.Muted.Render(domain.FormatCredits(acct.TotalPurchased)))
	barWidth := barWidthFor(lay)
	lines = append(lines, t.Bar.Render(Bar(clamped, barWidth))+" "+fmt.Sprintf("%s%% remaining", formatPercent(pct)))
	if snap.Runway.DailyBurn > 0 {
		lines = append(lines, t.Muted.Render("at "+domain.FormatCredits(snap.Runway.DailyBurn)+"/day"))
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
	} else if rw.WindowDays > 0 {
		lines = append(lines, t.Muted.Render(fmt.Sprintf("based on %s over %d days", domain.FormatCredits(rw.WindowSpend), rw.WindowDays)))
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

func keyCard(snap *domain.Snapshot, opts ViewOptions, lay Layout) string {
	t := opts.Styles
	if snap == nil || snap.Key == nil {
		return card(t, lay, "This API key", t.Muted.Render("not configured"))
	}
	k := snap.Key
	limitText := "unlimited"
	used := ""
	if k.Limit != nil {
		limitText = domain.FormatCredits(*k.Limit)
		if k.LimitRemaining != nil {
			_, clamped := Percent(*k.Limit-*k.LimitRemaining, *k.Limit)
			used = " " + t.Bar.Render(Bar(clamped, minInt(barWidthFor(lay), 12)))
		}
	}
	rows := []string{t.Label.Render("Limit:     ") + t.Value.Render(limitText) + used}
	if k.LimitRemaining != nil {
		rows = append(rows, t.Label.Render("Remaining: ")+t.Value.Render(domain.FormatCredits(*k.LimitRemaining)))
	}
	if k.UsageMonthly > 0 {
		rows = append(rows, t.Label.Render("Month:     ")+t.Value.Render(domain.FormatCredits(k.UsageMonthly)))
	}
	if k.Label != "" {
		rows = append(rows, t.Muted.Render(truncate(k.Label, lay.ContentWidth-4)))
	}
	return card(t, lay, "This API key", strings.Join(rows, "\n"))
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

	nameWidth := lay.ContentWidth - 14
	if nameWidth < 10 {
		nameWidth = 10
	}
	var lines []string
	for _, m := range show {
		name := truncate(m.Model, nameWidth)
		lines = append(lines, padRight(name, nameWidth)+" "+padLeft(domain.FormatCredits(m.CreditSpend), 8))
	}

	tok := ""
	if snap.CacheHitRate != nil {
		tok = t.Label.Render("Cache hit  ") + t.Value.Render(fmt.Sprintf("%.1f%%", *snap.CacheHitRate))
	}

	left := strings.Join(lines, "\n")
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

func barWidthFor(lay Layout) int {
	w := lay.ContentWidth / 2
	if w < 8 {
		w = 8
	}
	if w > 24 {
		w = 24
	}
	return w
}

func formatPercent(v float64) string {
	if v >= 10 || v == 0 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// padRight pads using display width so ANSI escapes do not skew the column.
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
