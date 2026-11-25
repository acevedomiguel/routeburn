package ui

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// DefaultIntervalOptions are the automatic-refresh intervals the Settings
// overlay offers, in ascending order. Zero (manual refresh only) stays
// available through --refresh and the configuration file, but is not a menu
// choice.
func DefaultIntervalOptions() []time.Duration {
	return []time.Duration{10 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute, 10 * time.Minute}
}

// SettingsOptions describes the Settings overlay's contents. Everything is
// plain data so rendering stays pure and testable.
type SettingsOptions struct {
	Width, Height int
	Styles        Styles

	// IntervalOptions are the selectable intervals in display order.
	// IntervalIndex selects one, or -1 when the running interval is a custom
	// value that is not a menu option.
	IntervalOptions []time.Duration
	IntervalIndex   int
	Interval        time.Duration
	// IntervalPinned is true when a flag or environment variable outranks the
	// saved file, so a change applies now but not on the next start.
	IntervalPinned bool
	// IntervalSaved is true when a change is persisted to a configuration file.
	IntervalSaved bool

	// Focus is 0 for the interval row and 1 for the management-key row.
	Focus int

	KeyAvailable bool
	EnvKey       bool
	Editing      bool
	KeyLength    int
	Saving       bool

	Message        string
	MessageIsError bool
}

// SettingsContent is the Settings overlay body as styled lines.
func SettingsContent(opts SettingsOptions) []string {
	st := opts.Styles
	lines := []string{
		st.Title.Render("Refresh interval"),
		intervalRow(opts),
		st.Muted.Render(intervalNote(opts)),
		"",
		st.Title.Render("Management key"),
		keyRow(opts),
	}
	if note := keyNote(opts); note != "" {
		lines = append(lines, st.Muted.Render(note))
	}
	if opts.Message != "" {
		style := st.Success
		if opts.MessageIsError {
			style = st.Error
		}
		lines = append(lines, "", style.Render(opts.Message))
	}
	return lines
}

// intervalRow draws the interval options, bracketing the selected one and
// marking the row when it holds focus.
func intervalRow(opts SettingsOptions) string {
	parts := make([]string, 0, len(opts.IntervalOptions))
	for i, d := range opts.IntervalOptions {
		label := FormatIntervalOption(d)
		if i == opts.IntervalIndex {
			label = "[" + label + "]"
		} else {
			label = " " + label + " "
		}
		parts = append(parts, label)
	}
	row := strings.Join(parts, " ")
	if row == "" {
		row = "none"
	}
	return focusPrefix(opts.Focus == 0) + row
}

// intervalNote explains the current interval selection: a custom value, an
// externally pinned one, a session-only one, or one saved to disk.
func intervalNote(opts SettingsOptions) string {
	switch {
	case opts.IntervalIndex < 0:
		return "current: " + FormatIntervalOption(opts.Interval) + " (custom)"
	case opts.IntervalPinned:
		return "--refresh or ROUTEBURN_REFRESH_INTERVAL overrides the saved file on restart"
	case !opts.IntervalSaved:
		return "applies to this session only"
	default:
		return "saved to the configuration file"
	}
}

// keyRow draws the management-key row: a prompt to begin, the hidden input, or
// a progress marker while the key is checked.
func keyRow(opts SettingsOptions) string {
	prefix := focusPrefix(opts.Focus == 1)
	st := opts.Styles
	switch {
	case !opts.KeyAvailable:
		return prefix + st.Muted.Render("[ Update key · unavailable ]")
	case opts.Editing:
		return prefix + "New management key: " + strings.Repeat("*", opts.KeyLength) + "_"
	case opts.Saving:
		return prefix + st.Muted.Render("[ checking the key… ]")
	default:
		return prefix + st.Key.Render("[ Enter to update key ]")
	}
}

// keyNote explains the key row's current state.
func keyNote(opts SettingsOptions) string {
	switch {
	case opts.EnvKey:
		return "ROUTEBURN_MANAGEMENT_KEY supplies the key; unset it to store one here."
	case !opts.KeyAvailable:
		return "Key updates are unavailable in this mode."
	case opts.Saving:
		return "The key is being checked with one live read; the row unlocks when it finishes."
	case opts.Editing:
		return "The key is hidden. Enter validates and saves it; Esc cancels."
	default:
		return "Replacing the key checks it with one live read before saving."
	}
}

func focusPrefix(focused bool) string {
	if focused {
		return "› "
	}
	return "  "
}

// settingsHelp adapts a group of bindings to the help bubble.
type settingsHelp struct{ groups [][]key.Binding }

func (h settingsHelp) ShortHelp() []key.Binding {
	if len(h.groups) == 0 {
		return nil
	}
	return h.groups[0]
}

func (h settingsHelp) FullHelp() [][]key.Binding { return h.groups }

// SettingsOverlay centres the Settings panel above the dashboard frame, sized
// to the terminal exactly like the Help overlay. The canvas keeps the original
// terminal dimensions, so the dashboard and its pinned footer keep their rows.
func SettingsOverlay(background string, opts SettingsOptions) string {
	st := opts.Styles
	width := opts.Width
	if width <= 0 {
		width = 80
	}
	height := opts.Height
	if height <= 0 {
		height = 40
	}

	popupWidth := minInt(84, width-4)
	if popupWidth < 4 {
		popupWidth = 4
	}
	textWidth := maxInt(popupWidth-4, 1)

	var wrapped []string
	for _, line := range SettingsContent(opts) {
		wrapped = append(wrapped, strings.Split(lipgloss.Wrap(line, textWidth, " "), "\n")...)
	}
	footer := NewHelp(st, textWidth).View(
		settingsHelp{groups: NewSettingsKeyMap().SettingsFooter(opts.Editing, opts.KeyAvailable)})
	wrapped = append(wrapped, "", footer)

	maxHeight := height - 2
	if maxHeight < 3 {
		maxHeight = 3
	}
	popupHeight := minInt(len(wrapped)+2, maxHeight)
	if popupHeight < 2 {
		popupHeight = 2
	}
	if len(wrapped) > popupHeight-2 {
		wrapped = wrapped[:popupHeight-2]
	}

	popup := strings.Join(panel("RouteBurn settings", wrapped, popupWidth, popupHeight, st), "\n")

	x := maxInt((width-popupWidth)/2, 0)
	y := maxInt((height-popupHeight)/2, 0)
	compositor := lipgloss.NewCompositor(
		lipgloss.NewLayer(background),
		lipgloss.NewLayer(popup).X(x).Y(y).Z(1),
	)
	lines := strings.Split(lipgloss.NewCanvas(width, height).Compose(compositor).Render(), "\n")
	for i, line := range lines {
		lines[i] = padRight(line, width)
	}
	return strings.Join(lines, "\n")
}

// FormatIntervalOption renders an automatic-refresh interval the way the
// Settings menu spells it: whole minutes as "Nm", whole seconds as "Ns", and a
// disabled cadence as "off".
func FormatIntervalOption(d time.Duration) string {
	switch {
	case d <= 0:
		return "off"
	case d%time.Minute == 0:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	return strconv.Itoa(int(d/time.Second)) + "s"
}
