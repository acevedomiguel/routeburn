package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/spinner"
	"github.com/acevedomiguel/burnblocks/openrouter"
	"github.com/acevedomiguel/routeburn/internal/format"
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

	case tea.PasteMsg:
		return m.handlePaste(msg)

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

	case keySavedMsg:
		return m.applyKeySaved(msg)

	case tickMsg:
		if msg.gen != m.tickGen {
			// A newer timer superseded this one.
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

	if m.settingsVisible {
		return m.handleSettingsKey(msg)
	}

	if m.helpVisible {
		switch key {
		case "esc", "?", "q":
			m.helpVisible = false
			m.helpScroll = 0
		case "j", "down":
			m.scrollHelp(1)
		case "k", "up":
			m.scrollHelp(-1)
		case "pgdown":
			m.scrollHelp(maxInt(m.height-5, 1))
		case "pgup":
			m.scrollHelp(-maxInt(m.height-5, 1))
		}
		return m, nil
	}

	switch key {
	case "q":
		m.cancelInFlight()
		return m, tea.Quit
	case "?":
		m.helpVisible = true
		m.helpScroll = 0
		return m, nil
	case "s":
		m.openSettings()
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

// scrollHelp moves only the popup viewport, clamped to its wrapped content.
func (m *Model) scrollHelp(delta int) {
	limit := ui.HelpScrollLimit(m.viewOptions())
	current := m.helpScroll
	if current > limit {
		current = limit
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	if next > limit {
		next = limit
	}
	m.helpScroll = next
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
		Version:      m.version,
		HelpVisible:  m.helpVisible,
		HelpScroll:   m.helpScroll,
		Scroll:       m.scroll,
		Refreshing:   m.refreshing,
		TableOffset:  m.tableOffset,
		TableFocused: m.tableFocused,
		Demo:         m.demo,
	}
}

// handleSnapshot merges a fetch result, preserving usable data on failure.
func (m *Model) handleSnapshot(msg snapshotLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != 0 && msg.gen < m.gen {
		// An older response must never replace a newer snapshot, and it must
		// not clear the newer request's in-flight state.
		return m, m.scheduleTick()
	}
	m.refreshing = false
	m.loading = false
	if msg.gen > m.gen {
		m.gen = msg.gen
	}
	m.finishKeyRefresh(msg.err)

	// The cache is written here rather than in the fetch goroutine, because
	// only this point knows whether the response won. A superseded response
	// that saved itself would leave the next launch rendering stale data.
	if msg.err == nil && m.cache != nil && !m.cfg.NoCache {
		// A cache failure must never fail the refresh.
		_ = m.cache.Save(msg.snapshot)
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
	v.WindowTitle = m.windowTitle()
	return v
}

// windowTitle puts the headline figures in the terminal tab, the same way
// Kimiburn does: remaining credit, plus the runway when it is a number of
// days. Without a snapshot, or on the setup and error screens, it is just the
// app name.
func (m *Model) windowTitle() string {
	if m.state != StateReady || m.snapshot == nil {
		return "routeburn"
	}
	title := "routeburn · " + format.FormatCredits(m.snapshot.Account.Remaining) + " left"
	if rw := m.snapshot.Runway; rw.Basis == openrouter.BasisFinite {
		title += " · runway " + format.FormatDaysRemaining(rw)
	}
	return title
}

// render produces the frame. It is pure rendering.
func (m *Model) render() string {
	styles := ui.NewStyles(m.cfg.ColorsEnabled())
	opts := m.viewOptions()
	opts.Styles = styles
	opts.Spinner = m.spinner.View()

	var background string
	switch m.state {
	case StateSetup:
		background = ui.SetupView(ui.SetupOptions{
			Width:  m.width,
			Height: m.height,
			Styles: styles,
			Banner: m.errorBanner,
		})
	case StateFatalError:
		background = ui.FatalView(ui.FatalOptions{
			Width:  m.width,
			Height: m.height,
			Styles: styles,
			Reason: m.errorBanner,
		})
	default:
		background = ui.Render(m.snapshot, opts)
	}

	if m.settingsVisible {
		return ui.SettingsOverlay(background, m.settingsOptions())
	}
	if m.helpVisible {
		return ui.HelpOverlay(background, opts)
	}
	return background
}

// ColorsEnabled resolves the color setting for the model.
func (c Config) ColorsEnabled() bool { return !c.NoColor }

// ensure context stays used for future commands.
var _ = context.Background
