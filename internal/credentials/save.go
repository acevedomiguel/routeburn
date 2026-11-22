package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) pathFor(kind Kind) string {
	switch kind {
	case KindManagement:
		return s.managementTokenFile
	case KindAPIKey:
		return s.apiKeyFile
	default:
		return ""
	}
}

// Save persists a secret to the keyring, falling back to a 0600 file only when
// the keyring is disabled or unavailable.
func (s *Store) Save(ctx context.Context, kind Kind, secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ErrEmptySecret
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.keyringEnabled {
		if err := s.keyring.Set(ServiceName, string(kind), trimmed); err == nil {
			return nil
		}
	}

	path := s.pathFor(kind)
	if path == "" {
		return fmt.Errorf("credentials: no file path configured for %s", kind)
	}
	return writeSecretFile(path, trimmed)
}

func writeSecretFile(path, secret string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("credentials: create directory: %w", err)
		}
	}
	// An existing file keeps its old mode, so replace it to guarantee 0600.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("credentials: replace %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return fmt.Errorf("credentials: write %s: %w", path, err)
	}
	return nil
}

// Logout removes only RouteBurn-owned keyring entries and fallback files.
func (s *Store) Logout(ctx context.Context) error {
	var failures []error

	if s.keyringEnabled {
		for _, kind := range []Kind{KindManagement, KindAPIKey} {
			err := s.keyring.Delete(ServiceName, string(kind))
			if err != nil && !errors.Is(err, ErrKeyringUnavailable) {
				failures = append(failures, fmt.Errorf("credentials: remove %s from keyring: %w", kind, err))
			}
		}
	}

	for _, path := range []string{s.managementTokenFile, s.apiKeyFile} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("credentials: remove %s: %w", path, err))
		}
	}

	return errors.Join(failures...)
}
