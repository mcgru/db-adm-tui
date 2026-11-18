package profile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func sampleProfile() domain.Profile {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	return domain.Profile{
		Label:     "prod-ro",
		Engine:    domain.EnginePostgreSQL,
		Host:      "db1.internal",
		Port:      5432,
		Database:  "orders",
		User:      "app_ro",
		Source:    &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestStoreRoundTripAndPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "config", "db-adm-tui")
	store := NewStore(dir)
	ctx := context.Background()
	if err := store.Save(ctx, []domain.Profile{sampleProfile()}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %o, want 700", dirInfo.Mode().Perm())
	}
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Label != "prod-ro" || loaded[0].Source.Kind != domain.SourcePgpass {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestStoreRoundTripsAllowNoTLS(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	p := sampleProfile()
	p.AllowNoTLS = true
	if err := store.Save(context.Background(), []domain.Profile{p}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || !loaded[0].AllowNoTLS {
		t.Fatalf("loaded = %+v", loaded)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"allow_no_tls": true`) {
		t.Fatalf("profiles.json missing allow_no_tls:\n%s", data)
	}
}

func TestStoreLoadsHandEditedAllowNoTLSAlias(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	data := `{"version":1,"profiles":[{"label":"legacy","engine":"postgresql",` +
		`"host":"db1","port":5432,"database":"orders","user":"app","notls":true}]}`
	if err := os.WriteFile(store.Path(), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || !loaded[0].AllowNoTLS {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestStoreNeverWritesSecretFields(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	ctx := context.Background()
	if err := store.Save(ctx, []domain.Profile{sampleProfile()}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"password", "passwd", "secret", "sup3r"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("profiles.json contains %q:\n%s", forbidden, data)
		}
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["version"] != float64(SchemaVersion) {
		t.Fatalf("version = %v", doc["version"])
	}
}

func TestStoreMissingFileIsEmpty(t *testing.T) {
	store := NewStore(t.TempDir())
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestStoreCorruptFileFallsBack(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := os.WriteFile(store.Path(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background())
	if err == nil || len(loaded) != 0 {
		t.Fatalf("load = %+v, %v", loaded, err)
	}
	if err.Error() != "Profile storage is unreadable." {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestStoreRejectsUnknownVersion(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := os.WriteFile(store.Path(), []byte(`{"version":2,"profiles":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background()); err == nil {
		t.Fatal("version 2 must be unreadable")
	}
}

func TestStoreNormalizesPromptSource(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	p := sampleProfile()
	p.Source = &domain.Source{Kind: domain.SourcePrompt}
	if err := store.Save(context.Background(), []domain.Profile{p}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded[0].Source != nil {
		t.Fatalf("prompt source persisted: %+v", loaded[0].Source)
	}
}

func TestConfigDirHonorsXDG(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(root, "db-adm-tui") {
		t.Fatalf("dir = %q", dir)
	}
	t.Setenv("XDG_CONFIG_HOME", "relative")
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err = ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(home, ".config", "db-adm-tui") {
		t.Fatalf("relative XDG dir = %q", dir)
	}
}

func TestDisplayPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := DisplayPath(filepath.Join(home, ".pgpass")); got != "~/.pgpass" {
		t.Fatalf("display = %q", got)
	}
}
