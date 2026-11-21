package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

func sparkDaily(spend ...float64) []domain.DailyUsage {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	out := make([]domain.DailyUsage, 0, len(spend))
	for i, v := range spend {
		out = append(out, domain.DailyUsage{Day: start.AddDate(0, 0, i), CreditSpend: v})
	}
	return out
}

// The sparkline must be short: one row of blocks plus one caption row. The old
// five-row ASCII plot was rejected.
func TestSparklineIsTwoRows(t *testing.T) {
	out := Sparkline(sparkDaily(0, 0.5, 1.2, 0.3), ChartOptions{Width: 80, Styles: NewStyles(false)})
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("sparkline must be exactly 2 rows, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Daily credit spend") {
		t.Errorf("first row must be the section head, got %q", lines[0])
	}
}

// Block characters must scale with the actual values.
func TestSparklineUsesBlockCharacters(t *testing.T) {
	out := Sparkline(sparkDaily(0, 0.5, 1.2, 0.3), ChartOptions{Width: 80, Styles: NewStyles(false)})
	if !strings.ContainsAny(out, "▁▂▃▄▅▆▇█") {
		t.Fatalf("sparkline must use block elements, got:\n%s", out)
	}
	for _, bad := range []string{"#", "+", "|"} {
		if strings.Contains(out, bad) {
			t.Errorf("sparkline must not fall back to ASCII %q:\n%s", bad, out)
		}
	}
}

// Zero days must render as the lowest block, not vanish.
func TestSparklineRendersZeroDaysAsBaseline(t *testing.T) {
	out := Sparkline(sparkDaily(0, 0, 0, 1.0), ChartOptions{Width: 80, Styles: NewStyles(false)})
	if !strings.Contains(out, "▁") {
		t.Errorf("zero days must render the lowest block:\n%s", out)
	}
}

// The caption must name the peak day and the peak amount.
func TestSparklineCaptionNamesPeak(t *testing.T) {
	out := Sparkline(sparkDaily(0.1, 1.2, 0.3), ChartOptions{Width: 80, Styles: NewStyles(false)})
	if !strings.Contains(out, "$1.20") {
		t.Errorf("caption must show the peak amount:\n%s", out)
	}
	if !strings.Contains(out, "Nov 02") {
		t.Errorf("caption must show the peak day:\n%s", out)
	}
}

// The sparkline must never exceed the terminal width.
func TestSparklineFitsWidth(t *testing.T) {
	var spend []float64
	for i := 0; i < 30; i++ {
		spend = append(spend, float64(i)/10)
	}
	for _, w := range []int{40, 60, 80, 100, 140} {
		out := Sparkline(sparkDaily(spend...), ChartOptions{Width: w, Styles: NewStyles(false)})
		for _, line := range strings.Split(out, "\n") {
			if lipgloss.Width(line) > w {
				t.Errorf("width %d: line %d columns wide: %q", w, lipgloss.Width(line), line)
			}
		}
	}
}

func TestSparklineEmptyAndZero(t *testing.T) {
	if out := Sparkline(nil, ChartOptions{Width: 80, Styles: NewStyles(false)}); !strings.Contains(out, "No daily data") {
		t.Errorf("empty series must say so, got %q", out)
	}
	if out := Sparkline(sparkDaily(0, 0, 0), ChartOptions{Width: 80, Styles: NewStyles(false)}); !strings.Contains(out, "No credit spend") {
		t.Errorf("all-zero series must say so, got %q", out)
	}
}
