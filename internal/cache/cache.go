package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// Sentinel errors for the cache.
var (
	ErrNoSnapshot         = errors.New("cache: no snapshot has been saved yet")
	ErrMalformed          = errors.New("cache: the cached snapshot is not valid JSON")
	ErrUnsupportedVersion = errors.New("cache: the cached snapshot uses an unsupported schema version")
)

// Store reads and writes the versioned snapshot cache.
type Store struct {
	path string

	// Now is injectable so tests control the freshness math.
	Now func() time.Time
}

// NewStore builds a cache store rooted at path.
func NewStore(path string) *Store {
	return &Store{path: path, Now: func() time.Time { return time.Now().UTC() }}
}

// Path reports the cache file location.
func (s *Store) Path() string { return s.path }

// Load reads the snapshot. A missing file is ErrNoSnapshot, and a file written
// by a newer or older schema is ErrUnsupportedVersion so callers never render
// data they do not understand.
func (s *Store) Load() (domain.Snapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Snapshot{}, ErrNoSnapshot
		}
		return domain.Snapshot{}, fmt.Errorf("cache: read %s: %w", s.path, err)
	}

	var snap domain.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return domain.Snapshot{}, ErrMalformed
	}
	if snap.SchemaVersion != domain.SchemaVersion {
		return domain.Snapshot{}, fmt.Errorf("%w: found %d, want %d", ErrUnsupportedVersion, snap.SchemaVersion, domain.SchemaVersion)
	}
	return snap, nil
}

// Save writes the snapshot atomically: a temporary file in the same directory
// is fsynced and renamed over the target so a crash never leaves half a file.
func (s *Store) Save(snap domain.Snapshot) error {
	if snap.SchemaVersion != domain.SchemaVersion {
		return fmt.Errorf("cache: refusing to write schema version %d, want %d", snap.SchemaVersion, domain.SchemaVersion)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: encode snapshot: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cache: create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("cache: create temporary file: %w", err)
	}
	tmpName := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("cache: set permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("cache: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("cache: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cache: close: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cache: rename: %w", err)
	}
	return nil
}

// Age reports how old the cached snapshot is relative to the injected clock.
func (s *Store) Age(snap domain.Snapshot) time.Duration {
	if snap.GeneratedAt.IsZero() {
		return 0
	}
	return s.Now().Sub(snap.GeneratedAt)
}
