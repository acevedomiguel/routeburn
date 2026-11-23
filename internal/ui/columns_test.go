package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// The model card spans the full content width, so the spend column must survive
// the wide layout. Truncating it to "..." makes the card useless.
func TestModelCardKeepsSpendValuesInWideLayout(t *testing.T) {
	snap := SampleSnapshot()

	for _, width := range []int{100, 110, 120, 140} {
		out := Render(&snap, ViewOptions{
			Width:  width,
			Height: 40,
			Styles: NewStyles(false),
			Now:    snap.GeneratedAt,
		})

		for _, want := range []string{"$7.21", "$3.82", "$1.31"} {
			if !strings.Contains(out, want) {
				t.Errorf("width %d: model card missing spend %q:\n%s", width, want, out)
			}
		}
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "anthropic/claude-sonnet-4") && strings.HasSuffix(trimmed, "...") {
				t.Errorf("width %d: spend column was truncated away: %q", width, line)
			}
		}
	}
}

// Token summary stays readable next to the model list.
func TestModelCardTokenSummaryIsVisible(t *testing.T) {
	snap := SampleSnapshot()
	out := Render(&snap, ViewOptions{Width: 100, Height: 40, Styles: NewStyles(false), Now: snap.GeneratedAt})

	for _, want := range []string{"Tokens", "8.4M", "Cache hit", "18.4%"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// Every rendered line fits the configured width, including the model rows.
func TestNoRenderedLineExceedsWidth(t *testing.T) {
	snap := SampleSnapshot()
	for width := 60; width <= 140; width += 4 {
		out := Render(&snap, ViewOptions{Width: width, Height: 40, Styles: NewStyles(false), Now: snap.GeneratedAt})
		for _, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line is %d wide: %q", width, w, line)
			}
		}
	}
}

// --- issue #12: "Cost by model" and "Tokens" share a row ---------------------

// panelCol returns the row index of the first line containing needle and the
// display column at which that panel's box starts. On the shared row two panels
// share a line, so the column is found by walking back from the title to the
// nearest top-left corner rather than taking the line's first one. The column is
// measured in cells, not bytes: the box glyphs are multi-byte and a byte offset
// would not line up with what the reader sees. It fails the test when the panel
// is absent.
func panelCol(t *testing.T, lines []string, needle string) (row, col int) {
	t.Helper()
	for i, l := range lines {
		at := strings.Index(l, needle)
		if at < 0 {
			continue
		}
		corner := strings.LastIndex(l[:at+len("╭")], "╭")
		if corner < 0 {
			t.Fatalf("no top-left corner before %q on row %d: %q", needle, i, l)
		}
		return i, lipgloss.Width(l[:corner])
	}
	t.Fatalf("panel %q not found in:\n%s", needle, strings.Join(lines, "\n"))
	return -1, -1
}

// tailFromCol returns the part of line starting at display column col. It is the
// display-aware counterpart to slicing a byte string, which would split a
// multi-byte box glyph in half.
func tailFromCol(line string, col int) string {
	at := 0
	for i, r := range line {
		if at >= col {
			return line[i:]
		}
		at += lipgloss.Width(string(r))
	}
	return ""
}

// bottomBorderCols returns the display columns of every bottom-left corner on a
// line, so a test can ask whether two side-by-side panels end on the same row.
func bottomBorderCols(line string) []int {
	var cols []int
	col := 0
	for _, r := range line {
		if r == '╰' {
			cols = append(cols, col)
		}
		col += lipgloss.Width(string(r))
	}
	return cols
}

// panelBottom returns the row index of the bottom border belonging to the panel
// whose top-left corner is at startCol. It walks down until the border glyph
// appears in that column.
func panelBottom(t *testing.T, lines []string, startRow, startCol int) int {
	t.Helper()
	for i := startRow; i < len(lines); i++ {
		for _, c := range bottomBorderCols(lines[i]) {
			if c == startCol {
				return i
			}
		}
	}
	t.Fatalf("no bottom border at column %d below row %d", startCol, startRow)
	return -1
}

// issue #12: on a tall terminal the model panel and the token split share one
// row, placed between the credits/last-30 row and the daily table. Pinned at
// 90 columns: at 100+ the sidebar arrangement takes over (see
// TestWideLayoutPutsTheTableBesideThePanels).
func TestCostByModelAndTokensShareARow(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	lines := strings.Split(out, "\n")

	creditsRow, creditsCol := panelCol(t, lines, "╭─┬ Credits")
	modelsRow, modelsCol := panelCol(t, lines, "╭─┬ Cost by model")
	tokensRow, tokensCol := panelCol(t, lines, "╭─┬ Tokens")
	sparkRow, _ := panelCol(t, lines, "╭─┬ Daily spend")
	last30Row, last30Col := panelCol(t, lines, "╭─┬ Last 30 days")
	if last30Row != creditsRow {
		t.Errorf("credits/last-30 must share the top row: credits=%d last30=%d", creditsRow, last30Row)
	}

	// Vertical order: the shared row sits below the credits/last-30 row and
	// above the sparkline.
	if !(creditsRow < modelsRow && modelsRow < sparkRow) {
		t.Errorf("shared row must sit between credits and sparkline: credits=%d shared=%d spark=%d",
			creditsRow, modelsRow, sparkRow)
	}
	if tokensRow != modelsRow {
		t.Errorf("Tokens row = %d, want the shared row %d:\n%s", tokensRow, modelsRow, out)
	}
	// The model panel is the left column (same column as credits); the token
	// panel is the right column (same column as last 30 days).
	if modelsCol != creditsCol {
		t.Errorf("Cost by model column = %d, want the left column %d", modelsCol, creditsCol)
	}
	if tokensCol != last30Col {
		t.Errorf("Tokens column = %d, want the right column %d", tokensCol, last30Col)
	}
	if modelsCol >= tokensCol {
		t.Errorf("Cost by model (%d) must sit left of Tokens (%d)", modelsCol, tokensCol)
	}
}

// Both panels on the shared row end on the same terminal row: the shorter panel
// is padded with blank rows so its bottom border lines up with the taller
// panel's, exactly like the credits/last-30 pattern.
func TestSharedRowPanelsEndOnTheSameRow(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	// 90 columns: at 100+ the sidebar stacks these panels instead.
	out := Render(&snap, ViewOptions{Width: 90, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	lines := strings.Split(out, "\n")

	modelsRow, modelsCol := panelCol(t, lines, "╭─┬ Cost by model")
	tokensRow, tokensCol := panelCol(t, lines, "╭─┬ Tokens")
	if modelsRow != tokensRow {
		t.Fatalf("panels are not on the same row: model=%d token=%d", modelsRow, tokensRow)
	}

	modelEnd := panelBottom(t, lines, modelsRow, modelsCol)
	tokenEnd := panelBottom(t, lines, tokensRow, tokensCol)
	if modelEnd != tokenEnd {
		t.Errorf("shared-row panels end on different rows: model=%d token=%d\n%s", modelEnd, tokenEnd, out)
	}
}

// A short terminal (height < 26) has no room for the token split: the panel is
// skipped and the model breakdown stays full width.
func TestShortTerminalDropsTokensAndKeepsModelFullWidth(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 120, Height: 20, Styles: plainStyles(), Now: fixedNow()})
	if strings.Contains(out, "╭─┬ Tokens") {
		t.Errorf("short terminal must not draw the token panel:\n%s", out)
	}
	lines := strings.Split(out, "\n")
	_, sparkCol := panelCol(t, lines, "╭─┬ Daily spend")
	_, modelCol := panelCol(t, lines, "╭─┬ Cost by model")
	if modelCol != sparkCol {
		t.Errorf("model panel column = %d, want full width at %d", modelCol, sparkCol)
	}
}

// The model panel sits in a half-width column on the shared row, so a long
// model name must truncate instead of overflowing the column and pushing the
// token panel off the terminal.
func TestModelPanelFitsTheNarrowColumnOnTheSharedRow(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	snap.Models = []domain.ModelUsage{
		{Model: "anthropic/a-very-long-model-name-that-cannot-fit", CreditSpend: 9.5},
		{Model: "openai/another-extremely-long-model-identifier", CreditSpend: 4.5},
	}
	for _, w := range []int{60, 80, 100, 140} {
		out := Render(&snap, ViewOptions{Width: w, Height: 40, Styles: plainStyles(), Now: fixedNow()})
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Fatalf("width %d line %d is %d wide: %q", w, i, got, line)
			}
		}
		if !strings.Contains(out, "Tokens") {
			t.Errorf("width %d: token panel lost to the model panel:\n%s", w, out)
		}
	}
}

// The token split shares the narrow right column on a small terminal. Every
// value the API reported must still be readable there: a panel too narrow for
// two pairs per row must fall back to one pair per row rather than clipping the
// value off the end of the line.
func TestTokenPanelKeepsEveryValueInTheNarrowColumn(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 60, Height: 28, Styles: plainStyles(), Now: fixedNow()})
	lines := strings.Split(out, "\n")

	_, tokensCol := panelCol(t, lines, "╭─┬ Tokens")
	for _, want := range []string{"7.1M", "1.3M", "1.5M", "480K"} {
		found := false
		for _, l := range lines {
			if strings.Contains(tailFromCol(l, tokensCol), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("token split lost %q in the narrow column:\n%s", want, out)
		}
	}
	for _, l := range lines {
		if strings.Contains(tailFromCol(l, tokensCol), "...") {
			t.Errorf("token split clipped a value in the narrow column: %q", l)
		}
	}
}

// The model panel is half as wide on the shared row, so it shows the top spender
// alone rather than a column of truncated names; the panel still carries its
// spend and share.
func TestNarrowModelColumnShowsTheTopSpender(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	out := Render(&snap, ViewOptions{Width: 60, Height: 40, Styles: plainStyles(), Now: fixedNow()})
	lines := strings.Split(out, "\n")

	_, modelsCol := panelCol(t, lines, "╭─┬ Cost by model")
	found := false
	for _, l := range lines {
		tail := tailFromCol(l, modelsCol)
		if strings.Contains(tail, "$7.21") {
			found = true
			if !strings.Contains(tail, "%") {
				t.Errorf("top spender lost its share: %q", l)
			}
		}
	}
	if !found {
		t.Errorf("top spender missing from the narrow model column:\n%s", out)
	}
}
