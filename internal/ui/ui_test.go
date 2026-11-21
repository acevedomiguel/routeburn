package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestClassifyLayout(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		want          Class
	}{
		{"tiny width", 59, 40, ClassTooSmall},
		{"tiny height", 80, 15, ClassTooSmall},
		{"exactly 60x16", 60, 16, ClassNarrow},
		{"narrow", 79, 30, ClassNarrow},
		{"normal lower", 80, 24, ClassNormal},
		{"normal upper", 99, 50, ClassNormal},
		{"wide", 100, 40, ClassWide},
		{"very wide", 240, 60, ClassWide},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.width, tc.height)
			if got.Class != tc.want {
				t.Fatalf("Class = %v, want %v", got.Class, tc.want)
			}
			if tc.want == ClassTooSmall {
				if got.Message != TooSmallMessage {
					t.Errorf("message = %q", got.Message)
				}
				return
			}
			if got.ContentWidth > tc.width {
				t.Errorf("content width %d exceeds terminal width %d", got.ContentWidth, tc.width)
			}
			if got.ContentWidth <= 0 {
				t.Errorf("content width = %d", got.ContentWidth)
			}
		})
	}
}

func TestLayoutTwoColumnOnlyWhenWide(t *testing.T) {
	if Classify(99, 40).TwoColumn {
		t.Error("99 columns must not use two columns")
	}
	if !Classify(100, 40).TwoColumn {
		t.Error("100 columns must use two columns")
	}
	if Classify(70, 40).TwoColumn {
		t.Error("narrow layouts must stack vertically")
	}
}

func TestLayoutScrollableOnlyWhenNarrow(t *testing.T) {
	if !Classify(70, 40).Scrollable {
		t.Error("narrow layout should be scrollable")
	}
	if Classify(80, 40).Scrollable {
		t.Error("normal layout should not scroll")
	}
}

func TestTooSmallMessage(t *testing.T) {
	if TooSmallMessage != "Terminal too small; resize to at least 60x16" {
		t.Fatalf("message = %q", TooSmallMessage)
	}
	if strings.Contains(TooSmallMessage, "\u00d7") {
		t.Error("message must use ASCII for narrow terminals")
	}
}

func TestProgressPercent(t *testing.T) {
	cases := []struct {
		name        string
		part, tot   float64
		want, wantP float64
	}{
		{"half", 50, 100, 50, 50},
		{"zero total", 10, 0, 0, 0},
		{"negative total", 10, -5, 0, 0},
		{"over budget", 250, 100, 250, 100},
		{"negative part", -10, 100, -10, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pct, clamped := Percent(tc.part, tc.tot)
			if pct != tc.want {
				t.Errorf("Percent = %v, want %v", pct, tc.want)
			}
			if clamped != tc.wantP {
				t.Errorf("clamped = %v, want %v", clamped, tc.wantP)
			}
		})
	}
}

func TestBarWidthIsStable(t *testing.T) {
	for width := 1; width <= 40; width++ {
		bar := Bar(50, width)
		if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
			t.Fatalf("width %d: bar %q is not bracketed", width, bar)
		}
	}
	if len([]rune(Bar(100, 10))) != len([]rune(Bar(0, 10))) {
		t.Error("bar width must not depend on the value")
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "just now"},
		{-5 * time.Second, "just now"},
		{45 * time.Second, "just now"},
		{3 * time.Minute, "3m ago"},
		{2*time.Hour + 14*time.Minute, "2h 14m ago"},
		{26 * time.Hour, "1d 2h ago"},
		{72 * time.Hour, "3d ago"},
	}
	for _, tc := range cases {
		if got := FormatAge(tc.d); got != tc.want {
			t.Errorf("FormatAge(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestAgoCachedLabel(t *testing.T) {
	got := CachedLabel(2*time.Hour+14*time.Minute, true)
	if got != "Cached data - updated 2h 14m ago" {
		t.Fatalf("label = %q", got)
	}
	if strings.Contains(got, "\u00b7") {
		t.Error("cached label must stay ASCII-safe")
	}
}

func TestNoColorStylesArePlain(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	s := NewStyles(false)
	if _, ok := s.Title.GetForeground().(lipgloss.NoColor); !ok {
		t.Error("no-color title must have no foreground color")
	}
	if out := s.Title.Render("hello"); out != "hello" {
		t.Errorf("rendered = %q", out)
	}
}

func TestColorStylesEmitANSI(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	s := NewStyles(true)
	if out := s.Accent.Render("hi"); !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI escapes, got %q", out)
	}
}

func TestSnapshotFixtureIsSanitized(t *testing.T) {
	snap := SampleSnapshot()
	if snap.Account.Remaining <= 0 {
		t.Error("fixture should have a positive balance")
	}
	if len(snap.Daily) != 30 {
		t.Errorf("fixture day count = %d, want 30", len(snap.Daily))
	}
	for _, d := range snap.Daily {
		if d.Day.Location() != time.UTC {
			t.Errorf("fixture day %v is not UTC", d.Day)
		}
	}
	if len(snap.Models) == 0 {
		t.Error("fixture should include model spend")
	}
}

func TestSampleSnapshotMatchesRange(t *testing.T) {
	now := time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
	snap := SampleSnapshotAt(now)
	rng := domain.DisplayRange(now)
	if !snap.Range.Start.Equal(rng.Start) || !snap.Range.End.Equal(rng.End) {
		t.Errorf("range = %+v, want %+v", snap.Range, rng)
	}
	if !snap.Daily[0].Day.Equal(rng.Start) {
		t.Errorf("first day = %v, want %v", snap.Daily[0].Day, rng.Start)
	}
	if !snap.Daily[len(snap.Daily)-1].Day.Equal(rng.End.AddDate(0, 0, -1)) {
		t.Error("last day must be the day before the exclusive end")
	}
}
