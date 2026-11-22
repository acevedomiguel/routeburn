package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// tableDaily builds n consecutive days ending on fixedNow's date, with spend of
// 0.10, 0.20, ... so the totals are easy to reason about.
func tableDaily(n int) []domain.DailyUsage {
	end := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	out := make([]domain.DailyUsage, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.DailyUsage{
			Day:          end.AddDate(0, 0, i-(n-1)),
			CreditSpend:  float64(i+1) / 10,
			Tokens:       int64(1000 * (i + 1)),
			Requests:     int64(10 * (i + 1)),
			TokensPrompt: int64(100 * (i + 1)),
			CachedTokens: int64(50 * (i + 1)),
		})
	}
	return out
}

// The panel fills exactly the height it is given, and every row is the panel
// width, so the frame geometry the dashboard depends on still holds.
func TestDailyTableFillsItsHeight(t *testing.T) {
	for _, h := range []int{6, 8, 12, 20, 36} {
		rows := DailyTable(tableDaily(30), DailyTableOptions{Width: 60, Height: h, Styles: plainStyles()})
		if len(rows) != h {
			t.Errorf("height %d: got %d rows", h, len(rows))
		}
		for i, r := range rows {
			if got := lipgloss.Width(r); got != 60 {
				t.Errorf("height %d row %d: width %d, want 60", h, i, got)
			}
		}
	}
}

// Newest day first: the first data row is the last day in the series. This is
// the whole point of the table replacing the sparkline, so the day order is
// pinned rather than incidental.
func TestDailyTableIsNewestFirst(t *testing.T) {
	rows := DailyTable(tableDaily(30), DailyTableOptions{Width: 60, Height: 9, Styles: plainStyles()})
	body := strings.Join(rows, "\n")
	if !strings.Contains(body, "Nov 21") {
		t.Fatalf("newest day missing:\n%s", body)
	}
	i21 := strings.Index(body, "Nov 21")
	i20 := strings.Index(body, "Nov 20")
	if i20 < 0 || i21 > i20 {
		t.Errorf("days are not newest-first: Nov 21 at %d, Nov 20 at %d", i21, i20)
	}
	if strings.Contains(body, "Oct 23") {
		t.Errorf("a day outside the window leaked in:\n%s", body)
	}
}

// One row per day, no invented days. The sparkline this replaces resampled 30
// days into ~62 columns, so each day appeared four or five times.
func TestDailyTableRendersEachDayOnce(t *testing.T) {
	daily := tableDaily(30)
	rows := DailyTable(daily, DailyTableOptions{Width: 60, Height: 36, Styles: plainStyles()})
	body := strings.Join(rows, "\n")
	seen := map[string]int{}
	for _, d := range daily {
		seen[d.Day.Format("Jan 02")]++
	}
	for label, want := range seen {
		if got := strings.Count(body, label); got != want {
			t.Errorf("%q appears %d times, want %d:\n%s", label, got, want, body)
		}
	}
}

// The pinned total row sums every day, not only the visible window, so
// scrolling never hides the number the panel exists to show.
func TestDailyTableTotalCoversEveryDay(t *testing.T) {
	daily := tableDaily(30)
	var want float64
	for _, d := range daily {
		want += d.CreditSpend
	}
	rows := DailyTable(daily, DailyTableOptions{Width: 60, Height: 8, Styles: plainStyles()})
	body := strings.Join(rows, "\n")
	if !strings.Contains(body, "30 days") {
		t.Fatalf("no total row:\n%s", body)
	}
	if !strings.Contains(body, domain.FormatCredits(want)) {
		t.Errorf("total row must sum all 30 days (%s):\n%s", domain.FormatCredits(want), body)
	}
}

// An offset past the end clamps instead of rendering a blank panel. Height 9
// leaves four day rows, so the last page is the four oldest days.
func TestDailyTableClampsAnOvershootingOffset(t *testing.T) {
	daily := tableDaily(30)
	rows := DailyTable(daily, DailyTableOptions{Width: 60, Height: 9, Styles: plainStyles(), Offset: 999})
	body := strings.Join(rows, "\n")
	if !strings.Contains(body, "Oct 26") || !strings.Contains(body, "Oct 23") {
		t.Errorf("clamped window must be the last page:\n%s", body)
	}
	if strings.Contains(body, "Nov 21") {
		t.Errorf("clamped window must not still show the newest day:\n%s", body)
	}
}

// Column widths plus the two padding columns per cell fill a body row exactly,
// at any panel width, so the frame's right border never drifts.
func TestDailyTableColumnsFillTheRowWidth(t *testing.T) {
	daily := tableDaily(30)
	for _, w := range []int{44, 52, 62, 82, 112, 142} {
		cols := dailyColumns(daily, w)
		sum := 0
		for _, c := range cols {
			sum += c.width + 2
		}
		if sum != w-4 {
			t.Errorf("width %d: columns + padding = %d, want %d", w, sum, w-4)
		}
	}
	if got := len(dailyColumns(daily, 62)); got != 5 {
		t.Errorf("wide table has %d columns, want 5", got)
	}
}

// A column the API never reported is dropped rather than shown as a wall of
// zeros, matching how the leader rows already behave.
func TestDailyTableOmitsZeroColumns(t *testing.T) {
	daily := tableDaily(5)
	for i := range daily {
		daily[i].Requests = 0
		daily[i].TokensPrompt = 0
		daily[i].CachedTokens = 0
	}
	body := strings.Join(DailyTable(daily, DailyTableOptions{Width: 60, Height: 11, Styles: plainStyles()}), "\n")
	if !strings.Contains(body, "DATE") || !strings.Contains(body, "SPEND") {
		t.Errorf("mandatory columns missing:\n%s", body)
	}
	if strings.Contains(body, "CACHE") {
		t.Errorf("cache column must be omitted when unreported:\n%s", body)
	}
	if strings.Contains(body, "REQS") {
		t.Errorf("request column must be omitted when unreported:\n%s", body)
	}
}

// A day without prompt tokens reads "-", never "0.0%", because a zero in this
// API means "not reported" rather than "measured zero".
func TestDailyTableUnreportedCacheRateIsADash(t *testing.T) {
	if got := formatCacheRate(0, 0); got != "-" {
		t.Errorf("unreported cache rate = %q, want \"-\"", got)
	}
	if got := formatCacheRate(50, 100); got != "50.0%" {
		t.Errorf("cache rate = %q, want 50.0%%", got)
	}
}

// A free-tier day — zero spend, real tokens — is data, not a gap.
func TestDailyTableKeepsZeroSpendDays(t *testing.T) {
	daily := tableDaily(3)
	daily[1].CreditSpend = 0
	body := strings.Join(DailyTable(daily, DailyTableOptions{Width: 60, Height: 9, Styles: plainStyles()}), "\n")
	if !strings.Contains(body, "$0.00") {
		t.Errorf("a zero-spend day must still render:\n%s", body)
	}
}

// No series at all is a message inside the frame, not an empty box.
func TestDailyTableEmptySeries(t *testing.T) {
	rows := DailyTable(nil, DailyTableOptions{Width: 60, Height: 8, Styles: plainStyles()})
	if len(rows) != 8 {
		t.Errorf("got %d rows, want 8", len(rows))
	}
	if !strings.Contains(strings.Join(rows, "\n"), "No daily data available.") {
		t.Errorf("empty series must explain itself:\n%s", strings.Join(rows, "\n"))
	}
}
