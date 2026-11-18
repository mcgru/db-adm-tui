package profile

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// ErrPromptRequired means the profile has no file source and needs a one-time
// password from the UI.
var ErrPromptRequired = errors.New("password required")

// SourceError names a source file and a safe reason. It never contains the
// password or the raw file content.
type SourceError struct {
	Kind   domain.ErrorKind
	Source string
	Reason string
}

// Error implements error.
func (e *SourceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Reason
}

// Resolver reads credential sources. Every connection re-reads the file; no
// value is cached.
type Resolver struct {
	home       string
	pgpassPath string
}

// NewResolver builds a resolver using the process environment.
func NewResolver() (*Resolver, error) {
	home, err := HomeDir()
	if err != nil {
		return nil, err
	}
	pgpass, err := PgpassPath()
	if err != nil {
		return nil, err
	}
	return &Resolver{home: home, pgpassPath: pgpass}, nil
}

// NewResolverAt builds a resolver for an explicit home and password file. It is
// used by tests and by the composition root when the environment is known.
func NewResolverAt(home, pgpassPath string) *Resolver {
	return &Resolver{home: home, pgpassPath: pgpassPath}
}

// Home returns the resolver's home directory.
func (r *Resolver) Home() string { return r.home }

// PgpassPath returns the absolute password-file path in effect.
func (r *Resolver) PgpassPath() string { return r.pgpassPath }

// MongoFiles returns the MongoDB connection files under the resolver's home.
func (r *Resolver) MongoFiles() []string { return MongoFilesIn(r.home) }

// DisplayPath shortens a path under the resolver's home directory to "~".
func (r *Resolver) DisplayPath(path string) string { return r.display(path) }

// Resolve returns the password for a profile from its source. It returns
// ErrPromptRequired when the profile has no source file.
func (r *Resolver) Resolve(p domain.Profile) (string, error) {
	secret, _, err := r.ResolveSource(p)
	return secret, err
}

// ResolveSource returns the password and the fresh Mongo selector from a
// profile's source. The selector lets a saved profile pick up host or option
// changes in a ~/.mongo.* file without caching them in the profile file.
func (r *Resolver) ResolveSource(p domain.Profile) (string, string, error) {
	if p.Source == nil || p.Source.Kind == domain.SourcePrompt {
		return "", "", ErrPromptRequired
	}
	switch p.Source.Kind {
	case domain.SourcePgpass:
		path, err := r.expand(p.Source.Path)
		if err != nil {
			return "", "", &SourceError{Kind: domain.KindConfig, Source: p.SourceLabel(), Reason: "source path is not valid"}
		}
		pf := ParsePgpass(path)
		source := r.display(path)
		switch {
		case pf.Missing:
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "file not found"}
		case pf.Ignored:
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "file is ignored because " + pf.IgnoreReason}
		case pf.ReadError != "":
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: pf.ReadError}
		}
		entry, ok := MatchPgpass(pf.Entries, p.Host, p.Port, p.Database, p.User)
		if !ok {
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "no matching line"}
		}
		return entry.Password, "", nil
	case domain.SourceMongoFile:
		path, err := r.expand(p.Source.Path)
		if err != nil {
			return "", "", &SourceError{Kind: domain.KindConfig, Source: p.SourceLabel(), Reason: "source path is not valid"}
		}
		mf := ParseMongoFile(path)
		source := r.display(path)
		switch {
		case mf.Missing:
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "file not found"}
		case mf.Err != "":
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: mf.Err}
		}
		if mf.Entry == nil {
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "no mongodb:// URI found"}
		}
		if !MatchMongo(*mf.Entry, p.Host, p.Port, p.Database, p.User) {
			return "", "", &SourceError{Kind: domain.KindConfig, Source: source, Reason: "source no longer matches this profile"}
		}
		return mf.Entry.Password, mf.Entry.Selector, nil
	default:
		return "", "", &SourceError{Kind: domain.KindConfig, Source: p.SourceLabel(), Reason: "source kind is not supported"}
	}
}

// MatchPostgres finds the current ~/.pgpass source matching a connection
// tuple. It returns the display path of the file.
func (r *Resolver) MatchPostgres(host string, port int, database, user string) (string, PgpassEntry, bool) {
	pf := ParsePgpass(r.pgpassPath)
	if pf.Missing || pf.Ignored || pf.ReadError != "" {
		return "", PgpassEntry{}, false
	}
	entry, ok := MatchPgpass(pf.Entries, host, port, database, user)
	if !ok {
		return "", PgpassEntry{}, false
	}
	return r.display(r.pgpassPath), entry, true
}

// MatchMongo finds the first ~/.mongo.* source matching a connection tuple.
// Files are scanned in sorted order.
func (r *Resolver) MatchMongo(host string, port int, database, user string) (string, MongoEntry, bool) {
	files := MongoFilesIn(r.home)
	for _, path := range files {
		mf := ParseMongoFile(path)
		if mf.Entry == nil || mf.Err != "" {
			continue
		}
		if MatchMongo(*mf.Entry, host, port, database, user) {
			return r.display(path), *mf.Entry, true
		}
	}
	return "", MongoEntry{}, false
}

// expand resolves a stored display path against the resolver's home directory.
func (r *Resolver) expand(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		return r.home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(r.home, path[2:]), nil
	}
	return path, nil
}

func (r *Resolver) display(path string) string {
	if r.home == "" || path == "" {
		return path
	}
	if path == r.home {
		return "~"
	}
	prefix := r.home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return "~" + path[len(r.home):]
	}
	return path
}

// SourceState describes a profile source's current health for the picker.
// Empty means the source is usable or the profile is prompt-based.
func (r *Resolver) SourceState(p domain.Profile) string {
	return sourceState(r.home, p)
}

// SourceState describes a profile source's current health using the process
// home directory.
func SourceState(p domain.Profile) string {
	home, _ := HomeDir()
	return sourceState(home, p)
}

func sourceState(home string, p domain.Profile) string {
	if p.Source == nil || p.Source.Kind == domain.SourcePrompt {
		return ""
	}
	path := p.Source.Path
	switch {
	case path == "~":
		path = home
	case strings.HasPrefix(path, "~/"):
		path = filepath.Join(home, path[2:])
	}
	switch p.Source.Kind {
	case domain.SourcePgpass:
		pf := ParsePgpass(path)
		switch {
		case pf.Missing:
			return "missing"
		case pf.Ignored:
			return "ignored"
		case pf.ReadError != "":
			return "unreadable"
		}
		if _, ok := MatchPgpass(pf.Entries, p.Host, p.Port, p.Database, p.User); !ok {
			return "stale"
		}
	case domain.SourceMongoFile:
		mf := ParseMongoFile(path)
		switch {
		case mf.Missing:
			return "missing"
		case mf.Err != "":
			return "error"
		}
		if mf.Entry == nil || !MatchMongo(*mf.Entry, p.Host, p.Port, p.Database, p.User) {
			return "stale"
		}
	default:
		return "unreadable"
	}
	return ""
}
