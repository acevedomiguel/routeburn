package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

func fixedSnapshot() domain.Snapshot {
	return domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC),
		Range:         domain.DisplayRange(time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC)),
		Account:       domain.AccountCredits{TotalPurchased: 30.4, TotalUsed: 12, Remaining: 18.4},
		Daily:         []domain.DailyUsage{{Day: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), CreditSpend: 0.4, Tokens: 1200}},
		Runway:        domain.RunwayEstimate{Basis: domain.BasisFinite},
	}
}

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	return NewStore(path), path
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	s, path := newStore(t)
	want := fixedSnapshot()

	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.SchemaVersion != want.SchemaVersion {
		t.Errorf("schema version = %d", got.SchemaVersion)
	}
	if !got.GeneratedAt.Equal(want.GeneratedAt) {
		t.Errorf("generated at = %v", got.GeneratedAt)
	}
	if got.Account.Remaining != 18.4 {
		t.Errorf("remaining = %v", got.Account.Remaining)
	}
	if len(got.Daily) != 1 || got.Daily[0].Tokens != 1200 {
		t.Errorf("daily = %+v", got.Daily)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
}

func TestLoadMissingFileReturnsErrNoSnapshot(t *testing.T) {
	s, _ := newStore(t)
	_, err := s.Load()
	if !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("err = %v, want ErrNoSnapshot", err)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	s, path := newStore(t)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := s.Load()
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
}

func TestLoadRejectsUnknownSchemaVersion(t *testing.T) {
	s, path := newStore(t)
	snap := fixedSnapshot()
	snap.SchemaVersion = 99
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = s.Load()
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("err = %v, want ErrUnsupportedVersion", err)
	}
}

func TestSaveIsAtomicAndLeavesNoTempFiles(t *testing.T) {
	s, path := newStore(t)
	if err := s.Save(fixedSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save(fixedSnapshot()); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected only the snapshot file, got %v", names)
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deeper", "snapshot.json")
	s := NewStore(path)

	if err := s.Save(fixedSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
}

func TestSaveDoesNotStoreCredentials(t *testing.T) {
	s, path := newStore(t)
	if err := s.Save(fixedSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, forbidden := range []string{"sk-or-v1", "management", "Authorization", "api_key"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("cache must never contain %q: %s", forbidden, data)
		}
	}
}

func TestSaveToleratesEmptySnapshot(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Save(domain.Snapshot{}); err == nil {
		t.Fatal("expected a version error for a zero-value snapshot")
	}
}
