package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/spinner"
	"github.com/acevedomiguel/routeburn/internal/cache"
	"github.com/acevedomiguel/routeburn/internal/domain"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// Model is the Bubble Tea model for the whole application.
type Model struct {
	cfg Config

	// apiMu guards api. A fetch command runs on its own goroutine and reads the
	// client, while replacing the management key swaps it from Update, so the
	// pair must be synchronized.
	apiMu sync.RWMutex
	api   API

	creds Creds
	cache *cache.Store
	save  func(domain.Snapshot) error
	now   func() time.Time
	loc   *time.Location

	state       State
	width       int
	height      int
	snapshot    *domain.Snapshot
	previous    *domain.Snapshot
	loading     bool
	refreshing  bool
	cached      bool
	cacheAge    time.Duration
	lastAttempt time.Time
	lastSuccess time.Time
	nextAllowed time.Time
	errorBanner string
	notice      string
	version     string
	demo        bool
	helpVisible bool
	helpScroll  int
	scroll      int
	// tableOffset is the daily table's own scroll position, in days from the
	// newest. It survives losing focus so Tab back lands where you left off.
	tableOffset  int
	tableFocused bool

	// tickGen identifies the pending automatic-refresh timer. Advancing it
	// supersedes the previous timer without disturbing an in-flight refresh,
	// which has its own generation.
	tickGen int

	// Settings overlay state and the side effects it drives.
	settingsVisible     bool
	settingsSavedScroll int
	intervalOptions     []time.Duration
	setInterval         func(time.Duration) error
	intervalPinned      bool
	validateKey         func(context.Context, string) error
	saveKey             func(context.Context, string) error
	rebuildAPI          func(management, apiKey string) (API, error)
	envKey              bool
	settingsFocus       int
	settingsIndex       int
	keyInput            string
	keyEditing          bool
	keySaving           bool
	settingsMsg         string
	settingsErr         bool

	gen int

	spinner spinner.Model

	cancelMu    sync.Mutex
	cancelFetch context.CancelFunc

	fetchFn func(context.Context, FetchOptions) (domain.Snapshot, error)
}

// currentAPI returns the API client in use. It is safe to call from a fetch
// goroutine.
func (m *Model) currentAPI() API {
	m.apiMu.RLock()
	defer m.apiMu.RUnlock()
	return m.api
}

// setAPI replaces the API client, used when the management key is replaced.
func (m *Model) setAPI(api API) {
	m.apiMu.Lock()
	m.api = api
	m.apiMu.Unlock()
}

// setCancel records the cancel function of the running fetch so quit can stop it.
func (m *Model) setCancel(cancel context.CancelFunc) {
	m.cancelMu.Lock()
	m.cancelFetch = cancel
	m.cancelMu.Unlock()
}

// cancelInFlight stops any running fetch immediately.
func (m *Model) cancelInFlight() {
	m.cancelMu.Lock()
	cancel := m.cancelFetch
	m.cancelFetch = nil
	m.cancelMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// cancelPending reports whether a fetch is currently registered as in flight.
func (m *Model) cancelPending() bool {
	m.cancelMu.Lock()
	defer m.cancelMu.Unlock()
	return m.cancelFetch != nil
}

type Config struct {
	RefreshInterval time.Duration
	NoCache         bool
	NoColor         bool
	RangeDays       int
}

// Creds mirrors the resolved credential pair.
type Creds struct {
	Management string
	APIKey     string
}

// HasManagement reports whether a management key exists.
func (c Creds) HasManagement() bool { return c.Management != "" }

// Deps are the model dependencies. Everything is injectable for tests.
type Deps struct {
	Config   Config
	Creds    Creds
	API      API
	Cache    *cache.Store
	Now      func() time.Time
	Location *time.Location
	// Version is the build version shown in Help; empty means "dev".
	Version string
	// Demo labels the title row as synthetic fixture data.
	Demo bool
	// Fetch overrides the default snapshot builder (tests).
	Fetch func(context.Context, FetchOptions) (domain.Snapshot, error)

	// IntervalOptions are the selectable automatic-refresh intervals, in
	// ascending order. Nil selects ui.DefaultIntervalOptions.
	IntervalOptions []time.Duration
	// SetInterval persists an interval choice to the active configuration
	// file. Nil makes the choice session-only, which is what demo mode uses.
	SetInterval func(time.Duration) error
	// IntervalPinned is true when a flag or environment variable outranks the
	// saved file, so a change applies now but not on the next start.
	IntervalPinned bool
	// ValidateKey checks a candidate management key with one bounded read. A
	// nil ValidateKey, SaveKey, or RebuildAPI disables in-app key replacement.
	ValidateKey func(context.Context, string) error
	// SaveKey stores a management key that ValidateKey has already accepted,
	// replacing the previous one.
	SaveKey func(context.Context, string) error
	// RebuildAPI builds a fresh API client for a replaced management key, so
	// the new credential takes effect without a restart.
	RebuildAPI func(management, apiKey string) (API, error)
	// EnvKey is true when ROUTEBURN_MANAGEMENT_KEY supplies the key, so an
	// in-app replacement cannot take effect until it is unset.
	EnvKey bool
}

// New builds the initial model.
func New(deps Deps) *Model {
	now := deps.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	loc := deps.Location
	if loc == nil {
		loc = time.UTC
	}
	version := deps.Version
	if version == "" {
		version = "dev"
	}

	intervalOptions := deps.IntervalOptions
	if len(intervalOptions) == 0 {
		intervalOptions = ui.DefaultIntervalOptions()
	}

	m := &Model{
		cfg:     deps.Config,
		creds:   Creds{Management: deps.Creds.Management, APIKey: deps.Creds.APIKey},
		api:     deps.API,
		cache:   deps.Cache,
		now:     now,
		loc:     loc,
		version: version,
		demo:    deps.Demo,
		state:   StateLoading,
		gen:     0,
		spinner: ui.NewSpinner(ui.NewStyles(!deps.Config.NoColor)),

		intervalOptions: intervalOptions,
		setInterval:     deps.SetInterval,
		intervalPinned:  deps.IntervalPinned,
		validateKey:     deps.ValidateKey,
		saveKey:         deps.SaveKey,
		rebuildAPI:      deps.RebuildAPI,
		envKey:          deps.EnvKey,
		settingsIndex:   intervalIndexFor(deps.Config.RefreshInterval, intervalOptions),
	}
	if deps.Fetch != nil {
		m.fetchFn = deps.Fetch
	} else {
		m.fetchFn = func(ctx context.Context, opts FetchOptions) (domain.Snapshot, error) {
			return FetchSnapshot(ctx, m.currentAPI(), opts)
		}
	}

	if !m.creds.HasManagement() {
		m.state = StateSetup
		return m
	}

	// Render cached data immediately when it exists.
	if m.cache != nil && !m.cfg.NoCache {
		snap, err := m.cache.Load()
		switch {
		case err == nil:
			stored := snap
			m.snapshot = &stored
			m.cached = true
			if !snap.GeneratedAt.IsZero() {
				m.cacheAge = now().Sub(snap.GeneratedAt)
			}
			m.state = StateReady
		case errors.Is(err, cache.ErrNoSnapshot):
			// Nothing to show yet.
		default:
			m.errorBanner = "Ignored an unreadable cache - " + credRedact(err.Error())
		}
	}

	return m
}

// SetSize applies a terminal size.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// State reports the current state.
func (m *Model) State() State { return m.state }

// Snapshot returns the displayed snapshot, if any.
func (m *Model) Snapshot() *domain.Snapshot { return m.snapshot }

// Banner returns the current error banner text.
func (m *Model) Banner() string {
	if m.notice != "" {
		return m.notice
	}
	return m.errorBanner
}

// Cached reports whether the displayed snapshot came from the cache.
func (m *Model) Cached() bool { return m.cached }

// CacheAge reports how old the cached snapshot is.
func (m *Model) CacheAge() time.Duration { return m.cacheAge }

// RefreshActive reports whether a fetch is in flight.
func (m *Model) RefreshActive() bool { return m.refreshing }

// NextAllowed reports the server-directed earliest next request time.
func (m *Model) NextAllowed() time.Time { return m.nextAllowed }

// throttleRemaining reports how long a manual refresh must wait.
func (m *Model) throttleRemaining() time.Duration {
	next := m.nextAllowed
	if !m.lastAttempt.IsZero() && m.cfg.RefreshInterval > 0 {
		// The 10s floor applies to manual refreshes regardless of the
		// configured automatic interval, so a fast timer never lets r spam
		// the API.
		interval := m.cfg.RefreshInterval
		if interval < minRefreshInterval {
			interval = minRefreshInterval
		}
		fromAttempt := m.lastAttempt.Add(interval)
		if fromAttempt.After(next) {
			next = fromAttempt
		}
	}
	if next.IsZero() {
		return 0
	}
	if now := m.now(); now.Before(next) {
		return next.Sub(now)
	}
	return 0
}

const minRefreshInterval = 10 * time.Second

// RequestRefresh implements the manual refresh throttle.
func (m *Model) RequestRefresh() (bool, time.Duration) {
	if m.refreshing {
		return false, 0
	}
	if wait := m.throttleRemaining(); wait > 0 {
		return false, wait
	}
	return true, 0
}

// refreshCommands builds the command that performs one fetch batch.
func (m *Model) refreshCommands() tea.Cmd {
	if m.refreshing {
		return nil
	}
	m.refreshing = true
	m.loading = true
	m.lastAttempt = m.now()
	m.gen++
	gen := m.gen

	previous := m.snapshot
	now := m.now()
	loc := m.loc
	fetch := m.fetchFn
	cacheStore := m.cache
	noCache := m.cfg.NoCache

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.setCancel(cancel)
		defer m.setCancel(nil)

		snap, err := fetch(ctx, FetchOptions{
			Now:      now,
			Location: loc,
			Previous: previous,
		})
		if err == nil && cacheStore != nil && !noCache {
			// A cache failure must never fail the refresh.
			_ = cacheStore.Save(snap)
		}
		return snapshotLoadedMsg{gen: gen, snapshot: snap, err: err}
	}
}

// scheduleTick arms the single automatic refresh timer. Advancing the tick
// generation supersedes any pending timer, so changing the interval never
// leaves two schedules running; an in-flight refresh keeps its own generation
// and is unaffected.
func (m *Model) scheduleTick() tea.Cmd {
	if m.cfg.RefreshInterval <= 0 {
		return nil
	}
	delay := m.cfg.RefreshInterval
	if !m.nextAllowed.IsZero() {
		if remaining := m.nextAllowed.Sub(m.now()); remaining > delay {
			return nil
		}
	}
	m.tickGen++
	gen := m.tickGen
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return tickMsg{gen: gen}
	})
}

// fetchNow is a small helper used by tests and the JSON path.
func fetchOnce(ctx context.Context, api API, opts FetchOptions) (domain.Snapshot, error) {
	return FetchSnapshot(ctx, api, opts)
}

// NextRefreshAllowed reports the client's server-directed throttle.
func (m *Model) NextRefreshAllowedFromServer() time.Time {
	api := m.currentAPI()
	if api == nil {
		return time.Time{}
	}
	return api.NextRefreshAllowedAt()
}

// Ensure err is referenced in this file for clarity of intent.
var _ = or.ErrMissingManagementKey
var _ = fmt.Sprintf
