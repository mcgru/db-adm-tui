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
	dsn := buildDSN(p, "p@ss word/with:chars", sslRequire)
	for _, want := range []string{"postgres://", "app_ro", "db1:5432", "/orders", "sslmode=require", "passfile="} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn missing %q: %s", want, dsn)
		}
	}
	if strings.Contains(dsn, "p@ss word/with:chars") {
		t.Fatal("password was not URL-encoded")
	}
}

func TestBuildDSNDisableTLS(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	u, err := url.Parse(buildDSN(p, "", sslDisable))
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("sslmode"); got != sslDisable {
		t.Fatalf("sslmode = %q, want %q", got, sslDisable)
	}
}

func TestConnectionModesFollowAllowNoTLS(t *testing.T) {
	modes := connectionModes(domain.Profile{})
	if len(modes) != 1 || modes[0] != sslRequire {
		t.Fatalf("default modes = %v", modes)
	}
	modes = connectionModes(domain.Profile{AllowNoTLS: true})
	if len(modes) != 2 || modes[0] != sslRequire || modes[1] != sslDisable {
		t.Fatalf("allow-no-tls modes = %v", modes)
	}
}

func TestRetryWithoutTLSOnlyForEngineFailures(t *testing.T) {
	if !retryWithoutTLS(safeError("connect", "", errors.New("tls handshake failed"))) {
		t.Fatal("connection-level errors must retry without TLS")
	}
	for _, kind := range []domain.ErrorKind{
		domain.KindAuth, domain.KindPermission, domain.KindConfig,
		domain.KindTimeout, domain.KindCancelled, domain.KindGuard,
	} {
		if retryWithoutTLS(domain.NewError(kind, "connect", "no")) {
			t.Fatalf("%s error must not retry without TLS", kind)
		}
	}
}

func TestBuildDSNPreservesPasswordWithEmptyUser(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders",
	}
	dsn := buildDSN(p, "secret", sslRequire)
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
	cfg, err := pgx.ParseConfig(buildDSN(p, "secret", sslRequire))
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
	cfg, err := pgx.ParseConfig(buildDSN(p, "", sslRequire))
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
