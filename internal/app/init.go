package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Init implements tea.Model. It arms the startup refresh (or the setup
// screen's own first command).
func (m *Model) Init() tea.Cmd {
	if m.state == StateSetup {
		return nil
	}

	// Start a background refresh immediately; cached data is already visible.
	return tea.Batch(m.refreshCommands())
}

// StartRefresh exposes the refresh command for the CLI JSON path.
func (m *Model) StartRefresh() tea.Cmd { return m.refreshCommands() }

// ensure time is used for the tick helper above.
var _ = time.Second
