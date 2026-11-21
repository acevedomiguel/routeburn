package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
