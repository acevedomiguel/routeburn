package app

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

// runAll executes a tea.Cmd tree synchronously, returning the final message.
func runAll(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return msg
	}
	for _, c := range batch {
		if c != nil {
			c()
		}
	}
	return nil
}

// Quitting must cancel in-flight requests instead of waiting out the timeout.
func TestQuitCancelsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})

	m := testModel(t, testModelOptions{
		now: fixedNow(),
		fetch: func(ctx context.Context, opts openrouter.FetchOptions) (openrouter.Snapshot, error) {
			close(started)
			select {
			case <-ctx.Done():
				close(cancelled)
				return openrouter.Snapshot{}, ctx.Err()
			case <-time.After(10 * time.Second):
				return openrouter.Snapshot{}, errors.New("fetch was not cancelled")
			}
		},
	})

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected a refresh command")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		runAll(cmd)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("fetch never started")
	}

	_, quitCmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatal("q must quit")
	}

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("quitting did not cancel the in-flight request")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("refresh command did not return after cancellation")
	}
}

// Ctrl+C must cancel too.
func TestCtrlCCancelsInFlightRequests(t *testing.T) {
	cancelled := make(chan struct{})

	m := testModel(t, testModelOptions{
		now: fixedNow(),
		fetch: func(ctx context.Context, opts openrouter.FetchOptions) (openrouter.Snapshot, error) {
			select {
			case <-ctx.Done():
				close(cancelled)
				return openrouter.Snapshot{}, ctx.Err()
			case <-time.After(10 * time.Second):
				return openrouter.Snapshot{}, errors.New("not cancelled")
			}
		},
	})

	go runAll(m.Init())
	// Give the command goroutine a moment to register its context.
	deadline := time.Now().Add(2 * time.Second)
	for !m.cancelPending() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("ctrl+c did not cancel the in-flight request")
	}
}
