package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func plainStyles() Styles { return NewStyles(false) }

func fixedNow() time.Time {
	return time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
}

func TestRenderNeverPanicsAcrossWidths(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	s := plainStyles()
	for w := 20; w <= 240; w += 4 {
		for _, h := range []int{10, 16, 24, 40, 80} {
			out := Render(&snap, ViewOptions{Width: w, Height: h, Styles: s, Now: fixedNow()})
			if strings.TrimSpace(out) == "" {
				t.Fatalf("empty render at %dx%d", w, h)
			}
		}
	}
}

func TestRenderTooSmallShowsResizeHint(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 40, Height: 10, Styles: plainStyles()})
	if !strings.Contains(out, TooSmallMessage) {
		t.Fatalf("missing resize hint: %q", out)
	}
}

func TestRenderShowsCoreFacts(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 100, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	for _, want := range []string{
		"RouteBurn",
		"Credits remaining",
		"$42.10",
		"$100.00",
		"Estimated runway",
		"102 days",
		"Last 30 days",
		"This API key",
		"anthropic/claude-sonnet-4",
		"Daily credit spend",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render is missing %q", want)
		}
	}
}

func TestRenderWideUsesTwoColumns(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	wide := Render(&snap, ViewOptions{Width: 120, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	normal := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	if len(strings.Split(wide, "\n")) > len(strings.Split(normal, "\n")) {
		t.Error("two-column wide layout should not be taller than the stacked layout")
	}
	if !strings.Contains(wide, "Credits remaining") {
		t.Error("wide layout lost the credits card")
	}
}

func TestRenderLinesFitWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	snapshot := SampleSnapshotAt(fixedNow())
	s := NewStyles(true)
	for _, w := range []int{60, 72, 80, 100, 140} {
		out := Render(&snapshot, ViewOptions{Width: w, Height: 40, Styles: s, Now: fixedNow()})
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Fatalf("width %d line %d is %d wide: %q", w, i, got, line)
			}
		}
	}
}

func TestRenderNilSnapshotIsLoading(t *testing.T) {
	out := Render(nil, ViewOptions{Width: 80, Height: 24, Styles: plainStyles(), Now: fixedNow()})
	if !strings.Contains(out, "loading") {
		t.Fatalf("nil snapshot should render a loading card: %q", out)
	}
}

func TestRenderCachedLabel(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{
		Width: 80, Height: 30, Styles: plainStyles(), Now: fixedNow(),
		Cached: true, CacheAge: 2*time.Hour + 14*time.Minute,
	})
	if !strings.Contains(out, "Cached data - updated 2h 14m ago") {
		t.Fatalf("missing cached label: %q", out)
	}
}

func TestRenderBannerApppears(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{
		Width: 80, Height: 30, Styles: plainStyles(), Now: fixedNow(),
		Banner: "Refresh failed: network unreachable - press r to retry",
	})
	if !strings.Contains(out, "Refresh failed") {
		t.Fatalf("missing banner: %q", out)
	}
}

func TestRenderKeyUnlimited(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	snap.Key = &domain.KeyCredits{Label: "unlimited-key"}
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	if !strings.Contains(out, "unlimited") {
		t.Fatalf("expected unlimited label: %q", out)
	}
}

func TestRenderNoKeyConfigured(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	snap.Key = nil
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	if !strings.Contains(out, "not configured") {
		t.Fatalf("expected a not-configured key card: %q", out)
	}
}

func TestRenderRunwayUnavailable(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	snap.Runway = domain.RunwayEstimate{Basis: domain.BasisUnavailable}
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	if !strings.Contains(out, "Unavailable") {
		t.Fatalf("expected unavailable runway: %q", out)
	}
}

func TestRenderDeterministic(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	opts := ViewOptions{Width: 100, Height: 40, Styles: plainStyles(), Now: fixedNow()}
	first := Render(&snap, opts)
	for i := 0; i < 5; i++ {
		if got := Render(&snap, opts); got != first {
			t.Fatalf("render is not deterministic at iteration %d", i)
		}
	}
}

func TestRenderNoANSIWhenColorsDisabled(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 100, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	if strings.Contains(out, "\x1b[") {
		t.Fatal("no-color render contains ANSI escapes")
	}
}

func TestRenderScrollableNarrowClipsToHeight(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 70, Height: 20, Styles: plainStyles(), Now: fixedNow(), Scroll: 0})
	lines := strings.Split(out, "\n")
	if len(lines) > 24 {
		t.Fatalf("narrow layout ignored the height hint: %d lines", len(lines))
	}
}

func TestFooterOnlyAdvertisesImplementedKeys(t *testing.T) {
	s := plainStyles()
	footer := Footer(ViewOptions{Styles: s})
	for _, want := range []string{"[r]", "[q]", "[?]"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer missing %q: %q", want, footer)
		}
	}
	for _, notYet := range []string{"export", "settings", "analyze"} {
		if strings.Contains(footer, notYet) {
			t.Errorf("footer advertises unimplemented feature %q", notYet)
		}
	}
}

func TestHelpTextListsBindings(t *testing.T) {
	s := plainStyles()
	help := HelpText(s)
	for _, bind := range Bindings() {
		if !strings.Contains(help, bind.Action) {
			t.Errorf("help missing %q", bind.Action)
		}
	}
}
