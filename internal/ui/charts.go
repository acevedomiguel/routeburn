package ui

import (
	"fmt"
	"strings"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// ChartOptions configures the daily spend visualisation.
type ChartOptions struct {
	Width  int
	Height int
	Styles Styles
}

// sparkBlocks are the eight block elements, lowest to highest. They are
// unambiguous in every terminal font and need no colour.
var sparkBlocks = []rune{'\u2581', '\u2582', '\u2583', '\u2584', '\u2585', '\u2586', '\u2587', '\u2588'}

// Sparkline renders the last-30-days credit spend as a single row of block
// characters plus one caption row. It replaces the earlier five-row ASCII
// plot, which was hard to read and overflowed narrow terminals.
func Sparkline(daily []domain.DailyUsage, opts ChartOptions) string {
	t := opts.Styles
	width := opts.Width
	if width < 20 {
		width = 20
	}

	head := t.CardHead.Render("Daily credit spend")
	if len(daily) == 0 {
		return head + "\n" + t.Muted.Render("No daily data available.") + "\n"
	}

	peak := daily[0]
	for _, d := range daily {
		if d.CreditSpend > peak.CreditSpend {
			peak = d
		}
	}
	if peak.CreditSpend <= 0 {
		return head + "\n" + t.Muted.Render("No credit spend in the last 30 days.") + "\n"
	}

	// One block per day, but never wider than the terminal.
	slots := len(daily)
	if budget := width - 30; budget > 0 && budget < slots {
		slots = budget
	}
	if slots < 1 {
		slots = 1
	}
	bars := renderSparkBars(daily, slots)
	caption := fmt.Sprintf("peak %s on %s",
		domain.FormatCredits(peak.CreditSpend),
		peak.Day.UTC().Format("Jan 02"))

	line := bars
	if lipglossWidth(bars)+1+len(caption) <= width {
		line += " " + t.Muted.Render(caption)
	} else {
		return head + "\n" + bars + "\n" + t.Muted.Render(caption) + "\n"
	}
	return head + "\n" + line + "\n"
}

// renderSparkBars maps one block per day, preserving the whole series when it
// fits and sampling evenly when the terminal is too narrow.
func renderSparkBars(daily []domain.DailyUsage, slots int) string {
	if slots < 1 {
		slots = 1
	}
	values := make([]float64, slots)
	for i := range values {
		values[i] = -1
	}
	for i, d := range daily {
		slot := i * slots / len(daily)
		if slot >= slots {
			slot = slots - 1
		}
		if d.CreditSpend > values[slot] {
			values[slot] = d.CreditSpend
		}
	}

	peak := 0.0
	for _, v := range values {
		if v > peak {
			peak = v
		}
	}

	var b strings.Builder
	for _, v := range values {
		if v <= 0 {
			b.WriteRune(sparkBlocks[0])
			continue
		}
		idx := int(v / peak * float64(len(sparkBlocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBlocks) {
			idx = len(sparkBlocks) - 1
		}
		b.WriteRune(sparkBlocks[idx])
	}
	return b.String()
}

func lipglossWidth(s string) int {
	return len([]rune(s))
}

// ChartSummary is the text alternative for the chart, used by tests and
// screen readers.
func ChartSummary(daily []domain.DailyUsage) string {
	if len(daily) == 0 {
		return "No daily data available."
	}
	total := 0.0
	peak := daily[0]
	for _, d := range daily {
		total += d.CreditSpend
		if d.CreditSpend > peak.CreditSpend {
			peak = d
		}
	}
	return fmt.Sprintf("30-day spend %s; peak %s on %s",
		domain.FormatCredits(total),
		domain.FormatCredits(peak.CreditSpend),
		peak.Day.UTC().Format("Jan 02"))
}

// DateRangeLabel describes the chart window for headers.
func DateRangeLabel(r domain.TimeRange) string {
	if r.Start.IsZero() || r.End.IsZero() {
		return ""
	}
	last := r.End.AddDate(0, 0, -1)
	return r.Start.UTC().Format("Jan 02") + " - " + last.UTC().Format("Jan 02")
}
