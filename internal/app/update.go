package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/spinner"
	"github.com/acevedomiguel/routeburn/internal/ui"
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

	case tea.KeyPressMsg:
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
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		// Esc dismisses the overlay when it is open, but with the dashboard in
		// view it is the same as q: it closes the app.
		m.cancelInFlight()
		return m, tea.Quit
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
	case "tab":
		// Tab hands the scroll keys to the daily table and back. The body keeps
		// its own offset, so switching focus never loses a scroll position.
		m.tableFocused = !m.tableFocused
		return m, nil
	case "j", "down":
		if m.tableFocused {
			return m.moveTable(1)
		}
		m.scroll++
		return m, nil
	case "k", "up":
		if m.tableFocused {
			return m.moveTable(-1)
		}
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	case "pgdown":
		if m.tableFocused {
			return m.moveTable(ui.TableWindow(m.snapshot, m.viewOptions()))
		}
		m.scroll += maxInt(m.height-4, 1)
		return m, nil
	case "pgup":
		if m.tableFocused {
			return m.moveTable(-ui.TableWindow(m.snapshot, m.viewOptions()))
		}
		m.scroll -= maxInt(m.height-4, 1)
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	}

	return m, nil
}

// moveTable scrolls the daily table by delta days, clamped to the series. It
// stops one row short of the end so the pinned total row stays meaningful.
func (m *Model) moveTable(delta int) (tea.Model, tea.Cmd) {
	if m.snapshot == nil {
		return m, nil
	}
	rows := ui.TableWindow(m.snapshot, m.viewOptions())
	maxOffset := maxInt(len(m.snapshot.Daily)-rows, 0)
	next := m.tableOffset + delta
	if next < 0 {
		next = 0
	}
	if next > maxOffset {
		next = maxOffset
	}
	m.tableOffset = next
	return m, nil
}

// viewOptions is the struct Render builds, reused wherever the view geometry
// has to be recomputed outside a render pass.
func (m *Model) viewOptions() ui.ViewOptions {
	return ui.ViewOptions{
		Width:        m.width,
		Height:       m.height,
		Styles:       ui.NewStyles(m.cfg.ColorsEnabled()),
		Now:          m.now(),
		Cached:       m.cached,
		CacheAge:     m.cacheAge,
		Banner:       m.errorBanner,
		HelpVisible:  m.helpVisible,
		Scroll:       m.scroll,
		Refreshing:   m.refreshing,
		TableOffset:  m.tableOffset,
		TableFocused: m.tableFocused,
	}
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

// View implements tea.Model. The alternate screen is requested on the view
// itself; bubbletea v2 dropped the tea.WithAltScreen option.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render produces the frame. It is pure rendering.
func (m *Model) render() string {
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

	opts := m.viewOptions()
	opts.Styles = styles
	opts.Spinner = m.spinner.View()
	return ui.Render(m.snapshot, opts)
}

// ColorsEnabled resolves the color setting for the model.
func (c Config) ColorsEnabled() bool { return !c.NoColor }

// ensure context stays used for future commands.
var _ = context.Background
