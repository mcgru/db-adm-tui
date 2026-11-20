package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestUpdateCredentialMovesPgpassEntry(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	if err := os.WriteFile(path, []byte(
		"unrelated:5432:db:user:other\nold:5432:db:user:secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolverAt(home, path)
	old := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "old", Port: 5432,
		Database: "db", User: "user",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	next := old
	next.Host = "new"
	next.Port = 5433
	next.Database = "db2"
	next.User = "user2"
	source, err := resolver.UpdateCredential(old, next)
	if err != nil {
		t.Fatal(err)
	}
	if source == nil || source.Kind != domain.SourcePgpass || source.Path != "~/.pgpass" {
		t.Fatalf("source = %+v", source)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "#old:5432:db:user:secret") {
		t.Fatalf("old entry was not commented:\n%s", text)
	}
	if !strings.Contains(text, "new:5433:db2:user2:secret") {
		t.Fatalf("new entry missing:\n%s", text)
	}
	if !strings.Contains(text, "unrelated:5432:db:user:other") {
		t.Fatalf("unrelated entry changed:\n%s", text)
	}
}

func TestUpdateCredentialLabelOnlyIsNoop(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	content := "old:5432:db:user:secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolverAt(home, path)
	old := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "old", Port: 5432,
		Database: "db", User: "user",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	next := old
	next.Label = "renamed"
	if _, err := resolver.UpdateCredential(old, next); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("label-only edit rewrote the file:\n%s", data)
	}
}

func TestUpdateCredentialMovesMongoFile(t *testing.T) {
	home := t.TempDir()
	oldPath := filepath.Join(home, ".mongo.old-in-olddb")
	oldURI := "mongodb://old:secret@oldhost:27017/olddb?authSource=admin&replicaSet=rs0"
	if err := os.WriteFile(oldPath, []byte(oldURI+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	old := domain.Profile{
		Engine: domain.EngineMongoDB, Host: "oldhost", Port: 27017,
		Database: "olddb", User: "old",
		Source: &domain.Source{Kind: domain.SourceMongoFile, Path: "~/.mongo.old-in-olddb"},
	}
	next := old
	next.Host = "newhost"
	next.Port = 27018
	next.Database = "newdb"
	next.User = "new"
	source, err := resolver.UpdateCredential(old, next)
	if err != nil {
		t.Fatal(err)
	}
	if source == nil || source.Kind != domain.SourceMongoFile ||
		!strings.Contains(source.Path, ".mongo.new-in-newdb") {
		t.Fatalf("source = %+v", source)
	}
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("old file was deleted: %v", err)
	}
	if !strings.Contains(string(oldData), "#"+oldURI) {
		t.Fatalf("old URI was not commented:\n%s", oldData)
	}
	newPath := filepath.Join(home, ".mongo.new-in-newdb")
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(newData)
	for _, want := range []string{
		"mongodb://new:secret@newhost:27018/newdb",
		"authSource=newdb",
		"replicaSet=rs0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("new URI missing %q:\n%s", want, text)
		}
	}
}
