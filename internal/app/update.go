package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acevedomiguel/routeburn/internal/ui"
	"github.com/charmbracelet/bubbles/spinner"
)

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case windowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case cacheLoadedMsg:
		if msg.err == nil {
			stored := msg.snapshot
			m.snapshot = &stored
			m.cached = true
			m.cacheAge = msg.age
			m.state = StateReady
		}
		return m, nil

	case snapshotLoadedMsg:
		return m.handleSnapshot(msg)

	case tickMsg:
		if msg.gen != m.gen {
			// A newer batch superseded this timer.
			return m, nil
		}
		if cmd := m.refreshCommands(); cmd != nil {
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
		return m, nil

	case fatalErrorMsg:
		m.state = StateFatalError
		if msg.err != nil {
			m.errorBanner = msg.err.Error()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if !m.refreshing {
			// Stop animating once nothing is in flight.
			return m, nil
		}
		return m, cmd

	case noticeMsg:
		m.notice = msg.text
		return m, nil
	}

	return m, nil
}

// handleKey handles every key binding.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		m.cancelInFlight()
		return m, tea.Quit
	}

	if m.helpVisible {
		if key == "esc" || key == "?" || key == "q" {
			m.helpVisible = false
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "q":
		m.cancelInFlight()
		return m, tea.Quit
	case "?":
		m.helpVisible = true
		return m, nil
	case "esc":
		m.helpVisible = false
		m.scroll = 0
		return m, nil
	case "r":
		if ok, wait := m.RequestRefresh(); !ok {
			if wait > 0 {
				text := "Refresh available in " + formatWait(wait)
				return m, func() tea.Msg { return noticeMsg{text: text} }
			}
			return m, nil
		}
		m.notice = ""
		if cmd := m.refreshCommands(); cmd != nil {
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
		return m, nil
	case "j", "down":
		m.scroll++
		return m, nil
	case "k", "up":
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	case "pgdown":
		m.scroll += maxInt(m.height-4, 1)
		return m, nil
	case "pgup":
		m.scroll -= maxInt(m.height-4, 1)
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	}

	return m, nil
}

// handleSnapshot merges a fetch result, preserving usable data on failure.
func (m *Model) handleSnapshot(msg snapshotLoadedMsg) (tea.Model, tea.Cmd) {
	m.refreshing = false
	m.loading = false

	if msg.gen != 0 && msg.gen < m.gen {
		// An older response must never replace a newer snapshot.
		return m, m.scheduleTick()
	}
	if msg.gen > m.gen {
		m.gen = msg.gen
	}

	if msg.err != nil {
		m.errorBanner = "Refresh failed: " + credRedact(msg.err.Error()) + " - press r to retry"
		if m.snapshot == nil {
			m.state = StateFatalError
		}
		return m, m.scheduleTick()
	}

	snap := msg.snapshot
	if m.snapshot != nil {
		previous := *m.snapshot
		m.previous = &previous
	}
	m.snapshot = &snap
	m.cached = false
	m.errorBanner = ""
	m.notice = ""
	m.lastSuccess = m.now()
	m.state = StateReady
	if server := m.NextRefreshAllowedFromServer(); server.After(m.nextAllowed) {
		m.nextAllowed = server
	}
	return m, m.scheduleTick()
}

func formatWait(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	seconds := int(d / time.Second)
	if d%time.Second >= time.Second/2 {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}
	return itoa(seconds) + "s"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// View implements tea.Model. It is pure rendering.
func (m *Model) View() string {
	styles := ui.NewStyles(m.cfg.ColorsEnabled())

	if m.helpVisible {
		return ui.HelpTextAt(styles, m.width)
	}

	switch m.state {
	case StateSetup:
		return ui.SetupView(ui.SetupOptions{
			Width:  m.width,
			Height: m.height,
			Styles: styles,
			Banner: m.errorBanner,
		})
	case StateFatalError:
		return ui.FatalView(ui.FatalOptions{
			Width:  m.width,
			Height: m.height,
			Styles: styles,
			Reason: m.errorBanner,
		})
	}

	return ui.Render(m.snapshot, ui.ViewOptions{
		Width:      m.width,
		Height:     m.height,
		Styles:     styles,
		Now:        m.now(),
		Cached:     m.cached,
		CacheAge:   m.cacheAge,
		Banner:     m.errorBanner,
		Scroll:     m.scroll,
		Refreshing: m.refreshing,
		Spinner:    m.spinner.View(),
	})
}

// ColorsEnabled resolves the color setting for the model.
func (c Config) ColorsEnabled() bool { return !c.NoColor }

// ensure context stays used for future commands.
var _ = context.Background
