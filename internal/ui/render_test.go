package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/acevedomiguel/routeburn/internal/domain"
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
	// The app name is deliberately absent: the outer frame that carried it was
	// removed at the user's request, and nothing else repeats the wordmark.
	if strings.Contains(out, "RouteBurn") {
		t.Errorf("dashboard must not repeat the app name:\n%s", out)
	}
	for _, want := range []string{
		"Credits",
		"$42.10",
		"Runway",
		"102 days",
		"Last 30 days",
		"claude-sonnet-4",
		"Daily spend",
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
	if !strings.Contains(wide, "Credits") {
		t.Error("wide layout lost the credits panel")
	}
}

func TestRenderLinesFitWidth(t *testing.T) {
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

// The per-key limit card was removed at the user's request: only account
// credit is rendered, never the key's own limit.
func TestRenderOmitsThePerKeyLimit(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	for _, unwanted := range []string{"This API key", "Limit:", "production"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("render must not show the per-key section (%q):\n%s", unwanted, out)
		}
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

func TestFooterReflectsHelpOverlayKeys(t *testing.T) {
	footer := Footer(ViewOptions{Styles: plainStyles(), Width: 100, HelpVisible: true})
	for _, want := range []string{"scroll", "close", "quit"} {
		if !strings.Contains(footer, want) {
			t.Errorf("Help footer missing %q: %q", want, footer)
		}
	}
	for _, suspended := range []string{"refresh", "table"} {
		if strings.Contains(footer, suspended) {
			t.Errorf("Help footer advertises suspended dashboard action %q: %q", suspended, footer)
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

func TestHelpTextIncludesAboutRouteBurn(t *testing.T) {
	help := HelpText(plainStyles())
	for _, want := range []string{
		"About RouteBurn",
		"Version",
		"dev",
		"MIT license",
		"https://github.com/acevedomiguel/routeburn",
		"Miguel Angel Acevedo",
		"Cloud Architect and DevOps Engineer",
		"Hong Kong",
		"https://acevedomiguel.com/contact",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help about section is missing %q:\n%s", want, help)
		}
	}
}

func TestHelpOverlayPreservesTerminalGeometry(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, styles := range []Styles{plainStyles(), NewStyles(true)} {
		for _, size := range []struct{ width, height int }{{60, 16}, {72, 24}, {100, 40}, {140, 60}} {
			opts := ViewOptions{
				Width: size.width, Height: size.height, Styles: styles,
				Now: fixedNow(), Version: "1.2.3", HelpVisible: true,
			}
			background := Render(&snap, opts)
			out := HelpOverlay(background, opts)
			lines := strings.Split(out, "\n")
			if got := len(lines); got != size.height {
				t.Errorf("%dx%d overlay rendered %d lines, want %d", size.width, size.height, got, size.height)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got > size.width {
					t.Errorf("%dx%d overlay line %d is %d columns wide", size.width, size.height, i, got)
				}
			}
			last := lines[len(lines)-1]
			if !strings.Contains(last, "scroll") {
				t.Errorf("%dx%d overlay footer is missing popup scrolling: %q", size.width, size.height, last)
			}
			for _, suspended := range []string{"refresh", "table"} {
				if strings.Contains(last, suspended) {
					t.Errorf("%dx%d overlay footer advertises suspended action %q: %q", size.width, size.height, suspended, last)
				}
			}
			for _, corner := range []string{"╭", "╮", "╰", "╯"} {
				if !strings.Contains(out, corner) {
					t.Errorf("%dx%d overlay is missing rounded corner %q", size.width, size.height, corner)
				}
			}
			if !styles.Enabled && strings.Contains(out, "\x1b[") {
				t.Errorf("%dx%d no-color overlay contains ANSI escapes", size.width, size.height)
			}
		}
	}
}

func TestHelpOverlayIsCentered(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	opts := ViewOptions{Width: 100, Height: 40, Styles: plainStyles(), Now: fixedNow(), Version: "1.2.3", HelpVisible: true}
	out := HelpOverlay(Render(&snap, opts), opts)
	lines := strings.Split(out, "\n")
	marker := "╭─ RouteBurn help"
	top := -1
	left := 0
	popupWidth := 0
	for i, line := range lines {
		start := strings.Index(line, marker)
		if start < 0 {
			continue
		}
		top = i
		left = lipgloss.Width(line[:start])
		right := strings.Index(line[start:], "╮")
		if right < 0 {
			t.Fatalf("Help popup has no top-right corner: %q", line)
		}
		popupWidth = lipgloss.Width(line[start : start+right+len("╮")])
		break
	}
	if top < 0 {
		t.Fatalf("Help popup title not found:\n%s", out)
	}
	rightMargin := opts.Width - left - popupWidth
	if difference := left - rightMargin; difference < -1 || difference > 1 {
		t.Errorf("Help popup is not horizontally centered: left=%d right=%d", left, rightMargin)
	}

	bottomBorder := strings.Repeat(" ", left) + "╰" + strings.Repeat("─", popupWidth-2) + "╯"
	bottom := -1
	for i := top + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], bottomBorder) {
			bottom = i
			break
		}
	}
	if bottom < 0 {
		t.Fatalf("Help popup bottom border not found:\n%s", out)
	}
	bottomMargin := opts.Height - bottom - 1
	if difference := top - bottomMargin; difference < -1 || difference > 1 {
		t.Errorf("Help popup is not vertically centered: top=%d bottom=%d", top, bottomMargin)
	}
}

func TestHelpOverlayClampsScrolling(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	opts := ViewOptions{Width: 60, Height: 16, Styles: plainStyles(), Now: fixedNow(), Version: "1.2.3", HelpVisible: true}
	background := Render(&snap, opts)
	top := HelpOverlay(background, opts)
	negative := opts
	negative.HelpScroll = -10
	if got := HelpOverlay(background, negative); got != top {
		t.Error("negative Help scrolling must clamp to the top")
	}
	bottom := opts
	bottom.HelpScroll = 10000
	out := HelpOverlay(background, bottom)
	if !strings.Contains(out, "https://acevedomiguel.com/contact") {
		t.Fatalf("excessive Help scrolling must clamp to the bottom:\n%s", out)
	}
}
