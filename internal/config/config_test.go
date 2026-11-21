package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultsAreApplied(t *testing.T) {
	cfg, err := Load(nil, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.RefreshInterval != 10*time.Second {
		t.Errorf("refresh interval = %v, want 10s", cfg.RefreshInterval)
	}
	if cfg.RangeDays != 30 {
		t.Errorf("range days = %d, want 30", cfg.RangeDays)
	}
	if cfg.Theme != "auto" || cfg.Color != "auto" {
		t.Errorf("theme/color = %q/%q, want auto/auto", cfg.Theme, cfg.Color)
	}
	if !cfg.Keyring {
		t.Error("keyring must default to enabled")
	}
}

func TestValidateRejectsUnsupportedValues(t *testing.T) {
	tests := map[string]Config{
		"wrong version":       {Version: 2, RefreshInterval: 10 * time.Second, RangeDays: 30, Theme: "auto", Color: "auto"},
		"refresh below floor": {Version: 1, RefreshInterval: 9 * time.Second, RangeDays: 30, Theme: "auto", Color: "auto"},
		"negative refresh":    {Version: 1, RefreshInterval: -time.Second, RangeDays: 30, Theme: "auto", Color: "auto"},
		"range days":          {Version: 1, RefreshInterval: 10 * time.Second, RangeDays: 7, Theme: "auto", Color: "auto"},
		"unknown theme":       {Version: 1, RefreshInterval: 10 * time.Second, RangeDays: 30, Theme: "neon", Color: "auto"},
		"unknown color":       {Version: 1, RefreshInterval: 10 * time.Second, RangeDays: 30, Theme: "auto", Color: "sometimes"},
	}

	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected an error for %+v", cfg)
			}
		})
	}
}

func TestZeroRefreshIntervalDisablesAutoRefreshButStaysValid(t *testing.T) {
	cfg := Config{Version: 1, RefreshInterval: 0, RangeDays: 30, Theme: "auto", Color: "auto"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.AutoRefreshEnabled() {
		t.Error("a zero interval must disable automatic refresh")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nsurprise: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load([]string{path}, nil)
	if err == nil {
		t.Fatal("expected a strict decode error for an unknown field")
	}
	if !strings.Contains(err.Error(), "surprise") {
		t.Errorf("error should name the offending field: %v", err)
	}
}

func TestMissingConfigFileIsNotAnError(t *testing.T) {
	cfg, err := Load([]string{filepath.Join(t.TempDir(), "absent.yaml")}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RefreshInterval != 10*time.Second {
		t.Errorf("refresh interval = %v, want defaults to survive a missing file", cfg.RefreshInterval)
	}
}

func TestPrecedenceYAMLThenEnvThenFlags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nrefresh_interval: 20s\ntheme: dark\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{"ROUTEBURN_REFRESH_INTERVAL": "30s", "ROUTEBURN_THEME": "light"}
	cfg, err := Load([]string{path}, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RefreshInterval != 30*time.Second {
		t.Errorf("refresh interval = %v, want 30s from env", cfg.RefreshInterval)
	}
	if cfg.Theme != "light" {
		t.Errorf("theme = %q, want light from env", cfg.Theme)
	}

	flags := Overrides{RefreshInterval: durationPtr(45 * time.Second), Theme: strPtr("dark")}
	cfg, err = Load([]string{path}, env, flags)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RefreshInterval != 45*time.Second {
		t.Errorf("refresh interval = %v, want 45s from flags", cfg.RefreshInterval)
	}
	if cfg.Theme != "dark" {
		t.Errorf("theme = %q, want dark from flags", cfg.Theme)
	}
}

func TestInvalidEnvValueIsRejected(t *testing.T) {
	_, err := Load(nil, map[string]string{"ROUTEBURN_REFRESH_INTERVAL": "soon"})
	if err == nil {
		t.Fatal("expected an error for an unparsable interval")
	}
}

func TestFlagRefreshBelowFloorIsRejected(t *testing.T) {
	_, err := Load(nil, nil, Overrides{RefreshInterval: durationPtr(2 * time.Second)})
	if err == nil {
		t.Fatal("expected an error for a 2s flag override")
	}
}

func TestPathsUseUserDirs(t *testing.T) {
	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths: %v", err)
	}
	if !strings.HasSuffix(p.ConfigFile, filepath.Join("routeburn", "config.yaml")) {
		t.Errorf("config file = %q", p.ConfigFile)
	}
	if !strings.HasSuffix(p.CacheFile, filepath.Join("routeburn", "snapshot.json")) {
		t.Errorf("cache file = %q", p.CacheFile)
	}
}

func TestPathsHonorXDGEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache")

	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths: %v", err)
	}
	if !strings.HasPrefix(p.ConfigFile, "/tmp/xdg-config") {
		t.Errorf("config file = %q, want the XDG override", p.ConfigFile)
	}
	if !strings.HasPrefix(p.CacheFile, "/tmp/xdg-cache") {
		t.Errorf("cache file = %q, want the XDG override", p.CacheFile)
	}
}

func TestWriteSampleIsLoadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := WriteSample(path); err != nil {
		t.Fatalf("WriteSample: %v", err)
	}

	cfg, err := Load([]string{path}, nil)
	if err != nil {
		t.Fatalf("Load after WriteSample: %v", err)
	}
	if cfg.RefreshInterval != 10*time.Second {
		t.Errorf("refresh interval = %v, want 10s", cfg.RefreshInterval)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("permissions = %o, want no group/world access", perm)
	}
}

func TestLoadIgnoresEmptyEnvAndEmptyFlags(t *testing.T) {
	empty := ""
	cfg, err := Load(nil, map[string]string{"ROUTEBURN_THEME": ""}, Overrides{Theme: &empty})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "auto" {
		t.Errorf("theme = %q, want the default when overrides are empty", cfg.Theme)
	}
}

func TestUnreadableConfigPathIsReported(t *testing.T) {
	// A directory can never be decoded as configuration; that must surface as
	// an error rather than being mistaken for a missing file.
	dir := t.TempDir()
	_, err := Load([]string{dir}, nil)
	if err == nil {
		t.Fatal("expected an error when the config path is not a regular file")
	}
}

func durationPtr(d time.Duration) *time.Duration { return &d }
func strPtr(s string) *string                    { return &s }
