package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/acevedomiguel/routeburn/internal/app"
	"github.com/acevedomiguel/routeburn/internal/config"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
)

// isolatedHome points every platform directory at a temp dir and disables the
// keyring, so the CLI tests never touch the user's real state.
func isolatedHome(t *testing.T) config.Paths {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("ROUTEBURN_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("ROUTEBURN_REFRESH_INTERVAL", "")

	paths, err := config.DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths: %v", err)
	}
	return paths
}

// seedManagementKey stores a synthetic key in the restricted fallback file, so
// the runtime resolves a credential without an environment override.
func seedManagementKey(t *testing.T, paths config.Paths, key string) {
	t.Helper()
	if err := os.MkdirAll(paths.ConfigDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ManagementTokenFile(), []byte(key), 0o600); err != nil {
		t.Fatal(err)
	}
}

func updateModel(m *app.Model, msg tea.Msg) tea.Cmd {
	_, cmd := m.Update(msg)
	return cmd
}

func TestBuildTUIModelPersistsIntervalToConfig(t *testing.T) {
	paths := isolatedHome(t)
	seedManagementKey(t, paths, "sk-or-v1-stored")

	rt, err := buildRuntime(options{}, paths)
	if err != nil {
		t.Fatalf("buildRuntime: %v", err)
	}
	model := buildTUIModel(rt)
	model.SetSize(100, 40)

	// Open Settings and choose 30s.
	updateModel(model, tea.KeyPressMsg{Code: 's', Text: "s"})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyRight})

	raw, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		t.Fatalf("the interval change did not persist to %s: %v", paths.ConfigFile, err)
	}
	if !strings.Contains(string(raw), "refresh_interval: 30s") {
		t.Errorf("saved config missing the new interval:\n%s", raw)
	}
	if strings.Contains(strings.ToLower(string(raw)), "token") {
		t.Errorf("the saved config must never contain a credential:\n%s", raw)
	}
}

func TestBuildTUIModelWritesToTheExplicitConfigPath(t *testing.T) {
	paths := isolatedHome(t)
	seedManagementKey(t, paths, "sk-or-v1-stored")
	explicit := filepath.Join(t.TempDir(), "custom.yaml")

	rt, err := buildRuntime(options{configPath: explicit}, paths)
	if err != nil {
		t.Fatalf("buildRuntime: %v", err)
	}
	model := buildTUIModel(rt)
	model.SetSize(100, 40)

	updateModel(model, tea.KeyPressMsg{Code: 's', Text: "s"})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyRight})

	raw, err := os.ReadFile(explicit)
	if err != nil {
		t.Fatalf("the interval change did not honour --config: %v", err)
	}
	if !strings.Contains(string(raw), "refresh_interval: 30s") {
		t.Errorf("explicit config missing the new interval:\n%s", raw)
	}
	if _, err := os.Stat(paths.ConfigFile); err == nil {
		t.Error("the default config must not be written when --config is set")
	}
}

func TestBuildTUIModelValidatesAndStoresAReplacementKey(t *testing.T) {
	paths := isolatedHome(t)
	seedManagementKey(t, paths, "sk-or-v1-original")

	srv := fakeAPI(t)
	restore := newAPIClient
	newAPIClient = func(management, apiKey string) (*or.Client, error) {
		return or.NewClient(
			or.WithBaseURL(srv.URL),
			or.WithManagementKey(management),
			or.WithAPIKey(apiKey),
			or.WithVersion(version),
		)
	}
	t.Cleanup(func() { newAPIClient = restore })

	rt, err := buildRuntime(options{}, paths)
	if err != nil {
		t.Fatalf("buildRuntime: %v", err)
	}
	model := buildTUIModel(rt)
	model.SetSize(100, 40)

	updateModel(model, tea.KeyPressMsg{Code: 's', Text: "s"})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyDown})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyEnter})

	const replacement = "sk-or-v1-replacement-value"
	for _, r := range replacement {
		updateModel(model, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	cmd := updateModel(model, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the key returned no command")
	}
	if msg := cmd(); msg != nil {
		updateModel(model, msg)
	}

	raw, err := os.ReadFile(paths.ManagementTokenFile())
	if err != nil {
		t.Fatalf("read the fallback key file: %v", err)
	}
	if strings.TrimSpace(string(raw)) != replacement {
		t.Errorf("stored key = %q, want the replacement", raw)
	}
}

func TestBuildTUIModelExplainsTheEnvironmentKey(t *testing.T) {
	paths := isolatedHome(t)
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "sk-or-v1-from-env")

	rt, err := buildRuntime(options{}, paths)
	if err != nil {
		t.Fatalf("buildRuntime: %v", err)
	}
	model := buildTUIModel(rt)
	model.SetSize(100, 40)

	updateModel(model, tea.KeyPressMsg{Code: 's', Text: "s"})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyDown})
	updateModel(model, tea.KeyPressMsg{Code: tea.KeyEnter})

	view := model.View().Content
	if !strings.Contains(view, "ROUTEBURN_MANAGEMENT_KEY") {
		t.Fatalf("the overlay should explain the environment key:\n%s", view)
	}
	if strings.Contains(view, "sk-or-v1-from-env") {
		t.Fatal("the environment key must never be rendered")
	}
}

func TestBuildTUIModelFooterAdvertisesSettings(t *testing.T) {
	paths := isolatedHome(t)
	seedManagementKey(t, paths, "sk-or-v1-stored")

	rt, err := buildRuntime(options{}, paths)
	if err != nil {
		t.Fatalf("buildRuntime: %v", err)
	}
	model := buildTUIModel(rt)
	model.SetSize(100, 40)

	if view := model.View().Content; !strings.Contains(view, "[s]") {
		t.Errorf("the footer must advertise the settings key:\n%s", view)
	}
}
