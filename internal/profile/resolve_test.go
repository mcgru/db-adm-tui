package profile

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestResolvePromptAndNoSource(t *testing.T) {
	resolver := NewResolverAt(t.TempDir(), filepath.Join(t.TempDir(), ".pgpass"))
	p := domain.Profile{Label: "x", Engine: domain.EnginePostgreSQL, Host: "h", Port: 5432}
	if _, err := resolver.Resolve(p); !errors.Is(err, ErrPromptRequired) {
		t.Fatalf("no source err = %v", err)
	}
	p.Source = &domain.Source{Kind: domain.SourcePrompt}
	if _, err := resolver.Resolve(p); !errors.Is(err, ErrPromptRequired) {
		t.Fatalf("prompt err = %v", err)
	}
}

func TestResolvePgpassRereadsFile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:first\n", 0o600)
	resolver := NewResolverAt(home, path)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	secret, err := resolver.Resolve(p)
	if err != nil || secret != "first" {
		t.Fatalf("resolve = %q, %v", secret, err)
	}
	writeFile(t, path, "db1:5432:orders:app_ro:second\n", 0o600)
	secret, err = resolver.Resolve(p)
	if err != nil || secret != "second" {
		t.Fatalf("re-read = %q, %v", secret, err)
	}
}

func TestResolvePgpassFailuresNameSource(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "other:5432:orders:app_ro:secret\n", 0o600)
	resolver := NewResolverAt(home, path)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	_, err := resolver.Resolve(p)
	var sourceErr *SourceError
	if !errors.As(err, &sourceErr) || sourceErr.Source != "~/.pgpass" || !strings.Contains(sourceErr.Reason, "no matching line") {
		t.Fatalf("err = %+v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("resolution error leaked the password")
	}
}

func TestResolvePgpassIgnoredPermissions(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:secret\n", 0o644)
	resolver := NewResolverAt(home, path)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	_, err := resolver.Resolve(p)
	if err == nil || !strings.Contains(err.Error(), "ignored") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("ignored-file error leaked the password")
	}
}

func TestResolveMongoChangedSelector(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".mongo.analyst")
	writeFile(t, path, "mongodb://analyst:secret@mongo1:27017/analytics\n", 0o600)
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	p := domain.Profile{
		Label: "analytics", Engine: domain.EngineMongoDB, Host: "mongo2", Port: 27017,
		Database: "analytics", User: "analyst",
		Source: &domain.Source{Kind: domain.SourceMongoFile, Path: "~/.mongo.analyst"},
	}
	_, err := resolver.Resolve(p)
	if err == nil || !strings.Contains(err.Error(), "no longer matches") {
		t.Fatalf("err = %v", err)
	}
	p.Host = "mongo1"
	secret, err := resolver.Resolve(p)
	if err != nil || secret != "secret" {
		t.Fatalf("resolve = %q, %v", secret, err)
	}
}

func TestSourceState(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:secret\n", 0o644)
	resolver := NewResolverAt(home, path)
	p := domain.Profile{
		Label: "x", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	if got := resolver.SourceState(p); got != "ignored" {
		t.Fatalf("state = %q", got)
	}
	p.Source.Path = "~/.missing"
	if got := resolver.SourceState(p); got != "missing" {
		t.Fatalf("missing state = %q", got)
	}
	writeFile(t, path, "db1:5432:orders:app_ro:secret\n", 0o600)
	p.Source.Path = "~/.pgpass"
	p.Host = "other"
	if got := resolver.SourceState(p); got != "stale" {
		t.Fatalf("stale state = %q", got)
	}
}
