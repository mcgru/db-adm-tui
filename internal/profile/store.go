package profile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

// SchemaVersion is the on-disk profile schema version.
const SchemaVersion = 1

// Store is the file-backed profile store.
type Store struct {
	dir  string
	path string
}

// NewStore creates a store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir, path: filepath.Join(dir, "profiles.json")}
}

// NewDefaultStore creates the store under the XDG config directory.
func NewDefaultStore() (*Store, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	return NewStore(dir), nil
}

// Path returns the profiles.json path.
func (s *Store) Path() string { return s.path }

// Dir returns the profile directory.
func (s *Store) Dir() string { return s.dir }

type schema struct {
	Version  int              `json:"version"`
	Profiles []domain.Profile `json:"profiles"`
}

// Load reads profiles from disk. A missing file is an empty store. A corrupt,
// unreadable, or unsupported file returns a config error with no profiles.
func (s *Store) Load(ctx context.Context) ([]domain.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, ctxError(err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, unreadable()
	}
	var doc schema
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, unreadable()
	}
	if doc.Version != SchemaVersion {
		return nil, unreadable()
	}
	for i, p := range doc.Profiles {
		if err := p.Validate(); err != nil {
			return nil, unreadable()
		}
		doc.Profiles[i] = p.Normalize()
	}
	return doc.Profiles, nil
}

// Save writes profiles atomically with 0600 file and 0700 directory
// permissions. Only non-secret fields are persisted.
func (s *Store) Save(ctx context.Context, profiles []domain.Profile) error {
	if err := ctx.Err(); err != nil {
		return ctxError(err)
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return saveError(err)
	}
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return saveError(err)
	}
	doc := schema{Version: SchemaVersion, Profiles: make([]domain.Profile, 0, len(profiles))}
	for _, p := range profiles {
		p = p.Normalize()
		if err := p.Validate(); err != nil {
			return saveValidationError(err)
		}
		doc.Profiles = append(doc.Profiles, p)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return saveError(err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(s.dir, ".profiles-*.tmp")
	if err != nil {
		return saveError(err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return saveError(err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return saveError(err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return saveError(err)
	}
	if err := tmp.Close(); err != nil {
		return saveError(err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return saveError(err)
	}
	if err := os.Chmod(s.path, 0o600); err != nil {
		return saveError(err)
	}
	// Best effort: make the rename durable on filesystems that support it.
	if dir, err := os.Open(s.dir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func unreadable() error {
	return domain.NewError(domain.KindConfig, "load-profiles", "Profile storage is unreadable.")
}

func saveError(err error) error {
	e := domain.NewError(domain.KindConfig, "save-profiles", "Could not save profiles.")
	e.Err = err
	return e
}

func saveValidationError(err error) error {
	e := domain.NewError(domain.KindConfig, "save-profiles", "Could not save profiles: "+err.Error())
	e.Err = err
	return e
}

func ctxError(err error) error {
	if err == context.DeadlineExceeded {
		return domain.NewError(domain.KindTimeout, "profiles", "profile storage timed out")
	}
	return domain.NewError(domain.KindCancelled, "profiles", "cancelled")
}

// Ensure Store implements the profile port.
var _ ports.ProfileStore = (*Store)(nil)
