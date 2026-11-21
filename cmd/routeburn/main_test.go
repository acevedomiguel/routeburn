package main

import (
	"bytes"
	"strings"
	"testing"
)

// runCLI adapts the four-argument run for tests.
func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(""), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunVersionPrintsDevVersion(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--version")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if got, want := stdout, "routeburn version dev\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestRunVersionUsesLdflagsVariable(t *testing.T) {
	old := version
	version = "1.2.3"
	t.Cleanup(func() { version = old })

	_, stdout, _ := runCLI(t, "--version")
	if got, want := strings.TrimSpace(stdout), "routeburn version 1.2.3"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunHelpListsSupportedFlags(t *testing.T) {
	code, stdout, _ := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"Usage:", "routeburn setup", "--json", "--refresh", "--no-cache", "--no-color", "--version", "--help", "Exit codes:"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("help output missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunUnknownFlagIsUsageError(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--nope")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "flag provided but not defined") {
		t.Errorf("stderr = %q, want flag error", stderr)
	}
}

func TestRunUnknownCommandIsUsageError(t *testing.T) {
	code, _, stderr := runCLI(t, "frobnicate")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestRefreshBelowFloorIsRejected(t *testing.T) {
	code, _, stderr := runCLI(t, "--refresh", "5s")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "10s or greater") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestNegativeRefreshIsRejected(t *testing.T) {
	code, _, _ := runCLI(t, "--refresh", "-1s")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestConfigPathPrintsPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	code, stdout, stderr := runCLI(t, "config", "path")
	if code != 0 {
		t.Fatalf("exit code = %d (stderr %q)", code, stderr)
	}
	if !strings.HasSuffix(strings.TrimSpace(stdout), "routeburn/config.yaml") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestConfigPathHonorsExplicitConfig(t *testing.T) {
	code, stdout, _ := runCLI(t, "--config", "/tmp/custom.yaml", "config", "path")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got, want := strings.TrimSpace(stdout), "/tmp/custom.yaml"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestConfigWithoutSubcommandIsUsageError(t *testing.T) {
	code, _, stderr := runCLI(t, "config")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "routeburn config") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestConfigResetWritesSample(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/routeburn/config.yaml"

	code, stdout, stderr := runCLI(t, "--config", path, "config", "reset")
	if code != 0 {
		t.Fatalf("exit code = %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stdout, path) {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestMissingCredentialsExitThreeWithInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/.config")
	t.Setenv("XDG_CACHE_HOME", home+"/.cache")
	t.Setenv("ROUTEBURN_KEYRING", "false")
	t.Setenv("ROUTEBURN_MANAGEMENT_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	code, stdout, stderr := runCLI(t, "--json")
	if code != 3 {
		t.Fatalf("exit code = %d, want 3 (stdout %q, stderr %q)", code, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "routeburn setup") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestParseAcceptsZeroRefresh(t *testing.T) {
	opts, err := parse([]string{"--refresh", "0", "--json"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !opts.refreshSet || opts.refresh != 0 {
		t.Fatalf("opts = %+v", opts)
	}
	if !opts.jsonMode {
		t.Fatal("--json not parsed")
	}
}

func TestParseHelpBeatsFlagErrors(t *testing.T) {
	opts, err := parse([]string{"--nope", "--help"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.command != "help" {
		t.Fatalf("command = %q", opts.command)
	}
}
