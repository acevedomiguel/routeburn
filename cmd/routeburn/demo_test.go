package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/charmbracelet/colorprofile"
)

func TestParseDemoCommand(t *testing.T) {
	opts, err := parse([]string{"demo"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse demo: %v", err)
	}
	if opts.command != "demo" {
		t.Fatalf("command = %q, want %q", opts.command, "demo")
	}
}

func TestParseDemoRejectsExtraArguments(t *testing.T) {
	_, err := parse([]string{"demo", "extra"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected a usage error for extra arguments")
	}
}

// The demo model must render the synthetic fixture without credentials,
// configuration, or any network access.
func TestDemoModelRendersFixture(t *testing.T) {
	now := time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
	m := demoModel(func() time.Time { return now })

	msg := m.StartRefresh()()
	m.Update(msg)

	snap := m.Snapshot()
	if snap == nil {
		t.Fatal("demo model must render the fixture snapshot")
	}
	if snap.Account.Remaining != 42.10 {
		t.Errorf("remaining = %v, want the fixture value 42.10", snap.Account.Remaining)
	}
	if len(snap.Daily) != 30 {
		t.Errorf("fixture day count = %d, want 30", len(snap.Daily))
	}
	if m.Cached() {
		t.Error("demo snapshot must not be reported as cached")
	}
}

// Captures are only representative when color survives the recorder's
// environment, so the demo must pin a non-Ascii color profile.
func TestDemoForcesColorProfile(t *testing.T) {
	if demoColorProfile() == colorprofile.NoTTY || demoColorProfile() == colorprofile.Ascii {
		t.Fatal("demo must force a color-capable profile for recordings")
	}
}
