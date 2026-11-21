package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MinRefreshInterval is the floor for a positive refresh interval. A zero
// interval is valid and means automatic refresh is disabled.
const MinRefreshInterval = 10 * time.Second

// SupportedRangeDays is the only range the MVP implements.
const SupportedRangeDays = 30

// Config is the non-secret user configuration.
type Config struct {
	Version         int           `yaml:"version"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	RangeDays       int           `yaml:"range_days"`
	Theme           string        `yaml:"theme"`
	Color           string        `yaml:"color"`
	Keyring         bool          `yaml:"keyring"`
}

// Overrides carries explicit CLI flags, which win over everything else.
type Overrides struct {
	RefreshInterval *time.Duration
	Theme           *string
	Color           *string
	ConfigPath      *string
	NoColor         bool
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	return Config{
		Version:         1,
		RefreshInterval: MinRefreshInterval,
		RangeDays:       SupportedRangeDays,
		Theme:           "auto",
		Color:           "auto",
		Keyring:         true,
	}
}

// Validate enforces the documented constraints.
func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("config: version must be 1, got %d", c.Version)
	}
	if c.RefreshInterval != 0 && c.RefreshInterval < MinRefreshInterval {
		return fmt.Errorf("config: refresh_interval must be %s or greater", MinRefreshInterval)
	}
	if c.RefreshInterval < 0 {
		return fmt.Errorf("config: refresh_interval must not be negative")
	}
	if c.RangeDays != SupportedRangeDays {
		return fmt.Errorf("config: range_days must be %d, got %d", SupportedRangeDays, c.RangeDays)
	}
	switch c.Theme {
	case "auto", "dark", "light":
	default:
		return fmt.Errorf("config: theme must be auto, dark, or light, got %q", c.Theme)
	}
	switch c.Color {
	case "auto", "always", "never":
	default:
		return fmt.Errorf("config: color must be auto, always, or never, got %q", c.Color)
	}
	return nil
}

// AutoRefreshEnabled reports whether the scheduler should poll on a timer.
func (c Config) AutoRefreshEnabled() bool { return c.RefreshInterval > 0 }

// ColorsEnabled resolves the color setting against an explicit --no-color flag.
func (c Config) ColorsEnabled(noColor bool) bool {
	if noColor {
		return false
	}
	switch c.Color {
	case "never":
		return false
	case "always":
		return true
	default:
		return true
	}
}

// Load resolves configuration in the documented precedence order:
// explicit flags, environment variables, YAML file, built-in defaults.
func Load(paths []string, env map[string]string, flags ...Overrides) (Config, error) {
	cfg := Defaults()

	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := mergeFile(&cfg, path); err != nil {
			return Config{}, err
		}
	}

	if err := mergeEnv(&cfg, env); err != nil {
		return Config{}, err
	}

	if len(flags) > 0 {
		if err := mergeFlags(&cfg, flags[0]); err != nil {
			return Config{}, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func mergeFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("config: read %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	return nil
}

func mergeEnv(cfg *Config, env map[string]string) error {
	if env == nil {
		return nil
	}

	if v := strings.TrimSpace(env["ROUTEBURN_REFRESH_INTERVAL"]); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("config: ROUTEBURN_REFRESH_INTERVAL: %w", err)
		}
		cfg.RefreshInterval = d
	}
	if v := strings.TrimSpace(env["ROUTEBURN_THEME"]); v != "" {
		cfg.Theme = v
	}
	if v := strings.TrimSpace(env["ROUTEBURN_COLOR"]); v != "" {
		cfg.Color = v
	}
	if v := strings.TrimSpace(env["ROUTEBURN_KEYRING"]); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("config: ROUTEBURN_KEYRING: %w", err)
		}
		cfg.Keyring = b
	}
	return nil
}

func mergeFlags(cfg *Config, flags Overrides) error {
	if flags.RefreshInterval != nil {
		cfg.RefreshInterval = *flags.RefreshInterval
	}
	if flags.Theme != nil && *flags.Theme != "" {
		cfg.Theme = *flags.Theme
	}
	if flags.Color != nil && *flags.Color != "" {
		cfg.Color = *flags.Color
	}
	if flags.NoColor {
		cfg.Color = "never"
	}
	return nil
}

// WriteSample writes a commented default configuration with restrictive
// permissions, creating the parent directory when needed.
func WriteSample(path string) error {
	if err := os.MkdirAll(parentDir(path), 0o700); err != nil {
		return fmt.Errorf("config: create directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(sampleYAML), 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// sampleYAML is the first-run configuration written by WriteSample.
var sampleYAML = strings.Join([]string{
	"# RouteBurn configuration. Credentials are never stored here.",
	"version: 1",
	"refresh_interval: 10s",
	"range_days: 30",
	"theme: auto",
	"color: auto",
	"keyring: true",
}, "\n") + "\n"
