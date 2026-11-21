package app

import (
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	"github.com/acevedomiguel/routeburn/internal/ui"
)

var demoNow = time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)

func TestDemoSnapshotIsInternallyConsistent(t *testing.T) {
	snap := DemoSnapshot(demoNow)

	if snap.SchemaVersion != domain.SchemaVersion {
		t.Errorf("schema version = %d", snap.SchemaVersion)
	}
	if got, want := len(snap.Daily), 30; got != want {
		t.Fatalf("daily buckets = %d, want %d", got, want)
	}
	for i, bucket := range snap.Daily {
		want := snap.Range.Start.AddDate(0, 0, i)
		if !bucket.Day.Equal(want) {
			t.Errorf("bucket %d day = %s, want %s", i, bucket.Day, want)
		}
	}

	total := domain.SumDaily(snap.Daily)
	if snap.TotalTokens != total.Tokens {
		t.Errorf("total tokens = %d, want %d", snap.TotalTokens, total.Tokens)
	}

	if got := snap.Account.Remaining; got != 42.10 {
		t.Errorf("remaining = %v", got)
	}
	if snap.Runway.Basis != domain.BasisFinite {
		t.Errorf("basis = %q, want finite", snap.Runway.Basis)
	}
	if snap.Runway.DaysRemaining == nil || *snap.Runway.DaysRemaining <= 0 {
		t.Fatalf("days remaining = %v", snap.Runway.DaysRemaining)
	}

	for i := 1; i < len(snap.Models); i++ {
		if snap.Models[i].CreditSpend > snap.Models[i-1].CreditSpend {
			t.Errorf("models not sorted at %d: %+v", i, snap.Models)
		}
	}
}

func TestDemoSnapshotExcludesThePartialDay(t *testing.T) {
	// A snapshot taken at 00:30 must not treat 30 minutes of spend as a day.
	early := time.Date(2026, 11, 21, 0, 30, 0, 0, time.UTC)
	snap := DemoSnapshot(early)

	window := domain.RunwayRange(early)
	if !snap.Runway.DepletionDate.After(window.End) {
		t.Errorf("depletion date %s must be after the runway window end %s",
			snap.Runway.DepletionDate, window.End)
	}
	if snap.Runway.WindowDays != 30 {
		t.Errorf("window days = %d", snap.Runway.WindowDays)
	}
}

func TestDemoSnapshotRenders(t *testing.T) {
	snap := DemoSnapshot(demoNow)
	out := ui.Render(&snap, ui.ViewOptions{
		Width:  100,
		Height: 30,
		Styles: ui.NewStyles(false),
		Now:    demoNow,
	})

	for _, want := range []string{"Credits remaining", "$42.10", "Daily credit spend", "Cost by model", "anthropic/claude-sonnet-4"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "loading") {
		t.Errorf("demo snapshot must not render a loading state:\n%s", out)
	}
	if strings.ContainsAny(out, "\x1b") {
		t.Error("no-color render must not contain escape sequences")
	}
}
