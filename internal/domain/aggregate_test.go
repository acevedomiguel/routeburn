package domain

import (
	"math"
	"testing"
	"time"
)

func TestAggregateDailySumsSpendAndTokens(t *testing.T) {
	r := DisplayRange(utc(2026, time.November, 21, 2))
	rows := []DailyUsage{
		{Day: utc(2026, time.November, 20, 0), CreditSpend: 1.5, Tokens: 100},
		{Day: utc(2026, time.November, 21, 0), CreditSpend: 0.25, Tokens: 50},
		{Day: utc(2026, time.October, 1, 0), CreditSpend: 99, Tokens: 9999},
	}

	series := AggregateDaily(r, rows)

	if len(series) != 30 {
		t.Fatalf("buckets = %d, want exactly 30", len(series))
	}
	total := SumDaily(series)
	if math.Abs(total.CreditSpend-1.75) > 1e-9 {
		t.Errorf("spend = %v, want 1.75", total.CreditSpend)
	}
	if total.Tokens != 150 {
		t.Errorf("tokens = %d, want 150", total.Tokens)
	}
}

func TestAggregateDailyZeroFillsMissingDays(t *testing.T) {
	r := RunwayRange(utc(2026, time.November, 21, 2))
	series := AggregateDaily(r, nil)

	if len(series) != 30 {
		t.Fatalf("buckets = %d, want 30", len(series))
	}
	for _, b := range series {
		if b.CreditSpend != 0 || b.Tokens != 0 {
			t.Errorf("bucket %s must be zero-filled, got %+v", b.Day, b)
		}
	}
}

func TestAggregateDailyIgnoresRowsOutsideRangeAndKeepsActiveDayCount(t *testing.T) {
	r := RunwayRange(utc(2026, time.November, 21, 2))
	rows := []DailyUsage{
		{Day: utc(2026, time.November, 20, 0), CreditSpend: 2},
		{Day: utc(2026, time.November, 20, 18), CreditSpend: 3}, // same UTC day, second row
		{Day: utc(2026, time.November, 21, 5), CreditSpend: 7},  // excluded: today
	}

	agg := AggregateDaily(r, rows)
	if got := SumDaily(agg).CreditSpend; math.Abs(got-5) > 1e-9 {
		t.Errorf("spend = %v, want 5 (rows merged per day, today excluded)", got)
	}
	if got := SumDaily(agg).ActiveDays; got != 1 {
		t.Errorf("active days = %d, want 1", got)
	}
}

func TestSortModelsBySpendAndTruncate(t *testing.T) {
	models := []ModelUsage{
		{Model: "a", CreditSpend: 1, Tokens: 10},
		{Model: "b", CreditSpend: 5, Tokens: 50},
		{Model: "c", CreditSpend: 3, Tokens: 30},
	}

	sorted := SortModels(models)
	if sorted[0].Model != "b" || sorted[1].Model != "c" || sorted[2].Model != "a" {
		t.Errorf("sort order = %v, want b, c, a", []string{sorted[0].Model, sorted[1].Model, sorted[2].Model})
	}
	if models[0].Model != "a" {
		t.Error("SortModels must not mutate its input")
	}
}

func TestTopModelsAggregatesRemainderDistinctFromModelNamedOther(t *testing.T) {
	models := make([]ModelUsage, 0, 13)
	models = append(models, ModelUsage{Model: "Other", CreditSpend: 100, Tokens: 1000})
	for i := 0; i < 12; i++ {
		models = append(models, ModelUsage{Model: "m" + string(rune('a'+i)), CreditSpend: float64(12 - i), Tokens: 10})
	}

	top := TopModels(models, 10)
	if len(top) != 10 {
		t.Fatalf("rows = %d, want 10", len(top))
	}
	if top[0].Model != "Other" || top[len(top)-1].Model != "Other (remaining)" {
		t.Errorf("rows = %v..%v, want a real 'Other' first and 'Other (remaining)' last", top[0].Model, top[len(top)-1].Model)
	}
}

func TestFormatCreditsAndTokens(t *testing.T) {
	creditTests := map[float64]string{
		0:      "$0.00",
		12:     "$12.00",
		0.004:  "<$0.01",
		0.01:   "$0.01",
		1234.5: "$1,234.50",
	}
	for in, want := range creditTests {
		if got := FormatCredits(in); got != want {
			t.Errorf("FormatCredits(%v) = %q, want %q", in, got, want)
		}
	}

	tokenTests := map[int64]string{
		0:             "0",
		999:           "999",
		1500:          "1.5K",
		8_420_000:     "8.4M",
		1_250_000_000: "1.3B",
	}
	for in, want := range tokenTests {
		if got := FormatTokens(in); got != want {
			t.Errorf("FormatTokens(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatRunwayLines(t *testing.T) {
	remaining := 18.40
	finite := EstimateRunway(EstimateInput{Remaining: &remaining, WindowDays: 30, WindowSpend: 12, ActiveDays: 30, AnalyticsAvailable: true, Now: utc(2026, time.November, 21, 2)})
	if got := FormatDaysRemaining(finite); got != "46 days" {
		t.Errorf("days = %q, want %q", got, "46 days")
	}

	zero := 0.0
	burned := EstimateRunway(EstimateInput{Remaining: &zero, WindowDays: 30, WindowSpend: 12, ActiveDays: 30, AnalyticsAvailable: true, Now: utc(2026, time.November, 21, 2)})
	if got := FormatDaysRemaining(burned); got != "0 days" {
		t.Errorf("days = %q, want 0 days", got)
	}

	unavailable := EstimateRunway(EstimateInput{Remaining: &remaining, AnalyticsAvailable: false, Now: utc(2026, time.November, 21, 2)})
	if got := FormatDaysRemaining(unavailable); got != "Unavailable" {
		t.Errorf("days = %q, want Unavailable", got)
	}

	nospend := EstimateRunway(EstimateInput{Remaining: &remaining, WindowDays: 30, ActiveDays: 0, AnalyticsAvailable: true, Now: utc(2026, time.November, 21, 2)})
	if got := FormatDaysRemaining(nospend); got != "No recent spend" {
		t.Errorf("days = %q, want No recent spend", got)
	}
}

func TestFormatDepletionDate(t *testing.T) {
	remaining := 18.40
	now := utc(2026, time.November, 21, 2)
	est := EstimateRunway(EstimateInput{Remaining: &remaining, WindowDays: 30, WindowSpend: 12, ActiveDays: 30, AnalyticsAvailable: true, Now: now, Location: time.UTC})
	if got := FormatDepletionDate(est); got != "~Jan 06" {
		t.Errorf("date = %q, want ~Jan 06", got)
	}

	// The depletion date is a calendar date in the display zone: at 02:00 UTC
	// it is still 2026-11-20 in a UTC-10 zone, so the date is one day earlier.
	west := time.FixedZone("UTC-10", -10*60*60)
	westEst := EstimateRunway(EstimateInput{Remaining: &remaining, WindowDays: 30, WindowSpend: 12, ActiveDays: 30, AnalyticsAvailable: true, Now: now, Location: west})
	if got := FormatDepletionDate(westEst); got != "~Jan 05" {
		t.Errorf("date in UTC-10 = %q, want ~Jan 05", got)
	}

	if got := FormatDepletionDate(EstimateRunway(EstimateInput{Remaining: &remaining, AnalyticsAvailable: false})); got != "" {
		t.Errorf("date = %q, want empty when unavailable", got)
	}
}
