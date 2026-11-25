package ui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap holds every binding the dashboard understands. It satisfies the
// help.KeyMap interface, so the bubbles help bubble renders the footer and the
// overlay from this single source of truth.
type KeyMap struct {
	Refresh  key.Binding
	Settings key.Binding
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Help     key.Binding
	Close    key.Binding
	Quit     key.Binding
	Tab      key.Binding
}

// OverlayKeyMap describes only the keys available while Help is open.
type OverlayKeyMap struct {
	Scroll key.Binding
	Page   key.Binding
	Close  key.Binding
	Quit   key.Binding
}

// NewOverlayKeyMap builds the contextual footer shown beneath the Help popup.
func NewOverlayKeyMap() OverlayKeyMap {
	return OverlayKeyMap{
		Scroll: key.NewBinding(
			key.WithKeys("j", "k", "up", "down"),
			key.WithHelp("[↑↓]", "scroll"),
		),
		Page: key.NewBinding(
			key.WithKeys("pgup", "pgdown"),
			key.WithHelp("[pgup/pgdn]", "page"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "?", "q"),
			key.WithHelp("[esc/?/q]", "close"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("[ctrl+c]", "quit"),
		),
	}
}

// ShortHelp is the one-line Help popup footer.
func (k OverlayKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Scroll, k.Page, k.Close, k.Quit}
}

// FullHelp satisfies help.KeyMap; the popup footer uses ShortHelp.
func (k OverlayKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// NewKeyMap builds the default bindings. Only keys the app actually handles are
// listed, so the footer can never advertise a feature that does not exist.
func NewKeyMap() KeyMap {
	return KeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("[tab]", "table"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("[r]", "refresh"),
		),
		Settings: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("[s]", "settings"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("[↑↓]", "scroll"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("[↑]", "up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("[pgdn]", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("[pgup]", "page up"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("[?]", "help"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("[esc]", "quit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("[q]", "quit"),
		),
	}
}

// ShortHelp is the one-line footer, in render order.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Settings, k.Down, k.Tab, k.Help, k.Quit}
}

// FullHelp is the grouped overlay. SetUp remains the same as the footer.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Refresh, k.Settings, k.Down, k.Tab, k.Help},
		{k.Up, k.PageUp, k.PageDown},
		{k.Close, k.Quit},
	}
}

// SettingsKeyMap describes the keys available inside the Settings overlay.
type SettingsKeyMap struct {
	Move   key.Binding
	Change key.Binding
	Edit   key.Binding
	Close  key.Binding
	Quit   key.Binding
}

// NewSettingsKeyMap builds the Settings overlay bindings.
func NewSettingsKeyMap() SettingsKeyMap {
	return SettingsKeyMap{
		Move: key.NewBinding(
			key.WithKeys("up", "down", "k", "j"),
			key.WithHelp("[↑↓/j/k]", "select"),
		),
		Change: key.NewBinding(
			key.WithKeys("left", "right", "h", "l"),
			key.WithHelp("[←/→]", "interval"),
		),
		Edit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("[enter]", "update key"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "q", "s"),
			key.WithHelp("[esc]", "close"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("[ctrl+c]", "quit"),
		),
	}
}

// SettingsFooter is the Settings overlay's key hints. While a key is being
// typed only the keys that apply in that mode are listed, and the update-key
// hint is dropped when the row is unavailable, so the footer never advertises a
// disabled action.
func (k SettingsKeyMap) SettingsFooter(editing, keyAvailable bool) [][]key.Binding {
	if editing {
		return [][]key.Binding{{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("[enter]", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("[esc]", "cancel")),
			k.Quit,
		}}
	}
	// Browsing lists Close rather than Quit: Esc, q, and s all close the
	// overlay, so the hint covers both leaving and quitting, and the shorter
	// row still fits the popup on a 60-column terminal.
	row := []key.Binding{k.Move, k.Change}
	if keyAvailable {
		row = append(row, k.Edit)
	}
	return [][]key.Binding{append(row, k.Close)}
}

// Bindings is the flat, ordered list kept for callers that just want labels.
func Bindings() []Binding {
	km := NewKeyMap()
	var out []Binding
	for _, group := range km.FullHelp() {
		for _, b := range group {
			out = append(out, Binding{
				Keys:   b.Keys(),
				Hint:   b.Help().Key,
				Action: b.Help().Desc,
			})
		}
	}
	return out
}
