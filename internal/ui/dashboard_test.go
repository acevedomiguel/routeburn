package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// The dashboard must fill the alternative screen exactly: a header, a body, a
// status line, and a help bar pinned to the very last row. Anything shorter
// leaves the bar floating mid-screen.
func TestRenderFillsTheWholeScreen(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	s := plainStyles()
	for _, h := range []int{16, 20, 24, 30, 40, 60} {
		for _, w := range []int{60, 80, 100, 140} {
			out := Render(&snap, ViewOptions{Width: w, Height: h, Styles: s, Now: fixedNow()})
			if got := len(strings.Split(out, "\n")); got != h {
				t.Errorf("%dx%d rendered %d lines, want %d", w, h, got, h)
			}
		}
	}
}

func TestHelpBarSitsOnTheLastRow(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 100, Height: 30, Styles: plainStyles(), Now: fixedNow()})
	lines := strings.Split(out, "\n")
	last := lines[len(lines)-1]
	for _, want := range []string{"[r]", "[q]", "[?]"} {
		if !strings.Contains(last, want) {
			t.Errorf("last row %q is missing %q", last, want)
		}
	}
}

// The bottom bar must be a real bubble, not a hand-built string: narrow
// terminals have to truncate it instead of wrapping or overflowing.
func TestHelpBarFitsNarrowWidths(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for w := 60; w <= 120; w++ {
		out := Render(&snap, ViewOptions{Width: w, Height: 24, Styles: plainStyles(), Now: fixedNow()})
		for _, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Fatalf("width %d line is %d wide: %q", w, got, line)
			}
		}
	}
}

// The credits meter is the bubbles progress bar. Without a fill it renders no
// bar at all, so the presence of the empty rune proves the bubble is in use.
// The credits card shows the remaining balance only: the purchased total and
// the progress meter were removed at the user's request.
func TestCreditsCardShowsOnlyRemaining(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 100, Height: 30, Styles: plainStyles(), Now: fixedNow()})
	if !strings.Contains(out, "$42.10") {
		t.Fatalf("expected the remaining balance, got:\n%s", out)
	}
	for _, unwanted := range []string{"$100.00", "% remaining", "[#####"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("credits card must not show %q:\n%s", unwanted, out)
		}
	}
}

func TestKeyMapShortHelpOnlyListsImplementedKeys(t *testing.T) {
	km := NewKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Fatal("short help is empty")
	}
	for _, b := range short {
		if !b.Enabled() {
			t.Errorf("binding %q is disabled", b.Help().Key)
		}
		if b.Help().Key == "" || b.Help().Desc == "" {
			t.Errorf("binding %+v has no help text", b)
		}
	}
}

func TestKeyMapHasHelpOverlayGroups(t *testing.T) {
	km := NewKeyMap()
	groups := km.FullHelp()
	if len(groups) < 2 {
		t.Fatalf("full help needs grouped columns, got %d", len(groups))
	}
	seen := map[string]bool{}
	for _, g := range groups {
		for _, b := range g {
			seen[b.Help().Key] = true
		}
	}
	for _, want := range []string{"[r]", "[?]", "[q]", "[esc]"} {
		if !seen[want] {
			t.Errorf("full help is missing %q", want)
		}
	}
}

// Every key the app handles must be advertised, and nothing else.
func TestKeyMapMatchesHandledKeys(t *testing.T) {
	km := NewKeyMap()
	advertised := map[string]bool{}
	for _, g := range km.FullHelp() {
		for _, b := range g {
			for _, k := range b.Keys() {
				advertised[k] = true
			}
		}
	}
	for _, handled := range []string{"r", "j", "k", "up", "down", "pgup", "pgdown", "?", "esc", "q"} {
		if !advertised[handled] {
			t.Errorf("key %q is handled but not advertised", handled)
		}
	}
	for _, notHandled := range []string{"export", "settings", "analyze", "e", "s"} {
		if advertised[notHandled] {
			t.Errorf("key %q is advertised but not handled", notHandled)
		}
	}
}

func TestNoColorRenderHasNoEscapes(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	tp := t
	tp.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 100, Height: 30, Styles: NewStyles(false), Now: fixedNow()})
	if strings.Contains(out, "\x1b[") {
		t.Fatal("no-color render contains ANSI escapes")
	}
}

func TestFooterAndHelpUseTheKeyMap(t *testing.T) {
	s := plainStyles()
	footer := Footer(ViewOptions{Styles: s, Width: 100})
	if !strings.Contains(footer, "refresh") || !strings.Contains(footer, "quit") {
		t.Fatalf("footer = %q", footer)
	}
	help := HelpText(s)
	for _, want := range []string{"refresh", "scroll", "help", "quit"} {
		if !strings.Contains(help, want) {
			t.Errorf("help overlay is missing %q:\n%s", want, help)
		}
	}
}

// The menu bar must be flush with both edges, exactly like the bubbles list
// example, so it reads as a bar and not as loose text.
func TestFooterSpansTheFullWidth(t *testing.T) {
	s := plainStyles()
	for _, w := range []int{60, 80, 100, 140} {
		out := Footer(ViewOptions{Styles: s, Width: w})
		if got := lipgloss.Width(out); got != w {
			t.Errorf("footer at width %d measured %d columns", w, got)
		}
	}
}

// When the body overflows, the bar gains paginator dots showing where you are.
func TestFooterShowsPaginationDotsWhenOverflowing(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	tall := Render(&snap, ViewOptions{Width: 80, Height: 16, Styles: plainStyles(), Now: fixedNow()})
	last := strings.Split(tall, "\n")[15]
	if !strings.Contains(last, "●") && !strings.Contains(last, "○") {
		t.Errorf("overflowing layout has no paginator dots: %q", last)
	}

	big := Render(&snap, ViewOptions{Width: 140, Height: 60, Styles: plainStyles(), Now: fixedNow()})
	rows := strings.Split(big, "\n")
	if dots := strings.Contains(rows[len(rows)-1], "●"); dots {
		t.Error("single-screen layout should not show pagination dots")
	}
}

// Paginator is built from the bubbles paginator bubble, not hand-rolled.
func TestPaginatorUsesBubbles(t *testing.T) {
	p := NewPaginator(3, 1)
	if got := p.View(); got != "○●○" {
		t.Fatalf("paginator view = %q", got)
	}
}

// The model breakdown is a bubbles table: it renders a header row and keeps
// the spend column intact on every width.
func TestModelsCardUsesBubblesTable(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, w := range []int{80, 100, 140} {
		out := Render(&snap, ViewOptions{Width: w, Height: 40, Styles: plainStyles(), Now: fixedNow()})
		if !strings.Contains(out, "Model") || !strings.Contains(out, "Spend") {
			t.Fatalf("width %d: model table header missing", w)
		}
		if !strings.Contains(out, "anthropic/claude-sonnet-4") {
			t.Errorf("width %d: top model missing from the table", w)
		}
	}
}
