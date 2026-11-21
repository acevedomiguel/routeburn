package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acevedomiguel/routeburn/internal/cache"
)

// Demo mode must respect --json: scripts and screenshots need the contract, not
// an alternate screen that never exits.
func TestDemoModeJSONPrintsSnapshotAndExits(t *testing.T) {
	t.Setenv("ROUTEBURN_DEMO", "1")
	code, stdout, stderr := runCLI(t, "--json")

	if code != 0 {
		t.Fatalf("exit = %d (stderr %q)", code, stderr)
	}
	if strings.Contains(stdout, "\x1b[?1049h") {
		t.Fatal("demo json mode must not enter the alternate screen")
	}
	if strings.Contains(stdout, "Terminal too small") {
		t.Fatal("demo json mode must not render the dashboard")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if decoded["schema_version"].(float64) != 1 {
		t.Errorf("schema_version = %v", decoded["schema_version"])
	}
	if decoded["cached"] != false {
		t.Errorf("demo json must not claim cached data")
	}
	account := decoded["account"].(map[string]any)
	if account["remaining"].(float64) != 42.1 {
		t.Errorf("remaining = %v", account["remaining"])
	}
}

// Demo mode must not require credentials or touch the network.
func TestDemoModeJSONNeedsNoCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ROUTEBURN_DEMO", "1")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	code, stdout, stderr := runCLI(t, "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 without credentials (stderr %q)", code, stderr)
	}
	if !strings.Contains(stdout, "\"runway\"") {
		t.Errorf("stdout = %q", stdout)
	}
}

// Demo mode must not write a cache that later looks like live data.
func TestDemoModeDoesNotWriteCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ROUTEBURN_DEMO", "true")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	if code, _, stderr := runCLI(t, "--json"); code != 0 {
		t.Fatalf("exit = %d (stderr %q)", code, stderr)
	}

	store := cache.NewStore(filepath.Join(home, ".cache", "routeburn", "snapshot.json"))
	if _, err := store.Load(); err == nil {
		t.Fatal("demo mode must not persist a snapshot")
	}
}

func TestDemoModeJSONIsStableAcrossRuns(t *testing.T) {
	t.Setenv("ROUTEBURN_DEMO", "1")
	firstCode, first, _ := runCLI(t, "--json")
	for i := 0; i < 3; i++ {
		code, again, _ := runCLI(t, "--json")
		if code != firstCode {
			t.Fatalf("exit codes differ: %d vs %d", firstCode, code)
		}
		// generated_at moves with the wall clock, so compare the contract keys.
		for _, key := range []string{"\"schema_version\"", "\"account\"", "\"runway\"", "\"last_30_days\""} {
			if strings.Contains(first, key) != strings.Contains(again, key) {
				t.Fatalf("key %s differs between runs", key)
			}
		}
	}
}
