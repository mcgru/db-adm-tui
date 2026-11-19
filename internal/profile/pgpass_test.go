package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestPgpassParsesCommentsEscapesAndWildcards(t *testing.T) {
	data := strings.Join([]string{
		"# comment line",
		"",
		"db1:5432:orders:app_ro:first",
		"db2:*:*:*:port\\:escaped",
		"db3:5432:orders:app_ro:second",
		"*:*:*:*:wild",
	}, "\n")
	entries, warnings := ParsePgpassLines(data)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v", warnings)
	}
	if len(entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(entries))
	}
	if entries[1].Password != "port:escaped" {
		t.Fatalf("escaped password = %q", entries[1].Password)
	}
	entry, ok := MatchPgpass(entries, "db3", 5432, "orders", "app_ro")
	if !ok || entry.Password != "second" {
		t.Fatalf("match = %+v, %v (first match must win)", entry, ok)
	}
	entry, ok = MatchPgpass(entries, "anything", 1234, "whatever", "whoever")
	if !ok || entry.Password != "wild" {
		t.Fatalf("wildcard match = %+v, %v", entry, ok)
	}
}

func TestPgpassMalformedLinesWarnWithoutPassword(t *testing.T) {
	data := strings.Join([]string{
		"too:few:fields",
		"db1:notaport:orders:app_ro:secret",
		"db1:5432:orders:app_ro:trailing\\",
		"db1:5432:orders:app_ro:good",
	}, "\n")
	entries, warnings := ParsePgpassLines(data)
	if len(entries) != 1 || entries[0].Password != "good" {
		t.Fatalf("entries = %+v", entries)
	}
	if len(warnings) != 3 {
		t.Fatalf("warnings = %+v", warnings)
	}
	for _, warning := range warnings {
		if strings.Contains(warning.Reason, "secret") {
			t.Fatalf("warning leaked a password: %+v", warning)
		}
	}
}

func TestPgpassRequiresPrivatePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:secret\n", 0o644)
	pf := ParsePgpass(path)
	if !pf.Ignored || len(pf.Entries) != 0 {
		t.Fatalf("insecure file not ignored: %+v", pf)
	}
	if strings.Contains(pf.IgnoreReason, "secret") {
		t.Fatal("ignore reason leaked secret")
	}
}

func TestPgpassAcceptsOwnerOnlyPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:secret\n", 0o600)
	pf := ParsePgpass(path)
	if pf.Ignored || len(pf.Entries) != 1 {
		t.Fatalf("0600 file not read: %+v", pf)
	}
}

func TestPgpassMissingFileIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pf := ParsePgpass(filepath.Join(home, ".pgpass"))
	if !pf.Missing || pf.ReadError != "" || pf.Ignored {
		t.Fatalf("missing file = %+v", pf)
	}
}

func TestPgpassPathHonorsPGPASSFILE(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PGPASSFILE", "~/custom.pass")
	path, err := PgpassPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, "custom.pass") {
		t.Fatalf("path = %q", path)
	}
	t.Setenv("PGPASSFILE", "")
	path, err = PgpassPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".pgpass") {
		t.Fatalf("default path = %q", path)
	}
}

func TestParsePgpassLinesEmptyPassword(t *testing.T) {
	entries, warnings := ParsePgpassLines("db1:5432:orders:app_ro:\n")
	if len(warnings) != 0 || len(entries) != 1 || entries[0].Password != "" {
		t.Fatalf("entries=%+v warnings=%+v", entries, warnings)
	}
}

func FuzzParsePgpassLines(f *testing.F) {
	for _, seed := range []string{
		"db1:5432:orders:app_ro:secret\n",
		"# comment\n*\t*\t*\t*\t*\n",
		"bad:line\n",
		"host\\:name:5432:db:user:pass\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		entries, warnings := ParsePgpassLines(data)
		for _, warning := range warnings {
			if strings.Contains(warning.Reason, "secret") {
				t.Fatalf("warning leaked a password: %q", warning.Reason)
			}
		}
		_ = entries
	})
}
