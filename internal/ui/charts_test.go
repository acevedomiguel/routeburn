package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

func day(y int, m int, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func TestDailyChartAlwaysHasThirtyBuckets(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	if len(snap.Daily) != 30 {
		t.Fatalf("fixture has %d buckets", len(snap.Daily))
	}
	out := DailyChart(snap.Daily, ChartOptions{Width: 80, Height: 5, Styles: plainStyles()})
	if !strings.Contains(out, "Daily credit spend") {
		t.Fatal("missing chart heading")
	}
}

func TestDailyChartZeroValuesDoNotDivideByZero(t *testing.T) {
	zeros := make([]domain.DailyUsage, 30)
	for i := range zeros {
		zeros[i] = domain.DailyUsage{Day: day(2026, 10, 23).AddDate(0, 0, i)}
	}
	out := DailyChart(zeros, ChartOptions{Width: 80, Height: 5, Styles: plainStyles()})
	if !strings.Contains(out, "No credit spend") {
		t.Fatalf("expected empty state, got %q", out)
	}
}

func TestDailyChartNoData(t *testing.T) {
	out := DailyChart(nil, ChartOptions{Width: 80, Height: 5, Styles: plainStyles()})
	if !strings.Contains(out, "No daily data") {
		t.Fatalf("expected no-data message: %q", out)
	}
}

func TestDailyChartSpikeKeepsOtherBarsVisible(t *testing.T) {
	daily := make([]domain.DailyUsage, 30)
	for i := range daily {
		daily[i] = domain.DailyUsage{Day: day(2026, 10, 23).AddDate(0, 0, i), CreditSpend: 0.2}
	}
	daily[15].CreditSpend = 100
	out := DailyChart(daily, ChartOptions{Width: 80, Height: 5, Styles: plainStyles()})
	if !strings.Contains(out, "+") {
		t.Fatalf("expected partial-height markers for small bars: %q", out)
	}
}

func TestDailyChartRendersAtManyWidths(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for w := 20; w <= 200; w++ {
		out := DailyChart(snap.Daily, ChartOptions{Width: w, Height: 4, Styles: plainStyles()})
		if strings.TrimSpace(out) == "" {
			t.Fatalf("empty chart at width %d", w)
		}
	}
}

func TestChartSummary(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	s := ChartSummary(snap.Daily)
	if !strings.Contains(s, "30-day spend") {
		t.Fatalf("summary = %q", s)
	}
}

func TestDateRangeLabel(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	label := DateRangeLabel(snap.Range)
	if label != "Oct 23 - Nov 21" {
		t.Fatalf("label = %q", label)
	}
}
