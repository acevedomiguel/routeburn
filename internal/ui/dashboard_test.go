package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/acevedomiguel/routeburn/internal/domain"
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

// The body is fitted to the frame at every legal size, so it never overflows and
// the footer never shows pagination dots. A long model list is clipped to fit
// rather than scrolling the whole dashboard off screen.
func TestFooterHasNoPaginationDotsWhenBodyFits(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	snap.Models = append(snap.Models,
		domain.ModelUsage{Model: "google/gemini-3-pro", CreditSpend: 0.9},
		domain.ModelUsage{Model: "meta/llama-4-8b", CreditSpend: 0.4},
		domain.ModelUsage{Model: "mistral/large-3", CreditSpend: 0.2},
	)
	for _, h := range []int{16, 20, 40} {
		for _, w := range []int{60, 80, 140} {
			out := Render(&snap, ViewOptions{Width: w, Height: h, Styles: plainStyles(), Now: fixedNow()})
			rows := strings.Split(out, "\n")
			last := rows[len(rows)-1]
			if strings.Contains(last, "●") || strings.Contains(last, "○") {
				t.Errorf("%dx%d: fitted body should not show pagination dots: %q", w, h, last)
			}
		}
	}
}

// Paginator is built from the bubbles paginator bubble, not hand-rolled.
func TestPaginatorUsesBubbles(t *testing.T) {
	p := NewPaginator(3, 1)
	if got := p.View(); got != "○●○" {
		t.Fatalf("paginator view = %q", got)
	}
}

// The model breakdown is a bordered panel with leader rows: short model names,
// dot leaders, and an inline meter. The vendor prefix is stripped.
func TestModelsCardShowsShortNames(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, w := range []int{80, 100, 140} {
		out := Render(&snap, ViewOptions{Width: w, Height: 40, Styles: plainStyles(), Now: fixedNow()})
		if !strings.Contains(out, "Cost by model") {
			t.Fatalf("width %d: model panel title missing", w)
		}
		if !strings.Contains(out, "claude-sonnet-4") {
			t.Errorf("width %d: top model missing from the panel", w)
		}
		if strings.Contains(out, "anthropic/claude-sonnet-4") {
			t.Errorf("width %d: vendor prefix should be stripped", w)
		}
	}
}

// --- issue #12: model panel and token split share a row ----------------------

// The shared row must not cost the dashboard its exact height contract: the
// body still fills the frame at every legal size, so the footer lands on the
// last row and no pagination dots appear.
func TestSharedRowKeepsTheExactHeightBudget(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, h := range []int{16, 20, 24, 26, 30, 40, 60} {
		for _, w := range []int{60, 80, 100, 140} {
			out := Render(&snap, ViewOptions{Width: w, Height: h, Styles: plainStyles(), Now: fixedNow()})
			if got := len(strings.Split(out, "\n")); got != h {
				t.Errorf("%dx%d rendered %d lines, want %d", w, h, got, h)
			}
			last := strings.Split(out, "\n")[h-1]
			if strings.Contains(last, "●") || strings.Contains(last, "○") {
				t.Errorf("%dx%d: body overflowed into pagination: %q", w, h, last)
			}
		}
	}
}

// Once `detail` is true (a 26-row body, i.e. a 28-row terminal) the token split
// is shown and it shares a row with the model panel: model on the left half,
// tokens on the right half, aligned with the credits/last-30 row above.
func TestTallDashboardPutsTokensBesideCostByModel(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, h := range []int{28, 30, 40, 60} {
		out := Render(&snap, ViewOptions{Width: 120, Height: h, Styles: plainStyles(), Now: fixedNow()})
		lines := strings.Split(out, "\n")

		modelsRow, modelsCol := panelCol(t, lines, "╭─┬ Cost by model")
		tokensRow, tokensCol := panelCol(t, lines, "╭─┬ Tokens")
		_, creditsCol := panelCol(t, lines, "╭─┬ Credits")
		_, last30Col := panelCol(t, lines, "╭─┬ Last 30 days")

		if modelsRow != tokensRow {
			t.Errorf("height %d: model row=%d token row=%d, want the same row", h, modelsRow, tokensRow)
		}
		if modelsCol != creditsCol {
			t.Errorf("height %d: model col=%d, want the left column %d", h, modelsCol, creditsCol)
		}
		if tokensCol != last30Col {
			t.Errorf("height %d: token col=%d, want the right column %d", h, tokensCol, last30Col)
		}
	}
}

// The switch is exactly the `detail` boundary: a 26-row body shows the split
// beside the model panel, a 25-row body drops it and gives the model panel the
// whole width. dashboardBody is called directly so the boundary is pinned
// without depending on how many rows the status line and footer consume.
func TestDetailBoundaryDecidesTheSharedRow(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	lay := Classify(120, 40)
	opts := ViewOptions{Styles: plainStyles(), Now: fixedNow()}

	tall := strings.Join(dashboardBody(&snap, opts, lay, 26), "\n")
	if !strings.Contains(tall, "╭─┬ Tokens") {
		t.Fatalf("a 26-row body must show the token split:\n%s", tall)
	}
	tallLines := strings.Split(tall, "\n")
	modelRow, modelCol := panelCol(t, tallLines, "╭─┬ Cost by model")
	tokenRow, tokenCol := panelCol(t, tallLines, "╭─┬ Tokens")
	if modelRow != tokenRow || modelCol >= tokenCol {
		t.Errorf("26-row body: model row/col=%d/%d token row/col=%d/%d, want side by side",
			modelRow, modelCol, tokenRow, tokenCol)
	}

	short := strings.Join(dashboardBody(&snap, opts, lay, 25), "\n")
	if strings.Contains(short, "╭─┬ Tokens") {
		t.Errorf("a 25-row body must drop the token split:\n%s", short)
	}
	if _, col := panelCol(t, strings.Split(short, "\n"), "╭─┬ Cost by model"); col != 0 {
		t.Errorf("25-row body: model col=%d, want full width at 0", col)
	}
}

// Below 26 rows there is no room for the split: it is dropped and the model
// panel spans the full width instead of sharing a row.
func TestShortDashboardDropsTokensAndSpansModelFullWidth(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	// 16 is excluded: two border rows, the credits block, a gap, the table panel
	// and its gap come to more than 14 body rows, so there is no room for the
	// model panel at all. That case is covered by TestTinyTerminalDropsTheModelPanel.
	for _, h := range []int{20, 24, 27} {
		out := Render(&snap, ViewOptions{Width: 120, Height: h, Styles: plainStyles(), Now: fixedNow()})
		if strings.Contains(out, "╭─┬ Tokens") {
			t.Errorf("height %d: token panel must be dropped", h)
		}
		lines := strings.Split(out, "\n")
		if _, col := panelCol(t, lines, "╭─┬ Cost by model"); col != 0 {
			t.Errorf("height %d: model panel col=%d, want full width at 0", h, col)
		}
	}
}

// The single-column arrangement keeps the original stacked order: the token
// split, then the sparkline, then the model breakdown.
func TestSingleColumnKeepsStackedOrder(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	lay := Classify(120, 40)
	lay.TwoColumn = false
	lines := dashboardBody(&snap, ViewOptions{Styles: plainStyles(), Now: fixedNow()}, lay, 40)
	body := strings.Join(lines, "\n")

	tokens := strings.Index(body, "╭─┬ Tokens")
	spark := strings.Index(body, "╭─┬ Daily spend")
	models := strings.Index(body, "╭─┬ Cost by model")
	if tokens < 0 || spark < 0 || models < 0 {
		t.Fatalf("missing panel: tokens=%d spark=%d models=%d\n%s", tokens, spark, models, body)
	}
	if !(tokens < spark && spark < models) {
		t.Errorf("stacked order broken: tokens=%d spark=%d models=%d", tokens, spark, models)
	}
	for _, l := range lines {
		if strings.Contains(l, "╭─┬ Tokens") && strings.Count(l, "╭") != 1 {
			t.Errorf("single-column token panel must be full width: %q", l)
		}
	}
}
