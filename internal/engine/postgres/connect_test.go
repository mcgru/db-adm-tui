package postgres

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildDSN(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	dsn := buildDSN(p, "p@ss word/with:chars")
	for _, want := range []string{"postgres://", "app_ro", "db1:5432", "/orders", "sslmode=prefer", "passfile="} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn missing %q: %s", want, dsn)
		}
	}
	if strings.Contains(dsn, "p@ss word/with:chars") {
		t.Fatal("password was not URL-encoded")
	}
}

func TestBuildDSNPreservesPasswordWithEmptyUser(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders",
	}
	dsn := buildDSN(p, "secret")
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if u.User == nil {
		t.Fatal("password was dropped for an empty user")
	}
	if password, ok := u.User.Password(); !ok || password != "secret" {
		t.Fatalf("password = %q, %v", password, ok)
	}
}

func TestBuildDSNParsesWithoutReadingPassfile(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	cfg, err := pgx.ParseConfig(buildDSN(p, "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != "secret" {
		t.Fatalf("password = %q", cfg.Password)
	}
	if cfg.User != "app_ro" || cfg.Database != "orders" {
		t.Fatalf("config = user %q db %q", cfg.User, cfg.Database)
	}
}

func TestBuildDSNSuppressesPassfileFallback(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	if err := os.WriteFile(path, []byte("db1:5432:orders:app_ro:fromfile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PGPASSFILE", path)
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	cfg, err := pgx.ParseConfig(buildDSN(p, ""))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != "" {
		t.Fatalf("pgx read the passfile behind the resolver: %q", cfg.Password)
	}
}

func TestSafeErrorRedactsSecretAndMapsKinds(t *testing.T) {
	err := safeError("connect", "hunter2", errors.New("dial failed for hunter2"))
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaked password: %v", err)
	}
	if cause := errors.Unwrap(err); cause == nil || strings.Contains(cause.Error(), "hunter2") {
		t.Fatalf("unwrapped cause leaked or was missing: %v", cause)
	}
	if err.Kind != domain.KindEngine {
		t.Fatalf("kind = %v", err.Kind)
	}
	timeout := safeError("connect", "", context.DeadlineExceeded)
	if timeout.Kind != domain.KindTimeout {
		t.Fatalf("timeout kind = %v", timeout.Kind)
	}
	cancelled := safeError("connect", "", context.Canceled)
	if cancelled.Kind != domain.KindCancelled {
		t.Fatalf("cancel kind = %v", cancelled.Kind)
	}
}
