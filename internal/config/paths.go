package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths holds the resolved locations RouteBurn reads and writes.
type Paths struct {
	ConfigDir  string
	ConfigFile string
	CacheDir   string
	CacheFile  string
}

// ManagementTokenFile is the permission-restricted fallback for the
// management key. It is used only when the OS keyring is unavailable.
func (p Paths) ManagementTokenFile() string {
	return filepath.Join(p.ConfigDir, "management-token")
}

// APIKeyFile is the fallback file for a regular API key.
func (p Paths) APIKeyFile() string {
	return filepath.Join(p.ConfigDir, "api-key")
}

// DefaultPaths resolves platform-appropriate directories through the standard
// library so XDG and OS overrides are honoured automatically.
func DefaultPaths() (Paths, error) {
	configDir, err := userDir("XDG_CONFIG_HOME", os.UserConfigDir)
	if err != nil {
		return Paths{}, fmt.Errorf("config: resolve user config dir: %w", err)
	}
	cacheDir, err := userDir("XDG_CACHE_HOME", os.UserCacheDir)
	if err != nil {
		return Paths{}, fmt.Errorf("config: resolve user cache dir: %w", err)
	}

	p := Paths{
		ConfigDir: filepath.Join(configDir, "routeburn"),
		CacheDir:  filepath.Join(cacheDir, "routeburn"),
	}
	p.ConfigFile = filepath.Join(p.ConfigDir, "config.yaml")
	p.CacheFile = filepath.Join(p.CacheDir, "snapshot.json")
	return p, nil
}

func parentDir(path string) string {
	dir := filepath.Dir(path)
	if dir == "" {
		return "."
	}
	return dir
}

// userDir prefers an explicit XDG directory when the user set one, and falls
// back to the platform-native location otherwise. On macOS the standard library
// ignores XDG, but users who set it expect it to win.
func userDir(envVar string, fallback func() (string, error)) (string, error) {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v, nil
	}
	return fallback()
}
