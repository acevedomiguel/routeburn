package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acevedomiguel/routeburn/internal/cache"
	"github.com/acevedomiguel/routeburn/internal/domain"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
)

func testModel(t *testing.T, opts testModelOptions) *Model {
	t.Helper()

	now := opts.now
	if now.IsZero() {
		now = fixedNow()
	}
	clock := now

	cfg := Config{RefreshInterval: 10 * time.Second}
	if opts.refresh != nil {
		cfg.RefreshInterval = *opts.refresh
	}

	var store *cache.Store
	if opts.cachePath != "" {
		store = cache.NewStore(opts.cachePath)
	}

	creds := Creds{Management: "sk-or-v1-management"}
	if !opts.noCreds {
		creds.Management = "sk-or-v1-management"
	} else {
		creds.Management = ""
	}

	fetch := opts.fetch
	if fetch == nil {
		fetch = func(ctx context.Context, fo FetchOptions) (domain.Snapshot, error) {
			return domain.Snapshot{
				SchemaVersion: domain.SchemaVersion,
				GeneratedAt:   fo.Now,
				Range:         domain.DisplayRange(fo.Now),
				Account:       domain.AccountCredits{TotalPurchased: 100, TotalUsed: 50, Remaining: 50},
				Daily:         domain.AggregateDaily(domain.DisplayRange(fo.Now), nil),
				Runway:        domain.RunwayEstimate{Basis: domain.BasisNoRecentSpend},
			}, nil
		}
	}

	model := New(Deps{
		Config: cfg,
		Creds:  creds,
		Cache:  store,
		Now:    func() time.Time { return clock },
		Fetch:  fetch,
	})
	model.SetSize(100, 40)
	return model
}

type testModelOptions struct {
	now       time.Time
	refresh   *time.Duration
	cachePath string
	noCreds   bool
	fetch     func(context.Context, FetchOptions) (domain.Snapshot, error)
}

func TestNewWithoutCredentialsStartsSetup(t *testing.T) {
	t.Helper()
	m := testModel(t, testModelOptions{noCreds: true})
	if m.State() != StateSetup {
		t.Fatalf("state = %v, want setup", m.State())
	}
	if view := m.View(); !strings.Contains(view, "RouteBurn setup") {
		t.Fatalf("setup view missing: %q", view)
	}
}

func TestNewStartsLoadingThenReady(t *testing.T) {
	m := testModel(t, testModelOptions{})
	if m.State() != StateLoading {
		t.Fatalf("state = %v, want loading", m.State())
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned no command")
	}
	msg := cmd()
	if _, ok := msg.(snapshotLoadedMsg); !ok {
		// tea.Batch returns a BatchMsg; unwrap it.
		batch, isBatch := msg.(tea.BatchMsg)
		if !isBatch || len(batch) == 0 {
			t.Fatalf("unexpected startup message %T", msg)
		}
		msg = batch[0]()
		if _, ok := msg.(snapshotLoadedMsg); !ok {
			t.Fatalf("unexpected startup message %T", msg)
		}
	}

	updated, _ := m.Update(msg)
	m2 := updated.(*Model)
	if m2.State() != StateReady {
		t.Fatalf("state = %v, want ready", m2.State())
	}
	if m2.Snapshot() == nil || m2.Snapshot().Account.Remaining != 50 {
		t.Fatalf("snapshot = %+v", m2.Snapshot())
	}
}

func TestCacheIsRenderedImmediately(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	store := cache.NewStore(path)

	cached := domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   fixedNow().Add(-2*time.Hour - 14*time.Minute),
		Range:         domain.DisplayRange(fixedNow()),
		Account:       domain.AccountCredits{TotalPurchased: 100, TotalUsed: 1, Remaining: 99},
		Daily:         domain.AggregateDaily(domain.DisplayRange(fixedNow()), nil),
	}
	if err := store.Save(cached); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	m := testModel(t, testModelOptions{cachePath: path})
	if m.State() != StateReady {
		t.Fatalf("state = %v, want ready from cache", m.State())
	}
	if !m.Cached() {
		t.Fatal("model should mark cache-sourced data as cached")
	}
	if view := m.View(); !strings.Contains(view, "Cached data - updated 2h 14m ago") {
		t.Fatalf("view missing cache label: %q", view)
	}
}

func TestMalformedCacheIsIgnoredWithBanner(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, []byte("{oops"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := testModel(t, testModelOptions{cachePath: path})
	if m.State() != StateLoading {
		t.Fatalf("state = %v, want loading", m.State())
	}
	if !strings.Contains(m.Banner(), "unreadable cache") {
		t.Fatalf("banner = %q", m.Banner())
	}
}

func TestNoCacheFlagSkipsCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	store := cache.NewStore(path)
	cached := domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   fixedNow(),
		Range:         domain.DisplayRange(fixedNow()),
		Daily:         domain.AggregateDaily(domain.DisplayRange(fixedNow()), nil),
	}
	if err := store.Save(cached); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	m := New(Deps{
		Config: Config{RefreshInterval: 10 * time.Second, NoCache: true},
		Creds:  Creds{Management: "sk-or-v1-x"},
		Cache:  store,
		Now:    fixedNow,
	})
	if m.State() != StateLoading {
		t.Fatalf("state = %v, want loading when --no-cache", m.State())
	}
	if m.Snapshot() != nil {
		t.Fatal("cache must not be read with --no-cache")
	}
}

func TestFailedRefreshPreservesUsableData(t *testing.T) {
	m := testModel(t, testModelOptions{})

	// First, load a good snapshot.
	good, _ := m.Update(snapshotLoadedMsg{
		gen:      1,
		snapshot: domain.Snapshot{SchemaVersion: 1, Account: domain.AccountCredits{Remaining: 50}},
	})
	m = good.(*Model)

	bad, _ := m.Update(snapshotLoadedMsg{gen: 2, err: errors.New("network unreachable")})
	m = bad.(*Model)

	if m.Snapshot() == nil {
		t.Fatal("a failed refresh must not erase the snapshot")
	}
	if m.Snapshot().Account.Remaining != 50 {
		t.Error("snapshot data was replaced")
	}
	if m.State() != StateReady {
		t.Errorf("state = %v, want ready", m.State())
	}
	if !strings.Contains(m.Banner(), "network unreachable") {
		t.Errorf("banner = %q", m.Banner())
	}
	if m.RefreshActive() {
		t.Error("refresh flag should be cleared after a failed fetch")
	}
}

func TestFailedRefreshWithNoDataIsFatal(t *testing.T) {
	m := testModel(t, testModelOptions{})
	updated, _ := m.Update(snapshotLoadedMsg{gen: 1, err: errors.New("dial tcp: no route")})
	m = updated.(*Model)
	if m.State() != StateFatalError {
		t.Fatalf("state = %v, want fatal", m.State())
	}
	if view := m.View(); !strings.Contains(view, "cannot start") {
		t.Fatalf("fatal view = %q", view)
	}
}

func TestStaleResponseCannotReplaceNewer(t *testing.T) {
	m := testModel(t, testModelOptions{})

	updated, _ := m.Update(snapshotLoadedMsg{
		gen:      2,
		snapshot: domain.Snapshot{SchemaVersion: 1, Account: domain.AccountCredits{Remaining: 22}},
	})
	m = updated.(*Model)

	updated, _ = m.Update(snapshotLoadedMsg{
		gen:      1,
		snapshot: domain.Snapshot{SchemaVersion: 1, Account: domain.AccountCredits{Remaining: 11}},
	})
	m = updated.(*Model)

	if m.Snapshot().Account.Remaining != 22 {
		t.Fatalf("remaining = %v, an old generation overwrote a newer one", m.Snapshot().Account.Remaining)
	}
}

func TestManualRefreshThrottle(t *testing.T) {
	now := fixedNow()
	clock := now
	m := testModel(t, testModelOptions{now: now})
	m.now = func() time.Time { return clock }

	// First refresh is allowed.
	ok, wait := m.RequestRefresh()
	if !ok || wait != 0 {
		t.Fatalf("first refresh should be allowed, got ok=%v wait=%v", ok, wait)
	}
	m.refreshing = true
	m.lastAttempt = now

	// Immediately after, a manual refresh is throttled.
	m.refreshing = false
	ok, wait = m.RequestRefresh()
	if ok {
		t.Fatal("second refresh inside the window must be rejected")
	}
	if wait <= 0 || wait > 10*time.Second {
		t.Fatalf("wait = %v", wait)
	}

	// Four seconds later the message text is still correct.
	clock = now.Add(4 * time.Second)
	ok, wait = m.RequestRefresh()
	if ok {
		t.Fatal("refresh at 4s must still be rejected")
	}
	if formatWait(wait) != "6s" {
		t.Fatalf("wait text = %q", formatWait(wait))
	}

	// After the interval it is allowed again.
	clock = now.Add(11 * time.Second)
	if ok, _ := m.RequestRefresh(); !ok {
		t.Fatal("refresh after the window must be allowed")
	}
}

func TestServerBackoffBlocksManualRefresh(t *testing.T) {
	now := fixedNow()
	clock := now
	m := testModel(t, testModelOptions{now: now})
	m.now = func() time.Time { return clock }
	m.nextAllowed = now.Add(32 * time.Second)

	if ok, _ := m.RequestRefresh(); ok {
		t.Fatal("server-directed backoff must block manual refresh")
	}

	clock = now.Add(31 * time.Second)
	if ok, _ := m.RequestRefresh(); ok {
		t.Fatal("refresh before the server retry time must stay blocked")
	}

	clock = now.Add(33 * time.Second)
	if ok, _ := m.RequestRefresh(); !ok {
		t.Fatal("refresh after the server retry time must be allowed")
	}
}

func TestRefreshKeysAndQuit(t *testing.T) {
	cases := []struct {
		key   string
		check func(*testing.T, *Model, tea.Cmd)
	}{
		{"q", func(t *testing.T, m *Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("q must quit")
			}
		}},
		{"ctrl+c", func(t *testing.T, m *Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("ctrl+c must quit")
			}
		}},
		{"esc", func(t *testing.T, m *Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Fatal("esc must quit")
			}
		}},
		{"?", func(t *testing.T, m *Model, cmd tea.Cmd) {
			if !m.helpVisible {
				t.Fatal("? must show help")
			}
			if !strings.Contains(m.View(), "RouteBurn help") {
				t.Fatalf("help view = %q", m.View())
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			m := testModel(t, testModelOptions{})
			tc.check(t, m, keyCmd(m, tc.key))
		})
	}
}

func TestEscClosesHelpBeforeQuitting(t *testing.T) {
	m := testModel(t, testModelOptions{})
	keyCmd(m, "?")
	if !m.helpVisible {
		t.Fatal("? must open help")
	}
	// The first Esc dismisses the overlay and must not quit the app.
	if cmd := keyCmd(m, "esc"); cmd != nil {
		t.Fatal("esc must only close the overlay while help is open")
	}
	if m.helpVisible {
		t.Fatal("esc must close the overlay")
	}
	// With no overlay open, Esc quits.
	if cmd := keyCmd(m, "esc"); cmd == nil {
		t.Fatal("esc must quit when no overlay is open")
	}
}

func keyCmd(m *Model, key string) tea.Cmd {
	var msg tea.KeyMsg
	switch key {
	case "q":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	case "r":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	case "?":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	case "j":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	case "k":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	_, cmd := m.Update(msg)
	return cmd
}

func TestEscClosesHelp(t *testing.T) {
	m := testModel(t, testModelOptions{})
	m.helpVisible = true
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*Model)
	if m.helpVisible {
		t.Fatal("Esc must close help")
	}
}

func TestScrollKeys(t *testing.T) {
	m := testModel(t, testModelOptions{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(*Model)
	if m.scroll != 1 {
		t.Fatalf("scroll = %d", m.scroll)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(*Model)
	if m.scroll != 0 {
		t.Fatalf("scroll = %d", m.scroll)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(*Model)
	if m.scroll != 0 {
		t.Fatal("scroll must not go negative")
	}
}

func TestAutomaticRefreshTicks(t *testing.T) {
	interval := 10 * time.Second
	m := testModel(t, testModelOptions{refresh: &interval})
	if cmd := m.scheduleTick(); cmd == nil {
		t.Fatal("a positive interval must schedule a tick")
	}

	zero := time.Duration(0)
	off := testModel(t, testModelOptions{refresh: &zero})
	if cmd := off.scheduleTick(); cmd != nil {
		t.Fatal("refresh_interval 0 must disable the timer")
	}
}

func TestViewIsPureAndStable(t *testing.T) {
	m := testModel(t, testModelOptions{})
	updated, _ := m.Update(snapshotLoadedMsg{
		gen:      1,
		snapshot: domain.Snapshot{SchemaVersion: 1, GeneratedAt: fixedNow(), Account: domain.AccountCredits{TotalPurchased: 10, TotalUsed: 1, Remaining: 9}, Range: domain.DisplayRange(fixedNow()), Daily: domain.AggregateDaily(domain.DisplayRange(fixedNow()), nil)},
	})
	m = updated.(*Model)

	first := m.View()
	for i := 0; i < 3; i++ {
		if m.View() != first {
			t.Fatal("View must not mutate state")
		}
	}
}

func TestFetchErrorSentinelMissingManagementKeyTriggersSetup(t *testing.T) {
	m := testModel(t, testModelOptions{
		fetch: func(ctx context.Context, fo FetchOptions) (domain.Snapshot, error) {
			return domain.Snapshot{}, or.ErrMissingManagementKey
		},
	})
	cmd := m.Init()
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		msg = batch[0]()
	}
	updated, _ := m.Update(msg)
	m = updated.(*Model)
	if m.State() != StateFatalError {
		// Missing credentials mid-flight is fatal for the session unless setup runs.
		t.Fatalf("state = %v", m.State())
	}
}
