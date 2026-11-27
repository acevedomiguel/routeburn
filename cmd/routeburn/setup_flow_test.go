package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/burnblocks/openrouter"
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

	store := openrouter.NewCacheStore(filepath.Join(home, ".cache", "routeburn", "snapshot.json"))
	if _, err := store.Load(); err == nil {
		t.Fatal("a credential failure must not write a cache")
	}
}

// The wizard's view is a tea.View too, and it must not request the alternate
// screen: on a 24-row terminal the password step would scroll off.
func TestSetupModelViewIsAV2View(t *testing.T) {
	m := newSetupModel(options{}, openrouter.Paths{})
	m.width, m.height = 80, 24
	v := m.View()
	if v.AltScreen {
		t.Error("the setup wizard must not take the alternate screen")
	}
	if !strings.Contains(v.Content, "RouteBurn setup") {
		t.Errorf("setup view = %q", v.Content)
	}
}

// The wizard's text input accepts a bracketed paste, so a management key can be
// pasted at the first-run prompt as well as typed.
func TestSetupModelAcceptsPastedManagementKey(t *testing.T) {
	m := newSetupModel(options{}, openrouter.Paths{})
	const key = "sk-or-v1-synthetic-pasted-key"

	updated, _ := m.Update(tea.PasteMsg{Content: key})
	got, ok := updated.(*setupModel)
	if !ok {
		t.Fatalf("update returned %T, want *setupModel", updated)
	}
	if got.input.Value() != key {
		t.Fatalf("input = %q, want the pasted key", got.input.Value())
	}
}
