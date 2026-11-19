// Package history persists executed statements as JSONL and plain text under
// the XDG state directory. Writes are best-effort and never block a query.
package history

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

const (
	defaultMaxBytes = 10 << 20
	// defaultKeep is the total number of files per format: the current file
	// plus keep-1 rotated backups.
	defaultKeep = 5
)

// Store is a file-backed ports.HistoryStore.
type Store struct {
	mu       sync.Mutex
	dir      string
	maxBytes int64
	keep     int
	now      func() time.Time
	last     map[string]string
}

// NewStore returns a history store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{
		dir:      dir,
		maxBytes: defaultMaxBytes,
		keep:     defaultKeep,
		now:      time.Now,
		last:     map[string]string{},
	}
}

// NewDefaultStore uses $XDG_STATE_HOME/db-adm-tui, defaulting to
// ~/.local/state/db-adm-tui.
func NewDefaultStore() (*Store, error) {
	dir, err := DefaultDir()
	if err != nil {
		return nil, err
	}
	return NewStore(dir), nil
}

// DefaultDir returns the XDG history directory for db-adm-tui.
func DefaultDir() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "db-adm-tui", "history"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "db-adm-tui", "history"), nil
}

func (s *Store) jsonlPath() string { return filepath.Join(s.dir, "commands.jsonl") }
func (s *Store) textPath() string  { return filepath.Join(s.dir, "commands.log") }

// Load returns the profile's statements in execution order, collapsing
// consecutive duplicates.
func (s *Store) Load(ctx context.Context, profile string) ([]domain.HistoryEntry, error) {
	if s == nil || s.dir == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.Open(s.jsonlPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var entries []domain.HistoryEntry
	lastStatement := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		var entry domain.HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		s.last[entry.Profile] = entry.Statement
		if entry.Profile != profile {
			continue
		}
		if entry.Statement == lastStatement {
			continue
		}
		lastStatement = entry.Statement
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// Append writes one entry to both formats. A repeated consecutive statement
// for the same profile is skipped.
func (s *Store) Append(ctx context.Context, entry domain.HistoryEntry) error {
	if s == nil || s.dir == "" {
		return nil
	}
	if entry.Time.IsZero() {
		entry.Time = s.now().UTC()
	}
	if entry.DurationMs == 0 && entry.Duration > 0 {
		entry.DurationMs = entry.Duration.Milliseconds()
	}
	entry.Statement = Redact(entry.Statement, "")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		s.last = map[string]string{}
	}
	if s.last[entry.Profile] == entry.Statement {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return err
	}
	jsonLine, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	jsonLine = append(jsonLine, '\n')
	textLine := []byte(fmt.Sprintf("%s %s %s %s %dms %s %s\n",
		entry.Time.UTC().Format(time.RFC3339),
		entry.Profile,
		entry.Engine,
		entry.Mode,
		entry.DurationMs,
		entry.Status,
		strings.ReplaceAll(entry.Statement, "\n", " "),
	))
	if err := s.write(s.jsonlPath(), jsonLine); err != nil {
		return err
	}
	if err := s.write(s.textPath(), textLine); err != nil {
		return err
	}
	s.last[entry.Profile] = entry.Statement
	return nil
}

func (s *Store) write(path string, data []byte) error {
	if s.maxBytes > 0 {
		if info, err := os.Stat(path); err == nil && info.Size()+int64(len(data)) > s.maxBytes {
			if err := s.rotate(path); err != nil {
				return err
			}
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (s *Store) rotate(path string) error {
	if s.keep <= 1 {
		return os.Remove(path)
	}
	_ = os.Remove(fmt.Sprintf("%s.%d", path, s.keep-1))
	for i := s.keep - 2; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", path, i)
		to := fmt.Sprintf("%s.%d", path, i+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(path, path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Redact removes connection secrets and password literals before a statement
// is stored. The known connection secret is removed first when provided.
func Redact(statement, secret string) string {
	return domain.RedactSecrets(statement, secret)
}
