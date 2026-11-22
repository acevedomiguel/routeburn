package ui

import (
	"fmt"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// lipglossWidth is the printable width of a string. Every glyph the dashboard
// draws is single-width, so rune count is exact and avoids an escape-aware
// measurement on strings the caller has already built.
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
