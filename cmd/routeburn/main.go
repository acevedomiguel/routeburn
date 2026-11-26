package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

// Exit codes documented in the CLI contract.
const (
	exitOK          = 0
	exitFailure     = 1
	exitUsage       = 2
	exitCredentials = 3
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// options is the parsed command line.
type options struct {
	showVersion bool
	jsonMode    bool
	noCache     bool
	noColor     bool
	configPath  string
	refresh     time.Duration
	refreshSet  bool

	// command is one of dashboard, setup, logout, config-path,
	// config-reset, version, help
	command string
}

// usageError marks failures caused by the command line rather than the
// environment, so they exit 2 with an explanatory line.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// run executes the CLI with explicit streams so tests never touch the real
// terminal or the process exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, err := parse(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		var ue *usageError
		if errors.As(err, &ue) {
			fmt.Fprintf(stderr, "routeburn: %s\n", ue.msg)
		}
		return exitUsage
	}

	switch opts.command {
	case "version":
		fmt.Fprintf(stdout, "routeburn version %s\n", version)
		return exitOK
	case "help":
		printUsage(stdout)
		return exitOK
	case "demo":
		// The demo never touches configuration, credentials, or the network.
		return runDemo(stdin, stdout, stderr)
	}

	paths, err := openrouter.DefaultPaths()
	if err != nil {
		fmt.Fprintf(stderr, "routeburn: %v\n", err)
		return exitFailure
	}

	switch opts.command {
	case "config-path":
		if opts.configPath != "" {
			fmt.Fprintln(stdout, opts.configPath)
			return exitOK
		}
		fmt.Fprintln(stdout, paths.ConfigFile)
		return exitOK
	case "config-reset":
		return doConfigReset(paths, opts, stdout, stderr)
	case "logout":
		return doLogout(paths, opts, stdout, stderr)
	case "setup":
		return doSetup(paths, opts, stdin, stdout, stderr)
	}

	runtime, err := buildRuntime(opts, paths)
	if err != nil {
		if !errors.Is(err, errNoCredentials) {
			fmt.Fprintf(stderr, "routeburn: %v\n", err)
			return exitUsage
		}

		// A script must never block on a prompt, so JSON mode explains instead.
		if opts.jsonMode {
			fmt.Fprint(stderr, setupInstructions())
			return exitCredentials
		}

		// First run: do the setup here instead of sending the user to a second
		// command. The wizard is interactive when stdin is a terminal and reads
		// from the pipe otherwise.
		if code := doSetup(paths, opts, stdin, stdout, stderr); code != exitOK {
			return code
		}

		runtime, err = buildRuntime(opts, paths)
		if err != nil {
			if errors.Is(err, errNoCredentials) {
				fmt.Fprint(stderr, setupInstructions())
				return exitCredentials
			}
			fmt.Fprintf(stderr, "routeburn: %v\n", err)
			return exitUsage
		}
	}

	if opts.jsonMode {
		return runJSON(runtime, stdout, stderr)
	}
	return runTUI(runtime, stdin, stdout, stderr)
}

// parse turns argv into options, honoring help requests before flag errors.
func parse(args []string, stderr io.Writer) (options, error) {
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			return options{command: "help"}, nil
		}
	}

	fs := flag.NewFlagSet("routeburn", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts options
	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&opts.jsonMode, "json", false, "print one snapshot as JSON and exit")
	fs.BoolVar(&opts.noCache, "no-cache", false, "do not read or write cached snapshots")
	fs.BoolVar(&opts.noColor, "no-color", false, "disable colored output")
	fs.StringVar(&opts.configPath, "config", "", "use an explicit configuration file")
	fs.DurationVar(&opts.refresh, "refresh", 0, "override the refresh interval (0 disables auto-refresh)")

	if err := fs.Parse(args); err != nil {
		// The flag package already wrote its own message to stderr.
		return options{}, err
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "refresh" {
			opts.refreshSet = true
		}
	})

	if opts.showVersion {
		opts.command = "version"
		return opts, nil
	}

	rest := fs.Args()
	switch {
	case len(rest) == 0:
		opts.command = "dashboard"
	case rest[0] == "setup" && len(rest) == 1:
		opts.command = "setup"
	case rest[0] == "demo" && len(rest) == 1:
		opts.command = "demo"
	case rest[0] == "logout" && len(rest) == 1:
		opts.command = "logout"
	case rest[0] == "help":
		opts.command = "help"
	case rest[0] == "config" && len(rest) == 1:
		return options{}, &usageError{msg: "usage: routeburn config [path|reset]"}
	case rest[0] == "config" && len(rest) == 2 && rest[1] == "path":
		opts.command = "config-path"
	case rest[0] == "config" && len(rest) == 2 && rest[1] == "reset":
		opts.command = "config-reset"
	default:
		return options{}, &usageError{msg: fmt.Sprintf("unknown command %q", strings.Join(rest, " "))}
	}

	if !opts.refreshSet {
		return opts, nil
	}
	if opts.refresh < 0 {
		return options{}, &usageError{msg: "--refresh must not be negative"}
	}
	if opts.refresh > 0 && opts.refresh < openrouter.MinRefreshInterval {
		return options{}, &usageError{msg: fmt.Sprintf("--refresh must be %s or greater, or 0 to disable", openrouter.MinRefreshInterval)}
	}
	return opts, nil
}

// runtime bundles everything the dashboard and JSON paths need.
type runtime struct {
	opts   options
	cfg    openrouter.Config
	creds  openrouter.Creds
	store  *openrouter.CredentialStore
	client *openrouter.Client
	cache  *openrouter.CacheStore
	now    func() time.Time
	paths  openrouter.Paths
}

// configTarget is the configuration file a Settings change is written back to:
// the explicit --config path when given, the platform default otherwise.
func (r *runtime) configTarget() string {
	if r.opts.configPath != "" {
		return r.opts.configPath
	}
	return r.paths.ConfigFile
}

// errNoCredentials signals the documented exit code 3.
var errNoCredentials = errors.New("no credentials configured")

// newAPIClient builds the OpenRouter client from explicit credentials. Tests
// replace it so no test reaches the network.
var newAPIClient = func(management, apiKey string) (*openrouter.Client, error) {
	return openrouter.NewClient(
		openrouter.WithManagementKey(management),
		openrouter.WithAPIKey(apiKey),
		openrouter.WithVersion(version),
	)
}

// buildRuntime resolves configuration and credentials. It returns
// errNoCredentials (exit 3) when the management key is missing.
func buildRuntime(opts options, paths openrouter.Paths) (*runtime, error) {
	env := environment()

	configPaths := []string{paths.ConfigFile}
	if opts.configPath != "" {
		configPaths = []string{opts.configPath}
	}

	overrides := openrouter.Overrides{NoColor: opts.noColor}
	if opts.refreshSet {
		interval := opts.refresh
		overrides.RefreshInterval = &interval
	}

	cfg, err := openrouter.LoadConfig(configPaths, env, overrides)
	if err != nil {
		return nil, err
	}

	store := openrouter.NewCredentialStore(openrouter.CredentialOptions{
		Keyring:             osKeyringFor(cfg),
		ManagementTokenFile: paths.ManagementTokenFile(),
		APIKeyFile:          paths.APIKeyFile(),
		KeyringEnabled:      cfg.Keyring,
	})
	creds, err := store.Resolve(context.Background())
	if err != nil {
		return nil, err
	}
	if !creds.HasManagement() {
		return nil, errNoCredentials
	}

	client, err := newAPIClient(creds.Management, creds.APIKey)
	if err != nil {
		return nil, err
	}

	var cacheStore *openrouter.CacheStore
	if !opts.noCache {
		cacheStore = openrouter.NewCacheStore(paths.CacheFile)
	}

	return &runtime{
		opts:   opts,
		cfg:    cfg,
		creds:  creds,
		store:  store,
		client: client,
		cache:  cacheStore,
		now:    func() time.Time { return time.Now().UTC() },
		paths:  paths,
	}, nil
}

// environment collects the variables the configuration layer understands.
func environment() map[string]string {
	env := map[string]string{}
	for _, key := range []string{
		"ROUTEBURN_REFRESH_INTERVAL",
		"ROUTEBURN_THEME",
		"ROUTEBURN_COLOR",
		"ROUTEBURN_KEYRING",
	} {
		if v, ok := os.LookupEnv(key); ok {
			env[key] = v
		}
	}
	return env
}

// osKeyringFor returns the platform keyring only when it is enabled so that a
// disabled keyring never triggers a D-Bus connection.
func osKeyringFor(cfg openrouter.Config) openrouter.Keyring {
	if !cfg.Keyring {
		return nil
	}
	return openrouter.NewOSKeyring()
}

func setupInstructions() string {
	return "routeburn: no OpenRouter management key found.\n\n" +
		"Run `routeburn setup` to store one, or export ROUTEBURN_MANAGEMENT_KEY.\n" +
		"Create a management key at https://openrouter.ai/settings/management_keys\n"
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage:\n"+
		"  routeburn                 Start the dashboard\n"+
		"  routeburn setup           Store OpenRouter credentials\n"+
		"  routeburn logout          Remove locally stored credentials\n"+
		"  routeburn config path     Print the configuration file path\n"+
		"  routeburn config reset    Restore the sample configuration\n"+
		"  routeburn --json          Print one snapshot as JSON and exit\n"+
		"\nFlags:\n"+
		"  --refresh <duration>      Refresh interval (minimum %s, 0 disables)\n"+
		"  --json                    Print one snapshot as JSON and exit\n"+
		"  --no-cache                Do not read or write cached snapshots\n"+
		"  --no-color                Disable colored output\n"+
		"  --config <path>           Use an explicit configuration file\n"+
		"  --version                 Print version and exit\n"+
		"  --help                    Print this help and exit\n"+
		"\nExit codes:\n"+
		"  0 success   1 runtime or API failure   2 usage error   3 missing credentials\n"+
		"\nEngine: "+engineModule+" "+engineVersion()+"\n",
		openrouter.MinRefreshInterval)
}

// jsonSnapshotNow is the clock used by the JSON path; tests replace it.
var jsonSnapshotNow = func() time.Time { return time.Now() }

// appFetch is the snapshot builder used by both output paths.
var appFetch = openrouter.Fetch
