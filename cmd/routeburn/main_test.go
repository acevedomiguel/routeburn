package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersionPrintsDevVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if got, want := stdout.String(), "routeburn version dev\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionUsesLdflagsVariable(t *testing.T) {
	old := version
	version = "1.2.3"
	t.Cleanup(func() { version = old })

	var stdout, stderr bytes.Buffer
	run([]string{"--version"}, &stdout, &stderr)

	if got, want := strings.TrimSpace(stdout.String()), "routeburn version 1.2.3"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunHelpListsOnlySupportedFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	for _, want := range []string{"Usage:", "routeburn", "--version", "--help"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"--json", "--refresh", "setup"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("help output advertises unimplemented option %q:\n%s", unwanted, out)
		}
	}
}

func TestRunUnknownFlagIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--nope"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Errorf("stderr = %q, want flag error", stderr.String())
	}
}
