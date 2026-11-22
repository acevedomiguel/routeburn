package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap holds every binding the dashboard understands. It satisfies the
// help.KeyMap interface, so the bubbles help bubble renders the footer and the
// overlay from this single source of truth.
type KeyMap struct {
	Refresh  key.Binding
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Help     key.Binding
	Close    key.Binding
	Quit     key.Binding
	Tab      key.Binding
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
	return []key.Binding{k.Refresh, k.Down, k.Tab, k.Help, k.Quit}
}

// FullHelp is the grouped overlay. SetUp remains the same as the footer.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Refresh, k.Down, k.Tab, k.Help},
		{k.Up, k.PageUp, k.PageDown},
		{k.Close, k.Quit},
	}
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
