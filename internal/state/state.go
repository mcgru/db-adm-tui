// Package state persists per-profile workspace dimensions under the XDG state
// directory. It stores only integers; no credentials ever reach this file.
package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

const schemaVersion = 1

type document struct {
	Version int                              `json:"version"`
	Layouts map[string]domain.LayoutSettings `json:"layouts"`
}

// Store is a file-backed ports.LayoutStore.
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore returns a store at path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// NewDefaultStore uses $XDG_STATE_HOME/db-adm-tui/layouts.json, defaulting to
// ~/.local/state/db-adm-tui/layouts.json.
func NewDefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

// DefaultPath returns the XDG state file for workspace dimensions.
func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "db-adm-tui", "layouts.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "db-adm-tui", "layouts.json"), nil
}

// Path returns the backing file.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Load returns the saved dimensions for profile. A missing file is an empty
// layout, not an error.
func (s *Store) Load(ctx context.Context, profile string) (domain.LayoutSettings, error) {
	if s == nil || s.path == "" {
		return domain.LayoutSettings{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readLocked()
	if err != nil {
		return domain.LayoutSettings{}, err
	}
	return doc.Layouts[profile], nil
}

// Save upserts the dimensions for profile and atomically replaces the state
// file with 0600 permissions.
func (s *Store) Save(ctx context.Context, profile string, settings domain.LayoutSettings) error {
	if s == nil || s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readLocked()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if doc.Layouts == nil {
		doc.Layouts = map[string]domain.LayoutSettings{}
	}
	doc.Version = schemaVersion
	doc.Layouts[profile] = settings
	return s.writeLocked(doc)
}

func (s *Store) readLocked() (document, error) {
	doc := document{Version: schemaVersion, Layouts: map[string]domain.LayoutSettings{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return document{}, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return document{}, err
	}
	if doc.Version != schemaVersion {
		return document{}, errors.New("unsupported layout state version")
	}
	if doc.Layouts == nil {
		doc.Layouts = map[string]domain.LayoutSettings{}
	}
	return doc, nil
}

func (s *Store) writeLocked(doc document) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".layouts-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, s.path)
}
