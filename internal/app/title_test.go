package app

import (
	"testing"

	"github.com/acevedomiguel/burnblocks/openrouter"
	"github.com/acevedomiguel/routeburn/internal/format"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

// The terminal tab carries remaining credit and a numeric runway, like
// Kimiburn's quota title; anything else falls back to the app name.
func TestWindowTitleCarriesCreditAndRunway(t *testing.T) {
	m := testModel(t, testModelOptions{})
	snap := ui.SampleSnapshotAt(fixedNow())
	m.snapshot = &snap
	m.state = StateReady

	want := "routeburn · " + format.FormatCredits(snap.Account.Remaining) + " left · runway " + format.FormatDaysRemaining(snap.Runway)
	if snap.Runway.Basis != openrouter.BasisFinite {
		t.Fatalf("fixture runway basis = %q, want finite", snap.Runway.Basis)
	}
	if got := m.View().WindowTitle; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}

	snap.Runway.Basis = openrouter.BasisNoRecentSpend
	if got, want := m.View().WindowTitle, "routeburn · "+format.FormatCredits(snap.Account.Remaining)+" left"; got != want {
		t.Fatalf("non-numeric runway title = %q, want %q", got, want)
	}

	m.state = StateSetup
	if got := m.View().WindowTitle; got != "routeburn" {
		t.Fatalf("setup title = %q, want routeburn", got)
	}
}
