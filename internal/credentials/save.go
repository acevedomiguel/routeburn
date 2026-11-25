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
// the keyring is disabled or cannot take the write.
//
// Resolution reads the keyring before the file, so a successful save also
// clears the other backend. Leaving a stale value behind would let a later
// resolve silently return the credential that was just replaced instead of the
// one that was saved.
func (s *Store) Save(ctx context.Context, kind Kind, secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ErrEmptySecret
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.keyringEnabled && s.keyring.Set(ServiceName, string(kind), trimmed) == nil {
		return s.removeFallback(kind)
	}

	// Write the value durably before touching the keyring, so a failure here
	// leaves the previously resolvable credential intact.
	if err := s.writeFallback(kind, trimmed); err != nil {
		return err
	}
	if s.keyringEnabled {
		return s.clearStaleKeyring(kind)
	}
	return nil
}

// removeFallback deletes the fallback file after the keyring took the value, so
// the file cannot shadow the new value if the keyring later becomes
// unavailable. A missing file is not an error.
func (s *Store) removeFallback(kind Kind) error {
	path := s.pathFor(kind)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("credentials: remove stale fallback %s: %w", path, err)
	}
	return nil
}

// writeFallback stores the secret in the permission-restricted file.
func (s *Store) writeFallback(kind Kind, secret string) error {
	path := s.pathFor(kind)
	if path == "" {
		return fmt.Errorf("credentials: no file path configured for %s", kind)
	}
	return writeSecretFile(path, secret)
}

// clearStaleKeyring removes the keyring entry after the value was written to
// the fallback file. Resolution prefers a readable keyring entry, so one that
// survived the delete would shadow the file; that is reported as a failure
// rather than a success that will not take effect. The OS keyring reports every
// failure as "unavailable", so the entry is probed rather than inspected.
func (s *Store) clearStaleKeyring(kind Kind) error {
	if err := s.keyring.Delete(ServiceName, string(kind)); err != nil {
		if _, getErr := s.keyring.Get(ServiceName, string(kind)); getErr == nil {
			return fmt.Errorf("credentials: could not replace the stale %s entry in the OS keyring", kind)
		}
	}
	return nil
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
