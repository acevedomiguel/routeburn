package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/acevedomiguel/routeburn/internal/cache"
)

// A bare invocation without credentials must run setup itself rather than
// telling the user to run a second command. In a non-interactive context it
// still exits with instructions instead of hanging on a prompt.
func TestBareRunWithoutCredentialsStartsSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("ROUTEBURN_DEMO", "")

	code, stdout, stderr := runCLI(t)
	if code != exitCredentials {
		t.Fatalf("exit = %d, want %d (stdout %q, stderr %q)", code, exitCredentials, stdout, stderr)
	}

	// The user must not be told to run a command that would land them here again.
	if strings.Contains(stderr, "Run `routeburn setup`") {
		t.Errorf("stderr still delegates to a second command: %q", stderr)
	}
	if !strings.Contains(stderr, "management key") {
		t.Errorf("stderr should explain what is missing: %q", stderr)
	}
}

// --json keeps the scriptable behaviour: no prompting, non-zero exit.
func TestJSONWithoutCredentialsStillExitsWithoutPrompting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("ROUTEBURN_DEMO", "")

	code, stdout, stderr := runCLI(t, "--json")
	if code != exitCredentials {
		t.Fatalf("exit = %d, want %d", code, exitCredentials)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "routeburn setup") {
		t.Errorf("json mode should still mention setup for scripts: %q", stderr)
	}

	store := cache.NewStore(filepath.Join(home, ".cache", "routeburn", "snapshot.json"))
	if _, err := store.Load(); err == nil {
		t.Fatal("a credential failure must not write a cache")
	}
}
