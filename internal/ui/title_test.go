package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The title row matches Kimiburn's: centred, and marked synthetic in the demo.
func TestTitleRowIsCentredAndMarksTheDemo(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	for _, tc := range []struct {
		demo bool
		want string
	}{
		{false, "ROUTEBURN · OpenRouter credits and runway"},
		{true, "ROUTEBURN · SYNTHETIC DATA · no provider calls"},
	} {
		out := Render(&snap, ViewOptions{Width: 120, Height: 40, Styles: plainStyles(), Now: fixedNow(), Demo: tc.demo})
		first := strings.Split(out, "\n")[0]
		left := strings.Index(first, tc.want)
		if left < 0 {
			t.Fatalf("demo=%t title missing: %q", tc.demo, first)
		}
		right := lipgloss.Width(first) - left - lipgloss.Width(tc.want)
		if diff := left - right; diff < -1 || diff > 1 {
			t.Errorf("demo=%t title not centred: left=%d right=%d", tc.demo, left, right)
		}
	}
}

// The table's page size is measured from the rendered frame. It must come out
// the same with colour on, or PgUp/PgDn degrade to single-row steps.
func TestTableWindowIsTheSameWithColour(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	plain := TableWindow(&snap, ViewOptions{Width: 120, Height: 40, Styles: NewStyles(false), Now: fixedNow()})
	colored := TableWindow(&snap, ViewOptions{Width: 120, Height: 40, Styles: NewStyles(true), Now: fixedNow()})
	if plain < 10 {
		t.Fatalf("plain table window = %d, want a full page of days", plain)
	}
	if colored != plain {
		t.Fatalf("coloured table window = %d, plain = %d", colored, plain)
	}
}
