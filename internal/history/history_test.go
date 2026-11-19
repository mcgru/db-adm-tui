package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestDefaultDirUsesXDGStateHistory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/state")
	dir, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/tmp/state", "db-adm-tui", "history"); dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
}

func TestAppendLoadAndRedact(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "db-adm-tui")
	store := NewStore(dir)
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	entry := domain.HistoryEntry{
		Profile:   "prod",
		Engine:    domain.EnginePostgreSQL,
		Statement: "ALTER USER app PASSWORD 'hunter2'",
		Mode:      "r/o",
		Status:    "ok",
		Duration:  10 * time.Millisecond,
	}
	if err := store.Append(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), "prod")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Statement != "ALTER USER app PASSWORD '***'" {
		t.Fatalf("loaded = %+v", loaded)
	}
	if loaded[0].DurationMs != 10 {
		t.Fatalf("duration = %d", loaded[0].DurationMs)
	}
	info, err := os.Stat(filepath.Join(dir, "commands.jsonl"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("jsonl mode = %v, %v", info, err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %v, %v", info, err)
	}
	text, err := os.ReadFile(filepath.Join(dir, "commands.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "hunter2") {
		t.Fatalf("text history leaked password: %s", text)
	}
}

func TestLoadSkipsOtherProfilesAndMalformedLines(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Append(context.Background(), domain.HistoryEntry{Profile: "a", Statement: "SELECT 1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), domain.HistoryEntry{Profile: "b", Statement: "SELECT 2"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(dir, "commands.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{bad\n"); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	loaded, err := store.Load(context.Background(), "a")
	if err != nil || len(loaded) != 1 || loaded[0].Statement != "SELECT 1" {
		t.Fatalf("loaded = %+v, %v", loaded, err)
	}
}

func TestRedactPasswordPairs(t *testing.T) {
	statement := `db.users.createUser({user: "a", pwd: "secret"})`
	if got := Redact(statement, ""); strings.Contains(got, "secret") {
		t.Fatalf("redacted = %q", got)
	}
	if got := Redact("CREATE USER x IDENTIFIED BY 'secret'", ""); strings.Contains(got, "secret") {
		t.Fatalf("redacted = %q", got)
	}
}

func TestRotationKeepsFiveFilesPerFormatAndCurrentEntry(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	store.maxBytes = 120
	for i := 0; i < 20; i++ {
		if err := store.Append(context.Background(), domain.HistoryEntry{
			Profile:   "p",
			Statement: fmt.Sprintf("SELECT %d %s", i, strings.Repeat("x", 40)),
		}); err != nil {
			t.Fatal(err)
		}
		store.last = map[string]string{}
	}
	for _, base := range []string{"commands.jsonl", "commands.log"} {
		files, err := filepath.Glob(filepath.Join(dir, base+"*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 5 {
			t.Fatalf("%s files = %d, want 5: %v", base, len(files), files)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SELECT 19") {
		t.Fatalf("current entry was lost: %s", data)
	}
}
