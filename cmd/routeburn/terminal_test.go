package main

import (
	"bytes"
	"os"
	"testing"
)

// /dev/null is a character device, so a mode check is not enough; stdin must be
// a real terminal before the wizard takes over.
func TestIsTerminalRejectsDevNull(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer file.Close()

	if isTerminal(file) {
		t.Fatal("/dev/null must not be treated as a terminal")
	}
}

func TestIsTerminalRejectsPipes(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	w.Close()

	if isTerminal(r) {
		t.Fatal("a pipe must not be treated as a terminal")
	}
}

// A first launch with no usable stdin must report the missing key, not block.
func TestBareRunWithDevNullStdinDoesNotBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/cfg")
	t.Setenv("XDG_CACHE_HOME", home+"/cache")
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer file.Close()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := run(nil, file, stdout, stderr)

	if code == 0 {
		t.Fatalf("expected a non-zero exit, got 0 (stderr %q)", stderr.String())
	}
}
