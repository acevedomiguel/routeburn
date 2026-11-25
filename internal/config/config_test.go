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
	// Absolute paths from the temp dir keep the assertion portable: on Windows
	// the leading slash is rewritten to a drive-relative path, so a literal
	// "/tmp/..." would never match.
	configHome := filepath.Join(t.TempDir(), "xdg-config")
	cacheHome := filepath.Join(t.TempDir(), "xdg-cache")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)

	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths: %v", err)
	}
	if !strings.HasPrefix(p.ConfigFile, configHome) {
		t.Errorf("config file = %q, want the XDG override", p.ConfigFile)
	}
	if !strings.HasPrefix(p.CacheFile, cacheHome) {
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
	// Windows has no POSIX permission bits; the 0600 request is honoured but
	// cannot be observed, so the assertion only applies where it means something.
	if PermissionsAreEnforced {
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			t.Errorf("permissions = %o, want no group/world access", perm)
		}
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

func TestFormatInterval(t *testing.T) {
	cases := map[time.Duration]string{
		0:                "0s",
		10 * time.Second: "10s",
		30 * time.Second: "30s",
		time.Minute:      "1m",
		5 * time.Minute:  "5m",
		90 * time.Second: "1m30s",
		10 * time.Minute: "10m",
	}
	for d, want := range cases {
		if got := FormatInterval(d); got != want {
			t.Errorf("FormatInterval(%v) = %q, want %q", d, got, want)
		}
	}
}

// SaveRefreshInterval rewrites only refresh_interval and leaves every other key
// in the file exactly as the user set it.
func TestSaveRefreshIntervalPreservesOtherSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "version: 1\nrefresh_interval: 10s\nrange_days: 30\ntheme: dark\ncolor: never\nkeyring: false\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SaveRefreshInterval(path, 5*time.Minute); err != nil {
		t.Fatalf("SaveRefreshInterval: %v", err)
	}

	cfg, err := Load([]string{path}, nil)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if cfg.RefreshInterval != 5*time.Minute {
		t.Errorf("refresh interval = %v, want 5m", cfg.RefreshInterval)
	}
	if cfg.Theme != "dark" {
		t.Errorf("theme = %q, want the file's dark", cfg.Theme)
	}
	if cfg.Color != "never" {
		t.Errorf("color = %q, want the file's never", cfg.Color)
	}
	if cfg.Keyring {
		t.Error("keyring = true, want the file's false to survive")
	}
	if cfg.RangeDays != 30 {
		t.Errorf("range_days = %d, want 30", cfg.RangeDays)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "refresh_interval: 5m") {
		t.Errorf("saved file does not spell the interval readably:\n%s", raw)
	}
	if strings.Contains(strings.ToLower(string(raw)), "token") || strings.Contains(strings.ToLower(string(raw)), "secret") {
		t.Errorf("saved config must never contain a credential:\n%s", raw)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if PermissionsAreEnforced && info.Mode().Perm()&0o077 != 0 {
		t.Errorf("permissions = %o, want no group/world access", info.Mode().Perm())
	}
}

// A missing file is created from the defaults so the dashboard can persist a
// choice before setup has written a configuration.
func TestSaveRefreshIntervalCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	if err := SaveRefreshInterval(path, 30*time.Second); err != nil {
		t.Fatalf("SaveRefreshInterval: %v", err)
	}

	cfg, err := Load([]string{path}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RefreshInterval != 30*time.Second {
		t.Errorf("refresh interval = %v, want 30s", cfg.RefreshInterval)
	}
	if cfg.RangeDays != SupportedRangeDays {
		t.Errorf("range_days = %d, want the default %d", cfg.RangeDays, SupportedRangeDays)
	}
}

// The value is validated before it is written, so an out-of-range choice can
// never leave an unloadable file behind.
func TestSaveRefreshIntervalRejectsInvalidAndLeavesFileIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := WriteSample(path); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := SaveRefreshInterval(path, 5*time.Second); err == nil {
		t.Fatal("expected an error for an interval below the floor")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("a rejected save must not rewrite the file:\nbefore %s\nafter %s", before, after)
	}
}

// A zero interval stays valid and round-trips as "0s".
func TestSaveRefreshIntervalAcceptsZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := SaveRefreshInterval(path, 0); err != nil {
		t.Fatalf("SaveRefreshInterval: %v", err)
	}
	cfg, err := Load([]string{path}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AutoRefreshEnabled() {
		t.Error("a saved zero interval must disable automatic refresh")
	}
}
