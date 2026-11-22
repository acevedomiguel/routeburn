package ui

import (
	"fmt"
	"math"
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

// Sparkline renders the last-30-days credit spend as a bordered panel.
//
// At the default three rows it draws one row of graded blocks with the peak
// caption flush right on the same row. When the caller gives it more height it
// draws a multi-row bar chart and moves the caption into the top border, so the
// bars get the panel's full width. Either way the series is resampled to fill
// the row instead of stopping short of the caption.
func Sparkline(daily []domain.DailyUsage, opts ChartOptions) string {
	t := opts.Styles
	width := opts.Width
	if width < 20 {
		width = 20
	}
	height := opts.Height
	if height < 3 {
		height = 3
	}
	rows := height - 2

	message := func(msg string) string {
		out := panel("Daily credit spend", []string{t.Muted.Render(msg)}, width, height, t)
		return strings.Join(out, "\n") + "\n"
	}
	if len(daily) == 0 {
		return message("No daily data available.")
	}

	peak := daily[0]
	for _, d := range daily {
		if d.CreditSpend > peak.CreditSpend {
			peak = d
		}
	}
	if peak.CreditSpend <= 0 {
		return message("No credit spend in the last 30 days.")
	}

	caption := fmt.Sprintf("peak %s · %s",
		domain.FormatCredits(peak.CreditSpend),
		peak.Day.UTC().Format("Jan 02"))

	// The panel body has width-4 usable columns.
	inner := width - 4
	if inner < 1 {
		inner = 1
	}

	if rows == 1 {
		// One row: graded blocks with the caption inline, as the design note
		// specifies. The series is resampled up to the available columns so the
		// chart reaches the caption instead of leaving a gap.
		captionWidth := len([]rune(caption))
		slots := inner - captionWidth - 2
		if slots < 1 {
			slots = 1
		}
		bars := renderSparkBars(daily, slots)
		row := bars
		if gap := inner - lipglossWidth(bars) - captionWidth; gap >= 1 {
			row = bars + strings.Repeat(" ", gap) + t.Muted.Render(caption)
		}
		out := panel("Daily credit spend", []string{row}, width, height, t)
		return strings.Join(out, "\n") + "\n"
	}

	body := renderBars(daily, inner, rows)
	out := panelWithCaption("Daily credit spend", caption, body, width, height, t)
	return strings.Join(out, "\n") + "\n"
}

// slotValues resamples the daily series into exactly slots columns. Narrow
// panels merge neighbouring days by taking the bucket's peak; wide panels
// repeat a day across the columns it owns. Either way every column is filled.
func slotValues(daily []domain.DailyUsage, slots int) []float64 {
	if slots < 1 {
		slots = 1
	}
	values := make([]float64, slots)
	n := len(daily)
	if n == 0 {
		return values
	}
	for i := 0; i < slots; i++ {
		start := i * n / slots
		end := (i + 1) * n / slots
		if end <= start {
			end = start + 1
		}
		if end > n {
			end = n
		}
		v := daily[start].CreditSpend
		for _, d := range daily[start:end] {
			if d.CreditSpend > v {
				v = d.CreditSpend
			}
		}
		values[i] = v
	}
	return values
}

// renderSparkBars maps one graded block per column, for the single-row form.
func renderSparkBars(daily []domain.DailyUsage, slots int) string {
	values := slotValues(daily, slots)
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

// renderBars draws a multi-row bar chart: one column per slot, rows top to
// bottom. A column fills ceil(value/peak*rows) cells from the bottom, so the
// tallest day touches the top row. Zero columns keep a baseline block on the
// bottom row so the series still reads as a series.
func renderBars(daily []domain.DailyUsage, slots, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	values := slotValues(daily, slots)
	peak := 0.0
	for _, v := range values {
		if v > peak {
			peak = v
		}
	}

	heights := make([]int, len(values))
	for i, v := range values {
		if v <= 0 || peak <= 0 {
			continue
		}
		h := int(math.Ceil(v / peak * float64(rows)))
		if h < 1 {
			h = 1
		}
		if h > rows {
			h = rows
		}
		heights[i] = h
	}

	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		fromBottom := rows - r
		var b strings.Builder
		for _, h := range heights {
			switch {
			case h >= fromBottom:
				b.WriteRune('█')
			case fromBottom == 1 && h == 0:
				b.WriteRune(sparkBlocks[0])
			default:
				b.WriteRune(' ')
			}
		}
		out[r] = b.String()
	}
	return out
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
