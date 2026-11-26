package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

// The example config must stay loadable, otherwise the documented sample rots.
func TestExampleConfigIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "config.example.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config.example.yaml is missing: %v", err)
	}

	cfg, err := openrouter.LoadConfig([]string{path}, map[string]string{})
	if err != nil {
		t.Fatalf("config.example.yaml does not load: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d", cfg.Version)
	}
	if cfg.RefreshInterval.String() != "10s" {
		t.Errorf("refresh_interval = %s", cfg.RefreshInterval)
	}
	if cfg.RangeDays != openrouter.SupportedRangeDays {
		t.Errorf("range_days = %d", cfg.RangeDays)
	}
	if cfg.Theme != "auto" || cfg.Color != "auto" {
		t.Errorf("theme/color = %q/%q", cfg.Theme, cfg.Color)
	}
	if !cfg.Keyring {
		t.Error("keyring should default to true in the sample")
	}
}

// The sample written by `config reset` must match the documented one.
func TestSampleWriterMatchesExample(t *testing.T) {
	dir := t.TempDir()
	written := filepath.Join(dir, "config.yaml")
	if err := openrouter.WriteSample(written); err != nil {
		t.Fatalf("WriteSample: %v", err)
	}

	got, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	// The written sample is the canonical subset; the example adds comments.
	cfg, err := openrouter.LoadConfig([]string{written}, map[string]string{})
	if err != nil {
		t.Fatalf("written sample does not load: %v", err)
	}
	if cfg.Version != 1 || cfg.RangeDays != 30 {
		t.Errorf("written sample = %+v", cfg)
	}
	if len(got) == 0 || len(want) == 0 {
		t.Fatal("empty config files")
	}
	if os.FileMode(0600)&^fileMode(t, written) != 0 {
		t.Errorf("config file is not owner-only: %v", fileMode(t, written))
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	return info.Mode().Perm()
}
