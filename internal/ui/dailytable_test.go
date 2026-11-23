package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	if got := len(dailyColumns(daily, 62)); got != 6 {
		t.Errorf("wide table has %d columns, want 6 with the bar column", got)
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

// The BAR column absorbs the surplus width that used to pad DATE: at width 62
// the fixed columns leave 11 spare cells, enough for a bar.
func TestDailyTableBarColumnAbsorbsTheSurplus(t *testing.T) {
	cols := dailyColumns(tableDaily(30), 62)
	if len(cols) != 6 {
		t.Fatalf("width 62 has %d columns, want 6 (DATE BAR SPEND TOKENS REQS CACHE)", len(cols))
	}
	if cols[0].title != "DATE" || cols[1].title != "" {
		t.Fatalf("first two columns = %q, %q; want DATE then the bar", cols[0].title, cols[1].title)
	}
	if cols[0].width != 11 {
		t.Errorf("DATE width = %d, want its fixed 11 now that BAR takes the surplus", cols[0].width)
	}
	sum := 0
	for _, c := range cols {
		sum += c.width + 2
	}
	if sum != 62-4 {
		t.Errorf("columns + padding = %d, want %d", sum, 62-4)
	}
}

// Bars scale to the peak day: the biggest spender fills the column, a
// half-spend day fills about half.
func TestDailyTableBarsScaleToThePeak(t *testing.T) {
	daily := []domain.DailyUsage{
		{Day: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), CreditSpend: 1.00, Tokens: 100},
		{Day: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC), CreditSpend: 2.00, Tokens: 200},
	}
	rows := DailyTable(daily, DailyTableOptions{Width: 62, Height: 8, Styles: plainStyles()})
	var peakRow, halfRow string
	for _, r := range rows {
		switch {
		case strings.Contains(r, "Nov 21"):
			peakRow = r
		case strings.Contains(r, "Nov 20"):
			halfRow = r
		}
	}
	if peakRow == "" || halfRow == "" {
		t.Fatalf("missing day rows:\n%s", strings.Join(rows, "\n"))
	}
	peak := strings.Count(peakRow, "▇")
	half := strings.Count(halfRow, "▇")
	if peak < 2 {
		t.Fatalf("the peak day has no bar: %q", peakRow)
	}
	if half != (peak+1)/2 && half != peak/2 {
		t.Errorf("half-spend bar = %d cells, want about half of the peak's %d", half, peak)
	}
}

// Expensive days are hot-coloured, cheap days green: the heat of a day is its
// share of the peak spend, mapped onto a green-to-red ramp.
func TestDailyTableHeatRampEndpoints(t *testing.T) {
	r0, g0, b0, _ := heatColor(0).RGBA()
	r1, g1, b1, _ := heatColor(1).RGBA()
	if g0 <= r0 || g0 <= b0 {
		t.Errorf("heat(0) must be green-dominant, got #%02x%02x%02x", r0>>8, g0>>8, b0>>8)
	}
	if r1 <= g1 || r1 <= b1 {
		t.Errorf("heat(1) must be red-dominant, got #%02x%02x%02x", r1>>8, g1>>8, b1>>8)
	}
}

// With colour on, the peak day's spend and bar carry escape sequences; a
// zero-spend day stays muted.
func TestDailyTableColoursExpensiveDays(t *testing.T) {
	daily := []domain.DailyUsage{
		{Day: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), CreditSpend: 0, Tokens: 0},
		{Day: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC), CreditSpend: 2.00, Tokens: 200},
	}
	rows := DailyTable(daily, DailyTableOptions{Width: 62, Height: 8, Styles: NewStyles(true)})
	var peakRow, zeroRow string
	for _, r := range rows {
		switch {
		case strings.Contains(r, "Nov 21"):
			peakRow = r
		case strings.Contains(r, "Nov 20"):
			zeroRow = r
		}
	}
	if !strings.Contains(peakRow, "\x1b[") {
		t.Errorf("the peak day must be heat-coloured: %q", peakRow)
	}
	if strings.Contains(zeroRow, "\x1b[38") && strings.Contains(zeroRow, "▇") {
		t.Errorf("a zero day must not draw a coloured bar: %q", zeroRow)
	}
}

// Under --no-color the whole panel is plain text, bars included.
func TestDailyTableNoColorHasNoEscapes(t *testing.T) {
	rows := DailyTable(tableDaily(30), DailyTableOptions{Width: 80, Height: 12, Styles: plainStyles()})
	if out := strings.Join(rows, "\n"); strings.Contains(out, "\x1b[") {
		t.Fatalf("no-color table contains escapes: %q", out)
	}
}

// A focused table marks its title so the Tab target is visible; unfocused
// renders stay unchanged.
func TestFocusedTableMarksItsTitle(t *testing.T) {
	daily := tableDaily(5)
	on := DailyTable(daily, DailyTableOptions{Width: 60, Height: 9, Styles: NewStyles(true), Focused: true})
	off := DailyTable(daily, DailyTableOptions{Width: 60, Height: 9, Styles: NewStyles(true)})
	if strings.Join(on, "\n") == strings.Join(off, "\n") {
		t.Fatal("a focused table must look different from an unfocused one")
	}
	if !strings.Contains(on[0], "\x1b[") {
		t.Errorf("focused title must be highlighted: %q", on[0])
	}
	plain := DailyTable(daily, DailyTableOptions{Width: 60, Height: 9, Styles: plainStyles(), Focused: true})
	if out := strings.Join(plain, "\n"); strings.Contains(out, "\x1b[") {
		t.Errorf("focused --no-color table must not emit escapes: %q", out)
	}
}

// padCell measures display cells, not runes: a heat-coloured value carries
// ANSI escapes and must still pad to its column width, or the SPEND column
// loses its right alignment in colour mode.
func TestPadCellAlignsColouredValues(t *testing.T) {
	coloured := heatStyle(NewStyles(true), 1).Render("$2.00")
	padded := padCell(coloured, 7, true)
	if got := lipgloss.Width(padded); got != 7 {
		t.Errorf("padded width = %d, want 7: %q", got, padded)
	}
	if !strings.HasPrefix(padded, "  ") {
		t.Errorf("right-aligned value must be left-padded: %q", padded)
	}
}

// In colour mode every day's spend sits on the same right edge, under the
// right-aligned SPEND header — including a plain, uncoloured zero day among
// heat-coloured ones.
func TestDailyTableSpendColumnStaysRightAlignedInColour(t *testing.T) {
	daily := tableDaily(5)
	daily[4].CreditSpend = 0 // a plain row among heat-coloured ones
	rows := DailyTable(daily, DailyTableOptions{Width: 80, Height: 11, Styles: NewStyles(true)})
	edge := -1
	for _, r := range rows {
		plain := ansi.Strip(r)
		i := strings.Index(plain, "$")
		if i < 0 || strings.Contains(plain, "days") {
			continue
		}
		if edge < 0 {
			edge = i
			continue
		}
		if i != edge {
			t.Errorf("spend column drifts: first edge %d, row edge %d: %q", edge, i, plain)
		}
	}
	if edge < 0 {
		t.Fatal("no spend values found")
	}
}
