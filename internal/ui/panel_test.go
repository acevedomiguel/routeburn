package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Every panel row must be exactly width columns wide: the frame only lines up
// if the geometry is exact.
func TestPanelIsExactWidthAndHeight(t *testing.T) {
	s := plainStyles()
	for _, w := range []int{20, 40, 61, 100} {
		for _, h := range []int{2, 3, 7} {
			rows := panel("Credits", []string{"Remaining ····· $18.43"}, w, h, s)
			if len(rows) != h {
				t.Fatalf("%dx%d panel has %d rows, want %d", w, h, len(rows), h)
			}
			for i, row := range rows {
				if got := lipgloss.Width(row); got != w {
					t.Errorf("%dx%d row %d is %d wide, want %d: %q", w, h, i, got, w, row)
				}
			}
		}
	}
}

// The title is inlaid in the top border, like btop's panel headers. The title
// is delimited by short downward ticks, not by full-height brackets.
func TestPanelInlaysTitleInTheTopBorder(t *testing.T) {
	rows := panel("Credits", nil, 40, 3, plainStyles())
	top := rows[0]
	if !strings.HasPrefix(top, "╭─┬") {
		t.Errorf("top border = %q", top)
	}
	if !strings.Contains(top, " Credits ") {
		t.Errorf("top border must inlay the title: %q", top)
	}
	if !strings.Contains(top, "┬") {
		t.Errorf("title must be delimited by downward ticks: %q", top)
	}
	if strings.Contains(top, "┤") || strings.Contains(top, "├") {
		t.Errorf("title must not use full-height brackets: %q", top)
	}
	if !strings.HasSuffix(top, "╮") {
		t.Errorf("top border must close: %q", top)
	}
	if !strings.HasPrefix(rows[len(rows)-1], "╰") {
		t.Errorf("bottom border = %q", rows[len(rows)-1])
	}
}

// A panel taller than its content pads with blank rows rather than leaving the
// frame short.
func TestPanelPadsToRequestedHeight(t *testing.T) {
	rows := panel("Credits", []string{"one"}, 40, 6, plainStyles())
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6", len(rows))
	}
	for i := 2; i < 5; i++ {
		if strings.TrimSpace(strings.Trim(rows[i], "│")) != "" {
			t.Errorf("row %d should be blank: %q", i, rows[i])
		}
	}
}

// A row longer than the inner width is clipped, never allowed to break the frame.
func TestPanelClipsLongRows(t *testing.T) {
	long := strings.Repeat("x", 200)
	rows := panel("Credits", []string{long}, 40, 3, plainStyles())
	for _, row := range rows {
		if got := lipgloss.Width(row); got != 40 {
			t.Fatalf("row is %d wide: %q", got, row)
		}
	}
}

// leader puts the label left and the value flush right, with dots between.
func TestLeaderAlignsValueRight(t *testing.T) {
	s := plainStyles()
	for _, w := range []int{20, 30, 45} {
		got := leader("Remaining", "$18.43", w, s)
		if width := lipgloss.Width(got); width != w {
			t.Errorf("width %d: leader is %d wide: %q", w, width, got)
		}
		if !strings.HasPrefix(got, "Remaining") {
			t.Errorf("width %d: label is not first: %q", w, got)
		}
		if !strings.HasSuffix(got, "$18.43") {
			t.Errorf("width %d: value is not last: %q", w, got)
		}
		if !strings.Contains(got, "·") {
			t.Errorf("width %d: no dot leader: %q", w, got)
		}
	}
}

// When the row is too narrow for dots, both halves stay readable and the row
// still fits.
func TestLeaderDegradesWhenTooNarrow(t *testing.T) {
	s := plainStyles()
	got := leader("Remaining", "$18.43", 16, s)
	if width := lipgloss.Width(got); width > 16 {
		t.Errorf("row overflows (%d columns): %q", width, got)
	}
	if !strings.Contains(got, "Remaining") || !strings.Contains(got, "$18.43") {
		t.Errorf("row lost content: %q", got)
	}

	// Below the combined length the row must clip rather than overflow.
	tiny := leader("Remaining", "$18.43", 12, s)
	if width := lipgloss.Width(tiny); width > 12 {
		t.Errorf("clipped row overflows (%d columns): %q", width, tiny)
	}
}

// meter fills proportionally: half a bar for half a ratio, and the same width
// regardless of the ratio.
func TestMeterFillsProportionally(t *testing.T) {
	s := plainStyles()
	for _, ratio := range []float64{0, 0.25, 0.5, 1} {
		got := meter(ratio, 10, s)
		if width := lipgloss.Width(got); width != 10 {
			t.Errorf("ratio %v: meter is %d wide, want 10: %q", ratio, width, got)
		}
	}
	half := meter(0.5, 10, s)
	if strings.Count(half, "▇") != 5 {
		t.Errorf("half meter has %d filled: %q", strings.Count(half, "▇"), half)
	}
	if strings.Count(meter(1, 10, s), "▇") != 10 {
		t.Error("full meter must be fully filled")
	}
	if strings.Count(meter(0, 10, s), "▇") != 0 {
		t.Error("empty meter must have no fill")
	}
}

// joinColumns pads shorter blocks with blank lines so every row lines up.
func TestJoinColumnsAlignsRows(t *testing.T) {
	left := []string{"aaaa", "bbbb", "cccc"}
	right := []string{"1111", "2222"}
	got := joinColumns(left, right)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	for i, row := range got {
		if !strings.HasPrefix(row, "aaaa") && !strings.HasPrefix(row, "bbbb") && !strings.HasPrefix(row, "cccc") {
			t.Errorf("row %d lost its left block: %q", i, row)
		}
	}
	if !strings.Contains(got[2], "cccc") {
		t.Errorf("last row = %q", got[2])
	}
}

// The status row carries the freshness text and no app name: the outer frame is
// gone, so nothing repeats the wordmark inside the app.
func TestStatusLineCarriesStatusWithoutWordmark(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	got := StatusLine(&snap, ViewOptions{Styles: plainStyles(), Now: fixedNow()}, 60)
	if width := lipgloss.Width(got); width != 60 {
		t.Errorf("status line is %d wide, want 60: %q", width, got)
	}
	if !strings.Contains(got, "Updated 15:42") {
		t.Errorf("status line must carry the status: %q", got)
	}
	if strings.Contains(got, "RouteBurn") {
		t.Errorf("status line must not repeat the app name: %q", got)
	}
	if strings.Contains(got, "╭") || strings.Contains(got, "│") {
		t.Errorf("status line must not draw a frame border: %q", got)
	}
}
