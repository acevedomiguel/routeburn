package ui

import (
	"fmt"
	"strings"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// ChartOptions configures the daily bar chart.
type ChartOptions struct {
	Width  int
	Height int
	Styles Styles
}

// DailyChart renders a vertical bar chart of daily credit spend. It uses only
// ASCII so it survives every terminal and color mode.
func DailyChart(daily []domain.DailyUsage, opts ChartOptions) string {
	t := opts.Styles
	width := opts.Width
	if width < 20 {
		width = 20
	}
	height := opts.Height
	if height < 3 {
		height = 3
	}

	head := t.CardHead.Render("Daily credit spend")
	if len(daily) == 0 {
		return head + "\n" + t.Muted.Render("No daily data available.")
	}

	max := 0.0
	for _, d := range daily {
		if d.CreditSpend > max {
			max = d.CreditSpend
		}
	}
	if max <= 0 {
		return head + "\n" + t.Muted.Render("No credit spend in the last 30 days.")
	}

	// Reserve room for the y-axis labels and one gutter column.
	axisWidth := 7
	buckets := len(daily)
	plot := width - axisWidth - 1
	if plot < buckets {
		plot = buckets
	}
	if plot > width-axisWidth-1 {
		plot = width - axisWidth - 1
	}
	if plot < 1 {
		plot = 1
	}

	// Sample buckets into plot slots when the terminal cannot fit one per day.
	values := make([]float64, plot)
	for i := 0; i < buckets; i++ {
		slot := i * plot / buckets
		if daily[i].CreditSpend > values[slot] {
			values[slot] = daily[i].CreditSpend
		}
	}

	labels := make([]string, height)
	for row := 0; row < height; row++ {
		// row 0 is the topmost bar row; vertical fill is proportional.
		from := max * float64(height-row-1) / float64(height)
		to := max * float64(height-row) / float64(height)
		var b strings.Builder
		if row == height-1 {
			b.WriteString(fmt.Sprintf("%6s ", domain.FormatCredits(0)))
		} else if row == 0 {
			b.WriteString(fmt.Sprintf("%6s ", domain.FormatCredits(max)))
		} else {
			b.WriteString(strings.Repeat(" ", axisWidth))
		}
		for _, v := range values {
			switch {
			case v >= to:
				b.WriteString("#")
			case v > from:
				b.WriteString("+")
			default:
				b.WriteString(" ")
			}
		}
		labels[row] = strings.TrimRight(b.String(), " ")
	}

	var out strings.Builder
	out.WriteString(head)
	out.WriteString("\n")
	for _, line := range labels {
		out.WriteString(line)
		out.WriteString("\n")
	}

	// Axis line plus date labels for first, middle, last buckets.
	out.WriteString(strings.Repeat("-", axisWidth))
	out.WriteString("+")
	out.WriteString(strings.Repeat("-", plot))
	out.WriteString("\n")

	first := daily[0].Day.UTC().Format("Jan 02")
	mid := daily[len(daily)/2].Day.UTC().Format("Jan 02")
	last := daily[len(daily)-1].Day.UTC().Format("Jan 02")
	spine := strings.Repeat(" ", axisWidth+1)
	if plot >= 24 {
		midPos := plot / 2
		line := []byte(spine + strings.Repeat(" ", plot))
		copy(line[axisWidth+1:], first)
		copy(line[axisWidth+1+midPos-len(mid)/2:], mid)
		start := axisWidth + 1 + plot - len(last)
		copy(line[start:], last)
		out.WriteString(strings.TrimRight(string(line), " "))
		out.WriteString("\n")
	} else {
		out.WriteString(spine + first + " .. " + last)
		out.WriteString("\n")
	}

	return out.String()
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
