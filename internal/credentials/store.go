package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Sentinel errors surfaced by the credential store.
var (
	ErrKeyringUnavailable  = errors.New("credentials: the OS keyring is unavailable")
	ErrInsecurePermissions = errors.New("credentials: the fallback file must be readable only by its owner (0600)")
	ErrNoManagementKey     = errors.New("credentials: no management API key is configured")
	ErrEmptySecret         = errors.New("credentials: refusing to store an empty secret")
)

// Kind identifies a credential slot.
type Kind string

// Supported credential slots.
const (
	KindManagement Kind = "management"
	KindAPIKey     Kind = "api-key"
)

// ServiceName is the keyring service namespace owned by RouteBurn.
const ServiceName = "routeburn"

// Environment variables RouteBurn reads credentials from.
const (
	EnvManagementKey = "ROUTEBURN_MANAGEMENT_KEY"
	EnvAPIKey        = "ROUTEBURN_API_KEY"
	EnvOpenRouterKey = "OPENROUTER_API_KEY"
)

// Keyring is the subset of the go-keyring API that the store uses.
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
	Delete(service, user string) error
}

// Creds is the resolved credential pair.
type Creds struct {
	Management string
	APIKey     string
}

// HasManagement reports whether a management key is available.
func (c Creds) HasManagement() bool { return strings.TrimSpace(c.Management) != "" }

// HasAPIKey reports whether a regular API key is available.
func (c Creds) HasAPIKey() bool { return strings.TrimSpace(c.APIKey) != "" }

// Redacted describes which credentials exist without revealing them.
func (c Creds) Redacted() string {
	return fmt.Sprintf("management=%t api_key=%t", c.HasManagement(), c.HasAPIKey())
}

// Options configures a Store. Tests inject a fake keyring and temp paths.
type Options struct {
	Keyring             Keyring
	ManagementTokenFile string
	APIKeyFile          string
	KeyringEnabled      bool
	Env                 map[string]string
}

// Store resolves and persists credentials in the documented order:
// environment, keyring, permission-restricted file, setup flow.
type Store struct {
	keyring             Keyring
	managementTokenFile string
	apiKeyFile          string
	keyringEnabled      bool

	// Env overrides the process environment; nil means os.Getenv.
	Env map[string]string
}

// NewStore builds a store from explicit options.
func NewStore(opts Options) *Store {
	return &Store{
		keyring:             opts.Keyring,
		managementTokenFile: opts.ManagementTokenFile,
		apiKeyFile:          opts.APIKeyFile,
		keyringEnabled:      opts.KeyringEnabled && opts.Keyring != nil,
		Env:                 opts.Env,
	}
}

func (s *Store) lookupEnv(key string) string {
	if s.Env != nil {
		return strings.TrimSpace(s.Env[key])
	}
	return strings.TrimSpace(os.Getenv(key))
}

// Resolve returns every credential the process can use right now. A missing
// management key is not an error here; callers decide how to react.
func (s *Store) Resolve(ctx context.Context) (Creds, error) {
	var creds Creds

	// Environment always wins and is never persisted.
	creds.Management = s.lookupEnv(EnvManagementKey)
	creds.APIKey = s.lookupEnv(EnvAPIKey)

	// A single compatibility fallback for people who already export the
	// standard OpenRouter variable. It can never satisfy the management slot.
	if creds.APIKey == "" {
		creds.APIKey = s.lookupEnv(EnvOpenRouterKey)
	}

	management, err := s.load(ctx, KindManagement, s.managementTokenFile)
	if err != nil {
		return Creds{}, err
	}
	if creds.Management == "" {
		creds.Management = management
	}

	apiKey, err := s.load(ctx, KindAPIKey, s.apiKeyFile)
	if err != nil {
		return Creds{}, err
	}
	if creds.APIKey == "" {
		creds.APIKey = apiKey
	}

	return creds, nil
}

// load reads a slot from the keyring, then from the restricted file.
func (s *Store) load(ctx context.Context, kind Kind, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if s.keyringEnabled {
		if v, err := s.keyring.Get(ServiceName, string(kind)); err == nil {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed, nil
			}
		}
	}

	return readSecretFile(path)
}

// readSecretFile enforces 0600 (no group or world access) before reading.
func readSecretFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("credentials: inspect %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("credentials: %s is a directory", path)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return "", ErrInsecurePermissions
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("credentials: read %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}
