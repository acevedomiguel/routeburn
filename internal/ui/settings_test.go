package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func blankBackground(width, height int) string {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = strings.Repeat(" ", width)
	}
	return strings.Join(lines, "\n")
}

func TestFormatIntervalOption(t *testing.T) {
	cases := map[time.Duration]string{
		0:                "off",
		10 * time.Second: "10s",
		30 * time.Second: "30s",
		time.Minute:      "1m",
		5 * time.Minute:  "5m",
		10 * time.Minute: "10m",
	}
	for d, want := range cases {
		if got := FormatIntervalOption(d); got != want {
			t.Errorf("FormatIntervalOption(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestSettingsContentShowsOptionsSelectionAndKeyRow(t *testing.T) {
	opts := SettingsOptions{
		Width: 80, Height: 24, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(),
		IntervalIndex:   2, Interval: time.Minute,
		KeyAvailable: true,
	}
	content := strings.Join(SettingsContent(opts), "\n")
	for _, want := range []string{"Refresh interval", "10s", "30s", "[1m]", "5m", "10m", "Management key", "Enter to update key"} {
		if !strings.Contains(content, want) {
			t.Errorf("settings content lacks %q:\n%s", want, content)
		}
	}

	// A custom interval shows no bracket and names the running value.
	custom := SettingsContent(SettingsOptions{
		Width: 80, Height: 24, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: -1, Interval: 45 * time.Second,
	})
	intervalRow := custom[1]
	if strings.Contains(intervalRow, "[") || strings.Contains(intervalRow, "]") {
		t.Errorf("a custom interval must not bracket an option: %q", intervalRow)
	}
	if !strings.Contains(strings.Join(custom, "\n"), "45s") {
		t.Errorf("a custom interval should be named:\n%s", strings.Join(custom, "\n"))
	}
}

func TestSettingsContentMarksFocusAndHidesKey(t *testing.T) {
	editing := strings.Join(SettingsContent(SettingsOptions{
		Width: 80, Height: 24, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 1, Interval: 30 * time.Second,
		KeyAvailable: true, Focus: 1, Editing: true, KeyLength: 5,
	}), "\n")
	if !strings.Contains(editing, "*****") {
		t.Errorf("an editing key should render a hidden placeholder:\n%s", editing)
	}
	if strings.Contains(editing, "secret") {
		t.Fatalf("the key value must never appear:\n%s", editing)
	}
}

// A key being checked explains why the row is locked, so the lock is not read
// as a broken control.
func TestSettingsContentExplainsTheSavingState(t *testing.T) {
	content := strings.Join(SettingsContent(SettingsOptions{
		Width: 80, Height: 24, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 1, Interval: 30 * time.Second,
		KeyAvailable: true, Focus: 1, Saving: true,
	}), "\n")
	for _, want := range []string{"checking the key", "unlocks when it finishes"} {
		if !strings.Contains(content, want) {
			t.Errorf("settings content lacks %q:\n%s", want, content)
		}
	}
}

// The overlay must preserve the terminal exactly: one line per row, never wider
// than the terminal, no ANSI escapes in no-color mode, and a visible title.
func TestSettingsOverlayKeepsExactGeometryAndNoColorPurity(t *testing.T) {
	states := []SettingsOptions{
		{IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 0, Interval: 10 * time.Second, KeyAvailable: true, Focus: 0},
		{IntervalOptions: DefaultIntervalOptions(), IntervalIndex: -1, Interval: 45 * time.Second, KeyAvailable: true, Focus: 1, Editing: true, KeyLength: 24, Message: "Checking the key…"},
		{IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 4, Interval: 10 * time.Minute, KeyAvailable: false, EnvKey: true, Message: "Not saved: a very long redacted reason that must wrap without overflowing", MessageIsError: true},
	}
	for _, size := range []struct{ w, h int }{{60, 16}, {72, 24}, {80, 24}, {100, 40}, {140, 60}} {
		for _, noColor := range []bool{false, true} {
			st := NewStyles(!noColor)
			for i, state := range states {
				state.Width, state.Height, state.Styles = size.w, size.h, st
				frame := SettingsOverlay(blankBackground(size.w, size.h), state)
				if noColor && strings.Contains(frame, "\x1b[") {
					t.Errorf("%dx%d state %d no-color settings gained ANSI", size.w, size.h, i)
				}
				lines := strings.Split(frame, "\n")
				if len(lines) != size.h {
					t.Errorf("%dx%d state %d: %d rows", size.w, size.h, i, len(lines))
				}
				for row, line := range lines {
					if got := lipgloss.Width(line); got > size.w {
						t.Errorf("%dx%d state %d row %d width=%d, want at most %d", size.w, size.h, i, row, got, size.w)
					}
				}
				if !strings.Contains(frame, "RouteBurn settings") {
					t.Errorf("%dx%d state %d settings overlay lacks its title", size.w, size.h, i)
				}
			}
		}
	}
}

// The overlay's footer advertises only the keys that work in the current mode.
func TestSettingsOverlayFooterTracksEditingMode(t *testing.T) {
	base := SettingsOptions{
		Width: 100, Height: 40, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 1, Interval: 30 * time.Second,
		KeyAvailable: true,
	}
	browsing := SettingsOverlay(blankBackground(100, 40), base)
	for _, want := range []string{"select", "interval", "update key", "close"} {
		if !strings.Contains(browsing, want) {
			t.Errorf("browsing footer missing %q:\n%s", want, browsing)
		}
	}
	// The full hint row must survive untruncated at the common 80-column size.
	if narrow := SettingsOverlay(blankBackground(80, 24), base); strings.Contains(narrow, "…") {
		t.Errorf("browsing footer is truncated at 80 columns:\n%s", narrow)
	}

	base.Editing = true
	editing := SettingsOverlay(blankBackground(100, 40), base)
	for _, want := range []string{"save", "cancel"} {
		if !strings.Contains(editing, want) {
			t.Errorf("editing footer missing %q:\n%s", want, editing)
		}
	}
	if strings.Contains(editing, "update key") {
		t.Errorf("editing footer must not advertise the update-key action:\n%s", editing)
	}

	// An unavailable key row must not be advertised either.
	base = SettingsOptions{
		Width: 100, Height: 40, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 1, Interval: 30 * time.Second,
		KeyAvailable: false,
	}
	unavailable := SettingsOverlay(blankBackground(100, 40), base)
	if strings.Contains(unavailable, "update key") {
		t.Errorf("footer must not advertise a disabled key row:\n%s", unavailable)
	}
	if !strings.Contains(unavailable, "interval") {
		t.Errorf("footer must keep the interval hint:\n%s", unavailable)
	}
}

// The overlay sits above the dashboard without moving its footer off the last
// row, so the dashboard keeps its geometry while Settings is open.
func TestSettingsOverlaySitsAboveTheDashboard(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	opts := ViewOptions{Width: 100, Height: 40, Styles: NewStyles(false), Now: fixedNow()}
	background := Render(&snap, opts)
	frame := SettingsOverlay(background, SettingsOptions{
		Width: 100, Height: 40, Styles: NewStyles(false),
		IntervalOptions: DefaultIntervalOptions(), IntervalIndex: 1, Interval: 30 * time.Second,
		KeyAvailable: true,
	})
	lines := strings.Split(frame, "\n")
	if len(lines) != 40 {
		t.Fatalf("overlay rendered %d rows, want 40", len(lines))
	}
	if !strings.Contains(frame, "RouteBurn settings") {
		t.Fatal("the overlay is not visible above the dashboard")
	}
	if !strings.Contains(frame, "[r]") {
		t.Error("the dashboard footer should still be present behind the overlay")
	}
}
