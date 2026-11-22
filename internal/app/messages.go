package app

import (
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// State is the top-level application state.
type State int

// Application states.
const (
	StateSetup State = iota
	StateLoading
	StateReady
	StateFatalError
)

// String renders the state for tests and logs.
func (s State) String() string {
	switch s {
	case StateSetup:
		return "setup"
	case StateLoading:
		return "loading"
	case StateReady:
		return "ready"
	case StateFatalError:
		return "fatal"
	default:
		return "unknown"
	}
}

// windowSizeMsg is our own copy of the terminal size event so tests can send
// it without constructing a tea.WindowSizeMsg.
type windowSizeMsg struct {
	Width  int
	Height int
}

// tickMsg fires when the scheduler wants a refresh.
type tickMsg struct{ gen int }

// refreshRequestedMsg is produced by the r key.
type refreshRequestedMsg struct{}

// snapshotLoadedMsg carries the result of one fetch batch.
type snapshotLoadedMsg struct {
	gen      int
	snapshot domain.Snapshot
	err      error
}

// cacheLoadedMsg carries a snapshot read from disk at startup.
type cacheLoadedMsg struct {
	snapshot domain.Snapshot
	age      time.Duration
	err      error
}

// setupCompleteMsg reports a successful first-run setup.
type setupCompleteMsg struct {
	creds credsResult
	err   error
}

// credsResult mirrors the resolved credential pair without importing the
// credentials package into message payloads.
type credsResult struct {
	Management string
	APIKey     string
}

// fatalErrorMsg aborts the dashboard because nothing usable exists.
type fatalErrorMsg struct{ err error }

// noticeMsg shows a transient one-line hint such as a throttle message.
type noticeMsg struct{ text string }
