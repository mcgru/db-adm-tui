package domain

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SourceKind identifies where a profile's credential comes from.
type SourceKind string

const (
	// SourcePgpass points at a psql-compatible password file.
	SourcePgpass SourceKind = "pgpass"
	// SourceMongoFile points at a selected ~/.mongo.* file.
	SourceMongoFile SourceKind = "mongo-file"
	// SourcePrompt means no file; the password is prompted once per connect.
	SourcePrompt SourceKind = "prompt"
)

// Valid reports whether k is a known source kind.
func (k SourceKind) Valid() bool {
	switch k {
	case SourcePgpass, SourceMongoFile, SourcePrompt:
		return true
	}
	return false
}

// Source is the optional non-secret pointer to a credential file. Path is a
// display path; "~" is expanded when the file is read. Selector carries the
// non-secret parts of a MongoDB URI (hosts and options) so a selected file can
// be replayed without storing its password.
type Source struct {
	Kind     SourceKind `json:"kind"`
	Path     string     `json:"path,omitempty"`
	Selector string     `json:"selector,omitempty"`
}

// Profile is one saved connection. It never contains a password.
type Profile struct {
	Label    string `json:"label"`
	Engine   Engine `json:"engine"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	// AllowNoTLS permits a PostgreSQL connection to fall back to a plaintext
	// attempt after the TLS attempt fails. TLS is always tried first.
	AllowNoTLS bool       `json:"allow_no_tls,omitempty"`
	Source     *Source    `json:"source,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
}

// UnmarshalJSON accepts the canonical allow_no_tls key plus the allowNoTLS and
// notls aliases used in hand-edited profile files.
func (p *Profile) UnmarshalJSON(data []byte) error {
	type profileAlias Profile
	var decoded profileAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = Profile(decoded)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"allow_no_tls", "allowNoTLS", "notls"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var enabled bool
		if err := json.Unmarshal(raw, &enabled); err != nil {
			return err
		}
		if enabled {
			p.AllowNoTLS = true
		}
	}
	return nil
}

// Key is the deterministic identity of a profile. MongoDB profiles are keyed
// by their source file so a changed URI refreshes the same profile; other
// profiles are keyed by their non-secret connection tuple.
func (p Profile) Key() string {
	if p.Source != nil && p.Source.Kind == SourceMongoFile && p.Source.Path != "" {
		return "mongo-file:" + p.Source.Path
	}
	return strings.Join([]string{
		string(p.Engine),
		strings.ToLower(p.Host),
		strconv.Itoa(p.Port),
		p.Database,
		p.User,
	}, "|")
}

// Addr returns host:port in dial-ready form.
func (p Profile) Addr() string {
	return net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
}

// RedactedURI returns a connection string with no password. It is suitable for
// display and logging.
func (p Profile) RedactedURI() string {
	u := url.URL{
		Scheme: p.Engine.Scheme(),
		Host:   p.Addr(),
	}
	if p.Database != "" {
		u.Path = "/" + p.Database
	}
	if p.User != "" {
		u.User = url.User(p.User)
	}
	return u.String()
}

// SourceLabel is the text shown in the picker's source column.
func (p Profile) SourceLabel() string {
	if p.Source == nil || p.Source.Kind == SourcePrompt {
		return "prompt"
	}
	if p.Source.Path != "" {
		return p.Source.Path
	}
	return string(p.Source.Kind)
}

// DefaultLabel builds the label used when the user leaves the label empty.
func DefaultLabel(database, host string) string {
	database = strings.TrimSpace(database)
	host = strings.TrimSpace(host)
	switch {
	case database != "" && host != "":
		return database + "@" + host
	case host != "":
		return host
	case database != "":
		return database
	default:
		return "profile"
	}
}

// Validate reports whether a profile can be stored and used. Error strings are
// safe to display: they never contain a secret.
func (p Profile) Validate() error {
	if strings.TrimSpace(p.Label) == "" {
		return fmt.Errorf("profile label is empty")
	}
	if !p.Engine.Valid() {
		return fmt.Errorf("profile engine %q is not supported", p.Engine)
	}
	if strings.TrimSpace(p.Host) == "" {
		return fmt.Errorf("profile host is empty")
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("profile port %d is not valid", p.Port)
	}
	if p.Source != nil {
		if !p.Source.Kind.Valid() {
			return fmt.Errorf("profile source kind %q is not supported", p.Source.Kind)
		}
		if p.Source.Kind != SourcePrompt && strings.TrimSpace(p.Source.Path) == "" {
			return fmt.Errorf("profile source path is empty")
		}
	}
	return nil
}

// SanitizeMongoSelector removes password-like query keys from a non-secret
// MongoDB selector. It is applied before a selector is stored or replayed.
func SanitizeMongoSelector(selector string) string {
	idx := strings.Index(selector, "?")
	if idx < 0 {
		return selector
	}
	base, query := selector[:idx], selector[idx+1:]
	parts := strings.Split(query, "&")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		key, _, _ := strings.Cut(part, "=")
		switch strings.ToLower(key) {
		case "password", "passwd", "pwd":
			continue
		}
		if part != "" {
			kept = append(kept, part)
		}
	}
	if len(kept) == 0 {
		return base
	}
	return base + "?" + strings.Join(kept, "&")
}

// Normalize returns a copy with surrounding whitespace removed and a prompt
// source represented by no source pointer.
func (p Profile) Normalize() Profile {
	p.Label = strings.TrimSpace(p.Label)
	p.Host = strings.TrimSpace(p.Host)
	p.Database = strings.TrimSpace(p.Database)
	p.User = strings.TrimSpace(p.User)
	if p.Label == "" {
		p.Label = DefaultLabel(p.Database, p.Host)
	}
	if p.Source != nil {
		p.Source.Path = strings.TrimSpace(p.Source.Path)
		p.Source.Selector = strings.TrimSpace(p.Source.Selector)
		if p.Source.Kind == SourceMongoFile {
			p.Source.Selector = SanitizeMongoSelector(p.Source.Selector)
		}
		if p.Source.Kind == SourcePrompt || (p.Source.Kind == "" && p.Source.Path == "") {
			p.Source = nil
		}
	}
	return p
}
