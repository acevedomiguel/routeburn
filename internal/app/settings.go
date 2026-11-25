package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/routeburn/internal/credentials"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// keyUpdateTimeout bounds the single validation read a key replacement
// performs, matching setup's timeout so the two paths behave the same.
const keyUpdateTimeout = 20 * time.Second

// maxKeyInputLength caps the hidden key buffer so a stray paste cannot grow it
// without bound.
const maxKeyInputLength = 4096

// DefaultIntervalOptions are the automatic-refresh intervals the Settings
// overlay offers, in ascending order. The renderer owns the list; this is a
// re-export for the CLI wiring.
func DefaultIntervalOptions() []time.Duration {
	return ui.DefaultIntervalOptions()
}

// intervalIndexFor returns the option index equal to interval, or -1 when the
// running interval is a custom value that is not one of the options.
func intervalIndexFor(interval time.Duration, options []time.Duration) int {
	for i, option := range options {
		if option == interval {
			return i
		}
	}
	return -1
}

// nearestIntervalIndex is the option index closest to interval, used as the
// starting point when a custom interval is not itself a menu option.
func nearestIntervalIndex(interval time.Duration, options []time.Duration) int {
	best, bestDelta := 0, time.Duration(-1)
	for i, option := range options {
		delta := interval - option
		if delta < 0 {
			delta = -delta
		}
		if bestDelta < 0 || delta < bestDelta {
			best, bestDelta = i, delta
		}
	}
	return best
}

// openSettings shows the Settings overlay over the current screen, remembering
// where to return when it closes.
func (m *Model) openSettings() {
	m.settingsVisible = true
	m.settingsSavedScroll = m.scroll
	m.settingsFocus = 0
	m.settingsIndex = intervalIndexFor(m.cfg.RefreshInterval, m.intervalOptions)
	// keySaving is deliberately left alone: a check already in flight still
	// owns the row, and reopening the overlay must not unlock a second attempt
	// that would race the first to the credential store.
	m.keyEditing, m.keyInput = false, ""
	m.settingsMsg, m.settingsErr = "", false
}

// closeSettings hides the overlay and restores the dashboard scroll position.
func (m *Model) closeSettings() {
	m.settingsVisible = false
	m.scroll = m.settingsSavedScroll
}

// keyAvailable reports whether in-app key replacement can run at all.
func (m *Model) keyAvailable() bool {
	return !m.demo && !m.envKey &&
		m.validateKey != nil && m.saveKey != nil && m.rebuildAPI != nil
}

// settingsOptions builds the pure render description for the overlay.
func (m *Model) settingsOptions() ui.SettingsOptions {
	return ui.SettingsOptions{
		Width: m.width, Height: m.height, Styles: ui.NewStyles(m.cfg.ColorsEnabled()),
		IntervalOptions: m.intervalOptions, IntervalIndex: m.settingsIndex, Interval: m.cfg.RefreshInterval,
		IntervalPinned: m.intervalPinned, IntervalSaved: m.setInterval != nil,
		Focus:        m.settingsFocus,
		KeyAvailable: m.keyAvailable(), EnvKey: m.envKey,
		Editing: m.keyEditing, KeyLength: len([]rune(m.keyInput)), Saving: m.keySaving,
		Message: m.settingsMsg, MessageIsError: m.settingsErr,
	}
}

// handleSettingsKey processes a key while the Settings overlay is open. While a
// key is being typed all printable input goes to the hidden buffer.
func (m *Model) handleSettingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.keyEditing {
		return m.handleKeyInput(msg)
	}
	keys := ui.NewSettingsKeyMap()
	switch {
	case key.Matches(msg, keys.Quit):
		m.cancelInFlight()
		return m, tea.Quit
	case key.Matches(msg, keys.Close):
		m.closeSettings()
		return m, nil
	case key.Matches(msg, keys.Move):
		if msg.String() == "up" || msg.String() == "k" {
			m.settingsFocus = 0
		} else {
			m.settingsFocus = 1
		}
		return m, nil
	case key.Matches(msg, keys.Change):
		if msg.String() == "left" || msg.String() == "h" {
			return m.changeInterval(-1)
		}
		return m.changeInterval(1)
	case msg.Code >= '1' && msg.Code <= '9':
		return m.chooseInterval(int(msg.Code - '1'))
	case key.Matches(msg, keys.Edit):
		// Only one attempt may be in flight. Each attempt persists the key
		// from its own goroutine, so a second concurrent attempt could write
		// the older key last and outlive the newer one in the store.
		if m.settingsFocus == 1 && m.keyAvailable() && !m.keySaving {
			m.keyEditing, m.keyInput = true, ""
			m.settingsMsg, m.settingsErr = "", false
		}
		return m, nil
	}
	return m, nil
}

// handleKeyInput collects the hidden key, then validates and saves it on Enter.
// Esc abandons the entry; every other printable key is key content, so the
// overlay's own letters (s, q) are not treated as commands here.
func (m *Model) handleKeyInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keys := ui.NewSettingsKeyMap()
	switch {
	case key.Matches(msg, keys.Quit):
		m.cancelInFlight()
		return m, tea.Quit
	case msg.Code == tea.KeyEscape:
		m.keyEditing, m.keyInput = false, ""
		m.settingsMsg, m.settingsErr = "", false
		return m, nil
	case msg.Code == tea.KeyEnter:
		return m.submitKey()
	case msg.Code == tea.KeyBackspace:
		if runes := []rune(m.keyInput); len(runes) > 0 {
			m.keyInput = string(runes[:len(runes)-1])
		}
		return m, nil
	case msg.Text != "" && len(m.keyInput)+len(msg.Text) <= maxKeyInputLength:
		m.keyInput += msg.Text
		return m, nil
	}
	return m, nil
}

// changeInterval moves the interval selection by delta and applies it. The
// interval row is the only one with a left/right control, so it responds even
// while the key row holds focus.
func (m *Model) changeInterval(delta int) (tea.Model, tea.Cmd) {
	if len(m.intervalOptions) == 0 {
		return m, nil
	}
	idx := m.settingsIndex
	if idx < 0 {
		idx = nearestIntervalIndex(m.cfg.RefreshInterval, m.intervalOptions)
	} else {
		idx += delta
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.intervalOptions) {
		idx = len(m.intervalOptions) - 1
	}
	return m.chooseInterval(idx)
}

// chooseInterval applies the option at idx: the session interval changes
// immediately and, when a persister is wired, the choice is saved.
func (m *Model) chooseInterval(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.intervalOptions) {
		return m, nil
	}
	chosen := m.intervalOptions[idx]
	m.settingsIndex = idx
	m.cfg.RefreshInterval = chosen

	var err error
	if m.setInterval != nil {
		err = m.setInterval(chosen)
	}
	switch {
	case err != nil:
		m.settingsMsg = "Applied for this session; could not save: " + credRedact(err.Error())
		m.settingsErr = true
	case m.intervalPinned:
		m.settingsMsg = "Applied. The external override still wins on restart."
		m.settingsErr = false
	case m.setInterval == nil:
		m.settingsMsg = "Applied for this session."
		m.settingsErr = false
	default:
		m.settingsMsg = "Saved to the configuration file."
		m.settingsErr = false
	}
	return m, m.scheduleTick()
}

// submitKey starts the asynchronous validate-then-save for the entered key.
// The buffer is cleared immediately so the secret is not retained in the model.
func (m *Model) submitKey() (tea.Model, tea.Cmd) {
	management := strings.TrimSpace(m.keyInput)
	m.keyInput = ""
	if management == "" {
		m.settingsMsg, m.settingsErr = "Type a management key, then press Enter.", true
		return m, nil
	}
	if !m.keyAvailable() {
		m.settingsMsg, m.settingsErr = "Key updates are unavailable in this mode.", true
		return m, nil
	}
	m.keyEditing, m.keySaving = false, true
	m.settingsMsg, m.settingsErr = "Checking the key…", false

	// Only the newest attempt may apply. The operator can submit again while a
	// check is in flight, and the older validation may well finish last.
	m.keyGen++
	gen := m.keyGen

	validate, save := m.validateKey, m.saveKey
	redactor := credentials.NewRedactor(management)
	return m, func() tea.Msg {
		validateCtx, cancelValidate := context.WithTimeout(context.Background(), keyUpdateTimeout)
		err := validate(validateCtx, management)
		cancelValidate()
		if err != nil {
			return keySavedMsg{gen: gen, err: errors.New(redactor.Redact(err.Error()))}
		}
		saveCtx, cancelSave := context.WithTimeout(context.Background(), keyUpdateTimeout)
		defer cancelSave()
		if err := save(saveCtx, management); err != nil {
			return keySavedMsg{gen: gen, err: errors.New(redactor.Redact(err.Error()))}
		}
		return keySavedMsg{gen: gen, key: management}
	}
}

// applyKeySaved reacts to a replacement key: a failure keeps the previous key
// and reports a redacted reason; a success rebuilds the client and forces an
// immediate read with the new credential.
func (m *Model) applyKeySaved(msg keySavedMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.keyGen {
		// A newer attempt is already in flight; its own result is authoritative.
		return m, nil
	}
	m.keySaving = false
	if msg.err != nil {
		m.settingsMsg = "Not saved: " + msg.err.Error()
		m.settingsErr = true
		return m, nil
	}

	// Rebuild the client so the next request carries the new key. A rebuild
	// failure is reported without discarding the stored key.
	api, err := m.rebuildAPI(msg.key, m.creds.APIKey)
	if err != nil {
		m.settingsMsg = "Key saved, but the client could not be rebuilt: " + credRedact(err.Error())
		m.settingsErr = true
		return m, nil
	}
	m.setAPI(api)
	m.creds.Management = msg.key
	m.settingsMsg, m.settingsErr = "Key saved. Refreshing with the new key…", false
	m.keyRefreshPending = true

	// A missing-key screen has served its purpose once a key is stored, so
	// closing the overlay returns to the dashboard rather than back to setup.
	if m.state == StateSetup || m.state == StateFatalError {
		m.state = StateReady
	}

	// Supersede any in-flight request made with the previous key and clear the
	// cadence so the new credential is exercised right away. A server that
	// still wants a slower pace re-imposes it through the next response's
	// NextRefreshAllowedAt.
	m.cancelInFlight()
	m.refreshing = false
	m.loading = false
	m.lastAttempt = time.Time{}
	m.nextAllowed = time.Time{}
	m.gen++
	if cmd := m.refreshCommands(); cmd != nil {
		return m, tea.Batch(cmd, m.spinner.Tick)
	}
	return m, m.scheduleTick()
}

// finishKeyRefresh replaces the "refreshing with the new key" notice with the
// forced read's actual outcome, so a failure is visible instead of the overlay
// claiming success indefinitely.
func (m *Model) finishKeyRefresh(err error) {
	if !m.keyRefreshPending {
		return
	}
	m.keyRefreshPending = false
	if err != nil {
		m.settingsMsg = "Key saved, but the refresh failed: " + credRedact(err.Error())
		m.settingsErr = true
		return
	}
	m.settingsMsg, m.settingsErr = "Key saved.", false
}
