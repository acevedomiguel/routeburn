package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/acevedomiguel/routeburn/internal/app"
	"github.com/acevedomiguel/routeburn/internal/config"
	"github.com/acevedomiguel/routeburn/internal/credentials"
	"github.com/acevedomiguel/routeburn/internal/domain"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
)

// doLogout removes locally stored credentials. It never revokes them at
// OpenRouter, because RouteBurn only has local access.
func doLogout(paths config.Paths, opts options, stdout, stderr io.Writer) int {
	store := credentialsStoreFor(paths, opts)
	if err := store.Logout(context.Background()); err != nil {
		fmt.Fprintf(stderr, "routeburn: logout: %v\n", err)
		return exitFailure
	}
	fmt.Fprintln(stdout, "Local RouteBurn credentials removed. Keys remain valid at OpenRouter.")
	return exitOK
}

// doConfigReset restores the sample configuration file.
func doConfigReset(paths config.Paths, opts options, stdout, stderr io.Writer) int {
	path := paths.ConfigFile
	if opts.configPath != "" {
		path = opts.configPath
	}
	if err := config.WriteSample(path); err != nil {
		fmt.Fprintf(stderr, "routeburn: config reset: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "Wrote %s\n", path)
	return exitOK
}

// runJSON prints one snapshot and never opens the dashboard.
func runJSON(rt *runtime, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	now := jsonSnapshotNow()
	opts := app.FetchOptions{Now: now}
	if rt.cache != nil {
		if prev, err := rt.cache.Load(); err == nil {
			opts.Previous = &prev
		}
	}

	snap, err := appFetch(ctx, rt.client, opts)
	cached := false
	if err != nil {
		if rt.cache != nil {
			if prev, loadErr := rt.cache.Load(); loadErr == nil {
				snap = prev
				cached = true
				snap.Warnings = append(snap.Warnings, domain.Warning{
					Section: "Data",
					Message: "live refresh failed; showing cached snapshot: " + credentials.Redact(err.Error()),
				})
			} else {
				fmt.Fprintf(stderr, "routeburn: %v\n", credentials.Redact(err.Error()))
				return exitFailure
			}
		} else {
			fmt.Fprintf(stderr, "routeburn: %v\n", credentials.Redact(err.Error()))
			return exitFailure
		}
	} else if rt.cache != nil {
		if saveErr := rt.cache.Save(snap); saveErr != nil {
			snap.Warnings = append(snap.Warnings, domain.Warning{
				Section: "Cache",
				Message: "could not write the snapshot cache: " + saveErr.Error(),
			})
		}
	}

	out, err := app.MarshalJSON(snap, cached)
	if err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	if _, err := stdout.Write(out); err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}

// runTUI starts the interactive dashboard.
func runTUI(rt *runtime, stdin io.Reader, stdout, stderr io.Writer) int {
	model := app.New(app.Deps{
		Config: app.Config{
			RefreshInterval: rt.cfg.RefreshInterval,
			NoCache:         rt.opts.noCache,
			NoColor:         !rt.cfg.ColorsEnabled(rt.opts.noColor),
			RangeDays:       rt.cfg.RangeDays,
		},
		Creds: app.Creds{
			Management: rt.creds.Management,
			APIKey:     rt.creds.APIKey,
		},
		API:   rt.client,
		Cache: rt.cache,
		Now:   func() time.Time { return time.Now().UTC() },
	})

	program := tea.NewProgram(model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}

// doSetup stores credentials after validating them with a live request.
func doSetup(paths config.Paths, opts options, stdin io.Reader, stdout, stderr io.Writer) int {
	fmt.Fprint(stdout, setupWelcome())

	if strings.TrimSpace(os.Getenv(credentials.EnvManagementKey)) != "" {
		fmt.Fprintln(stderr, "routeburn: ROUTEBURN_MANAGEMENT_KEY is set; nothing to set up.")
		fmt.Fprintln(stderr, "Unset it to store a key locally, or run `routeburn` to use the environment key.")
		return exitOK
	}

	if isTerminal(stdin) {
		// The interactive wizard validates and stores the keys itself.
		return runSetupInteractive(paths, opts, stdin, stdout, stderr)
	}

	management, apiKey, code := readSetupLines(stdin, stdout, stderr)
	if code != exitOK {
		return code
	}
	if management == "" {
		fmt.Fprintln(stderr, "routeburn: no management key was provided.")
		return exitCredentials
	}

	client, err := or.NewClient(
		or.WithManagementKey(management),
		or.WithAPIKey(apiKey),
		or.WithVersion(version),
	)
	if err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitUsage
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := client.Credits(ctx); err != nil {
		fmt.Fprintf(stderr, "routeburn: credential validation failed: %s\n", credentials.Redact(err.Error()))
		fmt.Fprintln(stderr, "Nothing was saved. Run `routeburn setup` again when the key works.")
		return exitFailure
	}

	store := credentialsStoreFor(paths, opts)
	if err := store.Save(ctx, credentials.KindManagement, management); err != nil {
		fmt.Fprintf(stderr, "routeburn: could not store the management key: %s\n", credentials.Redact(err.Error()))
		return exitFailure
	}
	if apiKey != "" {
		if err := store.Save(ctx, credentials.KindAPIKey, apiKey); err != nil {
			fmt.Fprintf(stderr, "routeburn: could not store the API key: %s\n", credentials.Redact(err.Error()))
			return exitFailure
		}
	}

	cfgPath := paths.ConfigFile
	if opts.configPath != "" {
		cfgPath = opts.configPath
	}
	if err := config.WriteSample(cfgPath); err != nil {
		fmt.Fprintf(stderr, "routeburn: credentials saved, but the configuration file could not be written: %v\n", err)
		return exitFailure
	}

	fmt.Fprintf(stdout, "\nCredentials saved. Configuration written to %s\n", cfgPath)
	fmt.Fprintln(stdout, "Run `routeburn` to open the dashboard.")
	return exitOK
}

// demoMode reports whether synthetic fixtures should be shown instead of live data.
func demoMode() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("ROUTEBURN_DEMO")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// runDemo renders the dashboard with synthetic fixtures and no network access.
func runDemo(opts options, stdin io.Reader, stdout, stderr io.Writer) int {
	now := time.Now().UTC()

	model := app.New(app.Deps{
		Config: app.Config{
			RefreshInterval: 0,
			NoCache:         true,
			NoColor:         opts.noColor,
			RangeDays:       30,
		},
		Creds: app.Creds{Management: "demo"},
		Now:   func() time.Time { return now },
		Fetch: func(context.Context, app.FetchOptions) (domain.Snapshot, error) {
			return app.DemoSnapshot(now), nil
		},
	})

	program := tea.NewProgram(model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}

// runDemoJSON prints the synthetic snapshot in the same contract as --json.
func runDemoJSON(stdout io.Writer) int {
	snap := app.DemoSnapshot(time.Now().UTC())

	payload, err := app.MarshalJSON(snap, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	if _, err := stdout.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "routeburn: %v\n", err)
		return exitFailure
	}
	return exitOK
}

// isTerminal reports whether r is an interactive terminal.
func isTerminal(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func setupWelcome() string {
	return "RouteBurn\n\n" +
		"A read-only OpenRouter dashboard. RouteBurn never sends inference requests.\n\n" +
		"Account balance and analytics require a Management API key.\n" +
		"Create one at https://openrouter.ai/settings/management_keys\n\n"
}

// readSetupLines reads the keys from a non-interactive stdin.
func readSetupLines(stdin io.Reader, stdout, stderr io.Writer) (string, string, int) {
	scanner := bufio.NewScanner(stdin)
	fmt.Fprint(stdout, "Management key: ")
	if !scanner.Scan() {
		fmt.Fprintln(stderr, "\nrouteburn: no OpenRouter management key was provided.")
		fmt.Fprintln(stderr, "RouteBurn needs one for account credits and analytics; create it at")
		fmt.Fprintln(stderr, "https://openrouter.ai/settings/management_keys and run this again.")
		return "", "", exitCredentials
	}
	management := strings.TrimSpace(scanner.Text())
	fmt.Fprint(stdout, "API key (optional, press Enter to skip): ")
	apiKey := ""
	if scanner.Scan() {
		apiKey = strings.TrimSpace(scanner.Text())
	}
	fmt.Fprintln(stdout)
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return "", "", exitFailure
	}
	return management, apiKey, exitOK
}

// credentialsStoreFor builds the credential store for the given options.
func credentialsStoreFor(paths config.Paths, opts options) *credentials.Store {
	return credentials.NewStore(credentials.Options{
		Keyring:             osKeyringFor(config.Config{Keyring: keyringRequested()}),
		ManagementTokenFile: paths.ManagementTokenFile(),
		APIKeyFile:          paths.APIKeyFile(),
		KeyringEnabled:      keyringRequested(),
	})
}

// keyringRequested reports whether the user wants the OS keyring used.
func keyringRequested() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("ROUTEBURN_KEYRING")))
	return value == "" || value == "1" || value == "true" || value == "yes" || value == "on"
}
