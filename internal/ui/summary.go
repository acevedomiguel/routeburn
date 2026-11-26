package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/acevedomiguel/burnblocks/openrouter"
	"github.com/acevedomiguel/routeburn/internal/format"
)

// lipglossWidth is the printable width of a string in display cells. Cells
// may arrive pre-coloured (the daily table's heat ramp), so the measurement
// must be escape-aware; rune counting would miscount every styled value.
func lipglossWidth(s string) int {
	return lipgloss.Width(s)
}

// ChartSummary is the text alternative for the chart, used by tests and
// screen readers.
func ChartSummary(daily []openrouter.DailyUsage) string {
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
		format.FormatCredits(total),
		format.FormatCredits(peak.CreditSpend),
		peak.Day.UTC().Format("Jan 02"))
}

// DateRangeLabel describes the chart window for headers.
func DateRangeLabel(r openrouter.TimeRange) string {
	if r.Start.IsZero() || r.End.IsZero() {
		return ""
	}
	last := r.End.AddDate(0, 0, -1)
	return r.Start.UTC().Format("Jan 02") + " - " + last.UTC().Format("Jan 02")
}
