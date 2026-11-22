package app

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

func TestBuildJSONContractShape(t *testing.T) {
	snap := SampleSnapshotForJSON()

	raw, err := MarshalJSON(snap, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid JSON: %s", raw)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"schema_version", "generated_at", "cached", "account", "last_30_days", "runway", "key", "warnings"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}

	if got := decoded["schema_version"].(float64); int(got) != domain.SchemaVersion {
		t.Errorf("schema_version = %v", got)
	}
	if decoded["cached"] != false {
		t.Errorf("cached = %v", decoded["cached"])
	}

	account := decoded["account"].(map[string]any)
	for _, key := range []string{"total_credits", "total_usage", "remaining"} {
		if _, ok := account[key]; !ok {
			t.Errorf("account missing %q", key)
		}
	}

	last := decoded["last_30_days"].(map[string]any)
	for _, key := range []string{"credit_spend", "tokens"} {
		if _, ok := last[key]; !ok {
			t.Errorf("last_30_days missing %q", key)
		}
	}

	runway := decoded["runway"].(map[string]any)
	for _, key := range []string{"window_days", "daily_burn", "days_remaining", "depletion_date", "basis", "low_confidence"} {
		if _, ok := runway[key]; !ok {
			t.Errorf("runway missing %q", key)
		}
	}
}

func TestMarshalJSONTrailingNewline(t *testing.T) {
	raw, err := MarshalJSON(domain.Snapshot{SchemaVersion: domain.SchemaVersion}, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Fatalf("output must end with a newline: %q", raw)
	}
}

func TestMarshalJSONBasisDefaultsWhenMissing(t *testing.T) {
	out := BuildJSON(domain.Snapshot{SchemaVersion: domain.SchemaVersion}, true)
	if out.Runway.Basis != string(domain.BasisUnavailable) {
		t.Fatalf("basis = %q", out.Runway.Basis)
	}
	if !out.Cached {
		t.Fatal("cached flag must survive")
	}
}

func TestMarshalJSONOmitsNilKeyAndKeepsWarnings(t *testing.T) {
	snap := domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		Warnings:      []domain.Warning{{Section: "Daily usage", Message: "unavailable"}},
	}
	out := BuildJSON(snap, false)
	if out.Key != nil {
		t.Fatalf("key = %+v, want nil", out.Key)
	}
	if len(out.Warnings) != 1 || out.Warnings[0] != "Daily usage: unavailable" {
		t.Fatalf("warnings = %v", out.Warnings)
	}
}

func TestMarshalJSONConvertsDepletionDate(t *testing.T) {
	date := time.Date(2027, 1, 6, 0, 0, 0, 0, time.UTC)
	days := 46.0
	snap := domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		Runway: domain.RunwayEstimate{
			WindowDays:    30,
			DailyBurn:     0.4,
			DaysRemaining: &days,
			DepletionDate: &date,
			Basis:         domain.BasisFinite,
		},
	}
	out := BuildJSON(snap, false)
	if out.Runway.DepletionDate == nil || *out.Runway.DepletionDate != "2027-01-06" {
		t.Fatalf("depletion_date = %v", out.Runway.DepletionDate)
	}
	if out.Runway.DaysRemaining == nil || *out.Runway.DaysRemaining != 46 {
		t.Fatalf("days_remaining = %v", out.Runway.DaysRemaining)
	}
}

func TestMarshalJSONIsStableAcrossCalls(t *testing.T) {
	snap := SampleSnapshotForJSON()
	first, err := MarshalJSON(snap, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for i := 0; i < 3; i++ {
		again, err := MarshalJSON(snap, false)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(again) != string(first) {
			t.Fatal("JSON output must be deterministic")
		}
	}

	if _, err := MarshalJSON(domain.Snapshot{}, false); err != nil {
		t.Fatalf("zero snapshot must marshal: %v", err)
	}
	if errors.Is(err, nil) {
		t.Log("ok")
	}
}
