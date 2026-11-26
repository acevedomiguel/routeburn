package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/burnblocks/openrouter"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// settingsWiring supplies the optional Settings side effects. Zero values keep
// the feature session-only or unavailable.
type settingsWiring struct {
	setInterval    func(time.Duration) error
	intervalPinned bool
	validateKey    func(context.Context, string) error
	saveKey        func(context.Context, string) error
	rebuildAPI     func(management, apiKey string) (openrouter.API, error)
	envKey         bool
	demo           bool
}

// settingsModel is a ready dashboard with the Settings overlay wired.
func settingsModel(t *testing.T, wiring settingsWiring) *Model {
	t.Helper()
	m := testModel(t, testModelOptions{})
	m.intervalOptions = ui.DefaultIntervalOptions()
	m.settingsIndex = intervalIndexFor(m.cfg.RefreshInterval, m.intervalOptions)
	m.setInterval = wiring.setInterval
	m.intervalPinned = wiring.intervalPinned
	m.validateKey = wiring.validateKey
	m.saveKey = wiring.saveKey
	m.rebuildAPI = wiring.rebuildAPI
	m.envKey = wiring.envKey
	m.demo = wiring.demo
	return m
}

// feed delivers any message to the model and returns its command.
func feed(m *Model, msg tea.Msg) tea.Cmd {
	_, cmd := m.Update(msg)
	return cmd
}

func pressKey(m *Model, msg tea.KeyPressMsg) tea.Cmd { return feed(m, msg) }

func typeKeys(m *Model, text string) {
	for _, r := range text {
		pressKey(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func openSettings(t *testing.T, m *Model) {
	t.Helper()
	pressKey(m, tea.KeyPressMsg{Code: 's', Text: "s"})
	if !m.settingsVisible {
		t.Fatal("s must open the settings overlay")
	}
}

// noopKey is a pairing of accept-everything key hooks for tests that only need
// the row to be available.
func noopKeyHooks() (func(context.Context, string) error, func(context.Context, string) error, func(string, string) (openrouter.API, error)) {
	return func(context.Context, string) error { return nil },
		func(context.Context, string) error { return nil },
		func(string, string) (openrouter.API, error) { return &fakeAPI{}, nil }
}

func TestSettingsOpensAndClosesRestoringScroll(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	m.scroll = 3
	openSettings(t, m)
	if m.settingsSavedScroll != 3 {
		t.Fatalf("saved scroll = %d, want 3", m.settingsSavedScroll)
	}

	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.settingsVisible || m.scroll != 3 {
		t.Fatalf("close: visible=%v scroll=%d, want the dashboard at scroll 3", m.settingsVisible, m.scroll)
	}

	// q and s close the overlay too, matching the Help overlay's keys.
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if m.settingsVisible {
		t.Fatal("q did not close Settings")
	}
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.settingsVisible {
		t.Fatal("s did not close Settings")
	}
}

func TestSettingsKeyEditingTreatsLettersAsContent(t *testing.T) {
	validate, save, rebuild := noopKeyHooks()
	m := settingsModel(t, settingsWiring{validateKey: validate, saveKey: save, rebuildAPI: rebuild})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	// s and q are overlay commands outside editing, but key content inside it.
	typeKeys(m, "sq")
	if !m.settingsVisible || !m.keyEditing {
		t.Fatalf("visible=%v editing=%v, want the overlay still editing", m.settingsVisible, m.keyEditing)
	}
	if m.keyInput != "sq" {
		t.Fatalf("keyInput = %q, want \"sq\"", m.keyInput)
	}
}

func TestSettingsIntervalSelectionPersistsAndRearms(t *testing.T) {
	var saved []time.Duration
	m := settingsModel(t, settingsWiring{setInterval: func(d time.Duration) error {
		saved = append(saved, d)
		return nil
	}})
	openSettings(t, m)
	if m.settingsIndex != 0 {
		t.Fatalf("initial index = %d, want 0 for 10s", m.settingsIndex)
	}

	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.cfg.RefreshInterval != 30*time.Second {
		t.Fatalf("interval = %v, want 30s", m.cfg.RefreshInterval)
	}
	if len(saved) != 1 || saved[0] != 30*time.Second {
		t.Fatalf("saved = %v, want one 30s write", saved)
	}
	if cmd == nil {
		t.Fatal("changing the interval did not re-arm the timer")
	}
	if m.settingsErr || m.settingsMsg == "" {
		t.Fatalf("message=%q err=%v", m.settingsMsg, m.settingsErr)
	}

	// A number key selects directly, and left clamps at the smallest option.
	pressKey(m, tea.KeyPressMsg{Code: '4', Text: "4"})
	if m.cfg.RefreshInterval != 5*time.Minute {
		t.Fatalf("number key interval = %v, want 5m", m.cfg.RefreshInterval)
	}
	for i := 0; i < 8; i++ {
		pressKey(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	}
	if m.cfg.RefreshInterval != 10*time.Second {
		t.Fatalf("clamped interval = %v, want 10s", m.cfg.RefreshInterval)
	}
}

func TestSettingsIntervalWithoutPersisterIsSessionOnly(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.cfg.RefreshInterval != 30*time.Second {
		t.Fatalf("interval = %v, want 30s", m.cfg.RefreshInterval)
	}
	if !strings.Contains(m.settingsMsg, "session") {
		t.Fatalf("message = %q, want a session-only note", m.settingsMsg)
	}
}

func TestSettingsIntervalSaveFailureStillAppliesForTheSession(t *testing.T) {
	m := settingsModel(t, settingsWiring{setInterval: func(time.Duration) error {
		return errors.New("config: parse /tmp/config.yaml: boom")
	}})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.cfg.RefreshInterval != 30*time.Second {
		t.Fatalf("interval = %v, want the session change to stand", m.cfg.RefreshInterval)
	}
	if !m.settingsErr || !strings.Contains(m.settingsMsg, "could not save") {
		t.Fatalf("message=%q err=%v", m.settingsMsg, m.settingsErr)
	}
}

func TestSettingsIntervalPinnedReportsTheExternalOverride(t *testing.T) {
	m := settingsModel(t, settingsWiring{
		setInterval:    func(time.Duration) error { return nil },
		intervalPinned: true,
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if !strings.Contains(m.settingsMsg, "restart") {
		t.Fatalf("message = %q, want a note that the override wins on restart", m.settingsMsg)
	}
	if m.settingsErr {
		t.Fatal("an external override is not an error")
	}
}

func TestSettingsIntervalChangeSupersedesTheRunningTimer(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	openSettings(t, m)
	stale := m.tickGen

	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("the interval change did not re-arm the timer")
	}
	if m.tickGen == stale {
		t.Fatal("the interval change did not start a new timer chain")
	}
	if next := feed(m, tickMsg{gen: stale}); next != nil {
		t.Fatal("a superseded tick still started work")
	}
	if next := feed(m, tickMsg{gen: m.tickGen}); next == nil {
		t.Fatal("the current tick did not start an attempt")
	}
}

func TestSettingsKeyEntryIsHiddenValidatedAndSaved(t *testing.T) {
	const key = "sk-or-v1-synthetic-key-value"
	var validated, stored string
	var rebuiltManagement, rebuiltAPIKey string
	rebuilt := &fakeAPI{}

	m := settingsModel(t, settingsWiring{
		validateKey: func(_ context.Context, got string) error { validated = got; return nil },
		saveKey:     func(_ context.Context, got string) error { stored = got; return nil },
		rebuildAPI: func(management, apiKey string) (openrouter.API, error) {
			rebuiltManagement, rebuiltAPIKey = management, apiKey
			return rebuilt, nil
		},
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.settingsFocus != 1 {
		t.Fatalf("focus = %d, want the key row", m.settingsFocus)
	}
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.keyEditing {
		t.Fatal("enter on the key row did not start editing")
	}

	typeKeys(m, key)
	if view := m.View().Content; strings.Contains(view, key) {
		t.Fatalf("the typed key appeared in the view:\n%s", view)
	}
	if bullets := strings.Repeat("*", len([]rune(key))); !strings.Contains(m.View().Content, bullets) {
		t.Fatalf("the hidden input did not render %d placeholders", len([]rune(key)))
	}

	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the key returned no command")
	}
	if m.keyInput != "" {
		t.Fatalf("the key buffer was not cleared: %q", m.keyInput)
	}
	if !m.keySaving {
		t.Fatal("the model should mark the key as being checked")
	}

	feed(m, cmd().(keySavedMsg))
	if validated != key || stored != key {
		t.Fatalf("validated=%q stored=%q, want the submitted key", validated, stored)
	}
	// The regular API key slot must be carried over unchanged.
	if rebuiltManagement != key || rebuiltAPIKey != m.creds.APIKey {
		t.Fatalf("rebuilt with management=%q apiKey=%q, want the new key and the unchanged API key", rebuiltManagement, rebuiltAPIKey)
	}
	if m.currentAPI() != openrouter.API(rebuilt) {
		t.Error("the model did not adopt the rebuilt client")
	}
	if m.creds.Management != key {
		t.Errorf("model creds = %q, want the new key", m.creds.Management)
	}
	if m.keySaving || m.settingsErr {
		t.Fatalf("after save: saving=%v err=%v msg=%q", m.keySaving, m.settingsErr, m.settingsMsg)
	}
}

func TestSettingsKeyFailureKeepsPreviousKeyAndRedacts(t *testing.T) {
	const key = "sk-or-v1-synthetic-secret-value"
	saved := false
	rebuilt := false
	m := settingsModel(t, settingsWiring{
		validateKey: func(_ context.Context, got string) error {
			return errors.New("server rejected " + got)
		},
		saveKey: func(context.Context, string) error { saved = true; return nil },
		rebuildAPI: func(string, string) (openrouter.API, error) {
			rebuilt = true
			return &fakeAPI{}, nil
		},
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	typeKeys(m, key)
	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the key returned no command")
	}
	feed(m, cmd().(keySavedMsg))

	if saved {
		t.Fatal("a failed validation must not save the key")
	}
	if rebuilt {
		t.Fatal("a failed validation must not rebuild the client")
	}
	if !m.settingsErr {
		t.Fatal("a failed validation should be reported as an error")
	}
	if strings.Contains(m.settingsMsg, key) {
		t.Fatalf("the failure message leaked the key: %q", m.settingsMsg)
	}
	if !strings.Contains(m.settingsMsg, "[redacted]") {
		t.Fatalf("the failure message should mark the redaction: %q", m.settingsMsg)
	}
	if m.creds.Management != "sk-or-v1-management" {
		t.Errorf("model creds changed on a failed validation: %q", m.creds.Management)
	}
}

func TestSettingsKeyRebuildFailureReportsWithoutLosingTheKey(t *testing.T) {
	const key = "sk-or-v1-synthetic-rebuild-value"
	stored := ""
	m := settingsModel(t, settingsWiring{
		validateKey: func(context.Context, string) error { return nil },
		saveKey:     func(_ context.Context, got string) error { stored = got; return nil },
		rebuildAPI: func(string, string) (openrouter.API, error) {
			return nil, errors.New("openrouter: refusing non-TLS base URL")
		},
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	typeKeys(m, key)
	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	feed(m, cmd().(keySavedMsg))

	if stored != key {
		t.Fatalf("stored = %q, want the key to be persisted before the rebuild", stored)
	}
	if !m.settingsErr || !strings.Contains(m.settingsMsg, "could not be rebuilt") {
		t.Fatalf("message=%q err=%v", m.settingsMsg, m.settingsErr)
	}
}

func TestSettingsEnvironmentKeyDisablesKeyUpdate(t *testing.T) {
	validate, save, rebuild := noopKeyHooks()
	m := settingsModel(t, settingsWiring{
		validateKey: validate, saveKey: save, rebuildAPI: rebuild, envKey: true,
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.keyEditing {
		t.Fatal("editing must be disabled when ROUTEBURN_MANAGEMENT_KEY is set")
	}
	if view := m.View().Content; !strings.Contains(view, "ROUTEBURN_MANAGEMENT_KEY") {
		t.Fatalf("the overlay should explain the environment key:\n%s", view)
	}
}

func TestSettingsKeyUnavailableInDemoMode(t *testing.T) {
	validate, save, rebuild := noopKeyHooks()
	m := settingsModel(t, settingsWiring{
		validateKey: validate, saveKey: save, rebuildAPI: rebuild, demo: true,
	})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.keyEditing {
		t.Fatal("demo mode must not offer key replacement")
	}
	if view := m.View().Content; !strings.Contains(view, "unavailable") {
		t.Fatalf("the overlay should mark the key row unavailable:\n%s", view)
	}
}

func TestSettingsKeySaveClearsFailureStateAndForcesRefresh(t *testing.T) {
	validate, save, rebuild := noopKeyHooks()
	m := settingsModel(t, settingsWiring{validateKey: validate, saveKey: save, rebuildAPI: rebuild})
	m.state = StateSetup
	m.errorBanner = "no management key"
	m.nextAllowed = fixedNow().Add(time.Hour)
	m.cfg.RefreshInterval = 0
	openSettings(t, m)

	cmd := feed(m, keySavedMsg{key: "sk-or-v1-replacement"})
	if m.state != StateReady {
		t.Fatalf("state = %v, want ready after the key is stored", m.state)
	}
	if !m.refreshing {
		t.Fatal("a saved key did not start a refresh")
	}
	if cmd == nil {
		t.Fatal("a saved key returned no command")
	}
	if !m.nextAllowed.IsZero() {
		t.Errorf("server backoff was not cleared: %v", m.nextAllowed)
	}
}

// The "refreshing with the new key" notice must be replaced by the forced
// read's real outcome, so a failure is visible instead of the overlay claiming
// success indefinitely.
func TestKeyRefreshNoticeClearsAfterTheRefresh(t *testing.T) {
	validate, save, rebuild := noopKeyHooks()
	m := settingsModel(t, settingsWiring{validateKey: validate, saveKey: save, rebuildAPI: rebuild})
	openSettings(t, m)

	feed(m, keySavedMsg{gen: m.keyGen, key: "sk-or-v1-new"})
	if !strings.Contains(m.settingsMsg, "Refreshing") {
		t.Fatalf("message = %q, want the refreshing notice", m.settingsMsg)
	}
	feed(m, snapshotLoadedMsg{gen: m.gen, snapshot: ui.SampleSnapshotAt(fixedNow())})
	if m.settingsMsg != "Key saved." {
		t.Fatalf("message = %q, want the saved notice after a successful read", m.settingsMsg)
	}
	if m.settingsErr {
		t.Error("a successful forced read must not report an error")
	}

	// A failed forced read must say so, and redact the reason.
	feed(m, keySavedMsg{gen: m.keyGen, key: "sk-or-v1-new"})
	feed(m, snapshotLoadedMsg{gen: m.gen, err: errors.New("network unreachable for sk-or-v1-secret")})
	if !m.settingsErr || !strings.Contains(m.settingsMsg, "refresh failed") {
		t.Fatalf("message=%q err=%v, want a reported refresh failure", m.settingsMsg, m.settingsErr)
	}
	if strings.Contains(m.settingsMsg, "sk-or-v1-secret") {
		t.Fatalf("the failure notice leaked a key: %q", m.settingsMsg)
	}
}

// A background refresh updates the data behind an open overlay without closing
// it or discarding the operator's position.
func TestBackgroundRefreshKeepsSettingsOpen(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	openSettings(t, m)

	snap := ui.SampleSnapshotAt(fixedNow())
	feed(m, snapshotLoadedMsg{gen: 1, snapshot: snap})
	if !m.settingsVisible {
		t.Fatal("a background refresh must not close Settings")
	}
	if m.Snapshot() == nil {
		t.Fatal("the background refresh did not update the snapshot")
	}
	if m.State() != StateReady {
		t.Fatalf("state = %v, want ready", m.State())
	}
}

// A stale response from a superseded request must not clear the in-flight state
// of the request that replaced it.
func TestStaleResponseDoesNotClearTheNewerRequestState(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	m.gen = 5
	m.refreshing = true

	feed(m, snapshotLoadedMsg{gen: 4, snapshot: ui.SampleSnapshotAt(fixedNow())})
	if !m.refreshing {
		t.Fatal("a stale response cleared the newer request's in-flight state")
	}
	if m.Snapshot() != nil {
		t.Fatal("a stale response replaced the displayed snapshot")
	}
}

func TestSettingsIntervalCustomValueIsReported(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	m.cfg.RefreshInterval = 45 * time.Second
	openSettings(t, m)
	if m.settingsIndex != -1 {
		t.Fatalf("settingsIndex = %d, want -1 for a custom interval", m.settingsIndex)
	}
	if view := m.View().Content; !strings.Contains(view, "45s") {
		t.Fatalf("the overlay should name a custom interval:\n%s", view)
	}

	// A custom interval still shifts to a real option rather than failing.
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.settingsIndex < 0 {
		t.Fatal("a custom interval should land on the nearest menu option")
	}
}

// The API client is read by a fetch goroutine and swapped by Update when the
// management key is replaced, so the accessor pair must be synchronized. Run
// with -race to make this meaningful.
func TestAPIReplacementIsSynchronized(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 500; i++ {
			if api := m.currentAPI(); api == nil {
				t.Errorf("currentAPI returned nil")
				return
			}
		}
	}()
	for i := 0; i < 500; i++ {
		m.setAPI(&fakeAPI{})
	}
	<-done
}

// submitKeyAttempt drives one full key entry: focus the row, start editing,
// type the value, and submit. It returns the pending command.
func submitKeyAttempt(t *testing.T, m *Model, key string) tea.Cmd {
	t.Helper()
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.keyEditing {
		t.Fatal("enter on the key row did not start editing")
	}
	typeKeys(m, key)
	cmd := pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the key returned no command")
	}
	return cmd
}

// waitForCancel blocks until the model has registered a cancel function for an
// in-flight fetch.
func waitForCancel(t *testing.T, m *Model) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !m.cancelPending() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !m.cancelPending() {
		t.Fatal("the fetch never registered its cancel function")
	}
}

// Only one replacement may be in flight. Attempts persist the key from their
// own goroutine, so a second concurrent attempt could write the older key last
// and leave the store disagreeing with the session.
func TestKeyEntryIsLockedWhileACheckIsInFlight(t *testing.T) {
	var persisted []string
	m := settingsModel(t, settingsWiring{
		validateKey: func(context.Context, string) error { return nil },
		saveKey: func(_ context.Context, key string) error {
			persisted = append(persisted, key)
			return nil
		},
		rebuildAPI: func(string, string) (openrouter.API, error) { return &fakeAPI{}, nil },
	})
	openSettings(t, m)

	attempt := submitKeyAttempt(t, m, "sk-or-v1-only")
	if !m.keySaving {
		t.Fatal("submitting the key did not mark the check in flight")
	}

	// A second entry must not start while the first is being checked.
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.keyEditing {
		t.Fatal("a second key entry started while a check was in flight")
	}

	// Closing and reopening the overlay must not unlock it either.
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyDown})
	pressKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.keyEditing {
		t.Fatal("reopening the overlay bypassed the in-flight check")
	}

	// Exactly the single submitted key may ever reach the store.
	feed(m, attempt())
	if len(persisted) != 1 || persisted[0] != "sk-or-v1-only" {
		t.Fatalf("persisted = %v, want only the submitted key", persisted)
	}
	if m.keySaving {
		t.Fatal("the row should unlock once the check finishes")
	}
}

// A completion that a newer submission has already superseded must not touch
// the client or the credentials the model is using.
func TestSupersededKeyCompletionIsDiscarded(t *testing.T) {
	var rebuilt []string
	m := settingsModel(t, settingsWiring{
		validateKey: func(context.Context, string) error { return nil },
		saveKey:     func(context.Context, string) error { return nil },
		rebuildAPI: func(management, _ string) (openrouter.API, error) {
			rebuilt = append(rebuilt, management)
			return &fakeAPI{}, nil
		},
	})
	openSettings(t, m)

	stale := submitKeyAttempt(t, m, "sk-or-v1-superseded")
	// Simulate a newer submission having taken over the generation. The UI
	// blocks this while a check runs, so it is asserted as a guard.
	m.keyGen++

	feed(m, stale())
	if len(rebuilt) != 0 {
		t.Fatalf("rebuilt with %v, want a discarded superseded completion", rebuilt)
	}
	if m.creds.Management != "sk-or-v1-management" {
		t.Fatalf("management key = %q, want the original", m.creds.Management)
	}
}

// The cancel slot belongs to a refresh generation. A superseded fetch finishes
// long after its replacement has registered, and must not release the
// replacement's cancel function: doing so would leave the new request
// uncancellable from quit or the next key replacement.
func TestSupersededFetchCannotClearTheNewestCancel(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	secondCancelled := make(chan struct{})

	var mu sync.Mutex
	calls := 0
	fetch := func(ctx context.Context, opts openrouter.FetchOptions) (openrouter.Snapshot, error) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()

		if n == 1 {
			close(started)
			// Hold the first fetch inside its work, so its cleanup runs after
			// the replacement fetch has registered its own cancel function.
			<-release
			return openrouter.Snapshot{}, ctx.Err()
		}
		select {
		case <-ctx.Done():
			close(secondCancelled)
			return openrouter.Snapshot{}, ctx.Err()
		case <-time.After(10 * time.Second):
			return openrouter.Snapshot{}, errors.New("the replacement fetch was not cancellable")
		}
	}

	m := testModel(t, testModelOptions{now: fixedNow(), fetch: fetch})

	first := m.refreshCommands()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		first()
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("the first fetch never started")
	}

	// Supersede the first fetch the way a key replacement does.
	m.cancelInFlight()
	m.refreshing = false
	second := m.refreshCommands()
	go second()
	waitForCancel(t, m)

	// Now let the superseded fetch run its cleanup.
	close(release)
	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the superseded fetch never returned")
	}

	if !m.cancelPending() {
		t.Fatal("the superseded fetch released the replacement's cancel function")
	}
	m.cancelInFlight()
	select {
	case <-secondCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("the replacement fetch could not be cancelled")
	}
}

// Registration is the mirror image of cleanup: a command whose goroutine
// starts late carries an older generation and must not displace the newer
// request's cancel function. Each fetch is distinguished by function identity
// so the test does not depend on goroutine scheduling.
func TestLateStartingFetchCannotEvictTheNewerCancel(t *testing.T) {
	release := make(chan struct{})
	olderEntered := make(chan struct{})
	newestCancelled := make(chan struct{})
	t.Cleanup(func() { close(release) })

	olderFetch := func(ctx context.Context, opts openrouter.FetchOptions) (openrouter.Snapshot, error) {
		close(olderEntered)
		<-release
		return openrouter.Snapshot{}, ctx.Err()
	}
	newestFetch := func(ctx context.Context, opts openrouter.FetchOptions) (openrouter.Snapshot, error) {
		select {
		case <-ctx.Done():
			close(newestCancelled)
			return openrouter.Snapshot{}, ctx.Err()
		case <-time.After(10 * time.Second):
			return openrouter.Snapshot{}, errors.New("the newest fetch was not cancellable")
		}
	}

	m := testModel(t, testModelOptions{now: fixedNow()})

	// The older command is built first but its goroutine starts last.
	m.fetchFn = olderFetch
	older := m.refreshCommands()
	m.refreshing = false
	m.fetchFn = newestFetch
	newest := m.refreshCommands()

	go newest()
	waitForCancel(t, m)

	go older()
	select {
	case <-olderEntered:
		// setCancel runs before the fetch body, so the older registration has
		// already been attempted by now.
	case <-time.After(2 * time.Second):
		t.Fatal("the older fetch never started")
	}

	m.cancelInFlight()
	select {
	case <-newestCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("a late-starting fetch evicted the newer cancel function")
	}
}

// The accepted response is written to the cache exactly once, and a response
// that loses its generation must never reach it: the next launch would
// otherwise render the superseded reading.
func TestStaleResponseCannotOverwriteTheCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	store := openrouter.NewCacheStore(path)

	var mu sync.Mutex
	calls := 0
	fetch := func(ctx context.Context, fo openrouter.FetchOptions) (openrouter.Snapshot, error) {
		mu.Lock()
		calls++
		remaining := float64(calls * 10)
		mu.Unlock()
		return openrouter.Snapshot{
			SchemaVersion: openrouter.SchemaVersion,
			GeneratedAt:   fo.Now,
			Range:         openrouter.DisplayRange(fo.Now),
			Account:       openrouter.AccountCredits{Remaining: remaining},
			Daily:         openrouter.AggregateDaily(openrouter.DisplayRange(fo.Now), nil),
		}, nil
	}

	m := testModel(t, testModelOptions{cachePath: path, fetch: fetch})

	older := m.refreshCommands()
	m.refreshing = false
	newer := m.refreshCommands()

	// The newer request finishes first and the older one last, so a cache write
	// performed by the losing command would land last.
	newerMsg := newer().(snapshotLoadedMsg)
	olderMsg := older().(snapshotLoadedMsg)
	accepted := newerMsg.snapshot.Account.Remaining
	discarded := olderMsg.snapshot.Account.Remaining
	if accepted == discarded {
		t.Fatalf("test setup: the two responses must differ, both %v", accepted)
	}

	feed(m, newerMsg)
	feed(m, olderMsg)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("the accepted response was not cached: %v", err)
	}
	if loaded.Account.Remaining != accepted {
		t.Fatalf("cache = %v, want the accepted response %v", loaded.Account.Remaining, accepted)
	}
}

// A normal accepted response still populates the cache, and --no-cache still
// suppresses the write now that it happens on the accepting side.
func TestCacheWriteFollowsTheNoCacheFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	fetch := func(ctx context.Context, fo openrouter.FetchOptions) (openrouter.Snapshot, error) {
		return openrouter.Snapshot{
			SchemaVersion: openrouter.SchemaVersion,
			GeneratedAt:   fo.Now,
			Range:         openrouter.DisplayRange(fo.Now),
			Daily:         openrouter.AggregateDaily(openrouter.DisplayRange(fo.Now), nil),
		}, nil
	}

	m := testModel(t, testModelOptions{cachePath: path, fetch: fetch})
	feed(m, m.refreshCommands()())
	if _, err := openrouter.NewCacheStore(path).Load(); err != nil {
		t.Fatalf("an accepted response must be cached: %v", err)
	}

	offPath := filepath.Join(t.TempDir(), "snapshot.json")
	noCache := New(Deps{
		Config: Config{RefreshInterval: 10 * time.Second, NoCache: true},
		Creds:  Creds{Management: "sk-or-v1-x"},
		Cache:  openrouter.NewCacheStore(offPath),
		Now:    fixedNow,
		Fetch:  fetch,
	})
	noCache.SetSize(100, 40)
	feed(noCache, noCache.refreshCommands()())
	if _, err := os.Stat(offPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("--no-cache must not write the cache: %v", err)
	}
}

// The help key must not open Help while Settings owns the keyboard.
func TestSettingsOwnsTheKeyboardWhileOpen(t *testing.T) {
	m := settingsModel(t, settingsWiring{})
	openSettings(t, m)
	pressKey(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.helpVisible {
		t.Fatal("? must not open Help on top of the Settings overlay")
	}
	if !m.settingsVisible {
		t.Fatal("? must not close Settings")
	}
}

var _ = openrouter.Snapshot{}
