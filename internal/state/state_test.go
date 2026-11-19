package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestSaveLoadLayouts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "db-adm-tui")
	store := NewStore(filepath.Join(dir, "layouts.json"))
	ctx := context.Background()
	if err := store.Save(ctx, "prod", domain.LayoutSettings{TreeWidth: 30, QueryHeight: 2}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, "billing", domain.LayoutSettings{TreeWidth: 18, QueryHeight: 3}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(ctx, "prod")
	if err != nil || got.TreeWidth != 30 || got.QueryHeight != 2 {
		t.Fatalf("prod = %+v, %v", got, err)
	}
	got, err = store.Load(ctx, "billing")
	if err != nil || got.TreeWidth != 18 || got.QueryHeight != 3 {
		t.Fatalf("billing = %+v, %v", got, err)
	}
	info, err := os.Stat(store.Path())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v, %v", info, err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %v, %v", info, err)
	}
}

func TestLoadMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "layouts.json"))
	got, err := store.Load(context.Background(), "missing")
	if err != nil || got != (domain.LayoutSettings{}) {
		t.Fatalf("missing = %+v, %v", got, err)
	}
	if err := os.WriteFile(store.Path(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), "missing"); err == nil {
		t.Fatal("corrupt file loaded")
	}
}

func TestDefaultPathUsesXDGState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/state")
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/tmp/state", "db-adm-tui", "layouts.json"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}
