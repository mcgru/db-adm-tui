package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestSavePostgresCredentialCreatesPrivateFile(t *testing.T) {
	home := t.TempDir()
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	source, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro", Password: "s:e\\cret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != domain.SourcePgpass || source.Path != "~/.pgpass" {
		t.Fatalf("source = %+v", source)
	}
	info, err := os.Stat(filepath.Join(home, ".pgpass"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	pf := ParsePgpass(filepath.Join(home, ".pgpass"))
	if len(pf.Entries) != 1 || pf.Entries[0].Password != "s:e\\cret" {
		t.Fatalf("entries = %+v", pf.Entries)
	}
}

func TestSaveCredentialRotatesMatchingLine(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, strings.Join([]string{
		"# keep this comment",
		"db1:5432:orders:app_ro:old",
		"other:5432:orders:other:keep",
		"",
	}, "\n"), 0o644)
	resolver := NewResolverAt(home, path)
	if _, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro", Password: "new",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	want := []string{
		"# keep this comment",
		"#db1:5432:orders:app_ro:old",
		"db1:5432:orders:app_ro:new",
		"other:5432:orders:other:keep",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("file:\n%s", data)
	}
	pf := ParsePgpass(path)
	if len(pf.Entries) != 2 || pf.Entries[0].Password != "new" || pf.Entries[1].Password != "keep" {
		t.Fatalf("entries = %+v", pf.Entries)
	}
}

func TestSaveCredentialAppendsWhenNoMatch(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pgpass")
	writeFile(t, path, "db1:5432:orders:app_ro:old\n", 0o600)
	resolver := NewResolverAt(home, path)
	if _, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EnginePostgreSQL, Host: "db2", Port: 5432,
		Database: "billing", User: "app_ro", Password: "new",
	}); err != nil {
		t.Fatal(err)
	}
	pf := ParsePgpass(path)
	if len(pf.Entries) != 2 || pf.Entries[0].Password != "old" || pf.Entries[1].Password != "new" {
		t.Fatalf("entries = %+v", pf.Entries)
	}
}

func TestSaveCredentialUsesExplicitSourcePath(t *testing.T) {
	home := t.TempDir()
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	source, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro", Password: "secret",
		SourcePath: "~/custom.pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.Path != "~/custom.pass" {
		t.Fatalf("source = %+v", source)
	}
	pf := ParsePgpass(filepath.Join(home, "custom.pass"))
	if len(pf.Entries) != 1 || pf.Entries[0].Password != "secret" {
		t.Fatalf("entries = %+v", pf.Entries)
	}
}

func TestSaveMySQLCredentialUsesMyPass(t *testing.T) {
	home := t.TempDir()
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	source, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EngineMySQL, Host: "db1", Port: 3306,
		Database: "billing", User: "app_ro", Password: "mysqlsecret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != domain.SourceMySQLPass || source.Path != "~/.mypass" {
		t.Fatalf("source = %+v", source)
	}
	mf := ParseMyPass(filepath.Join(home, ".mypass"))
	if len(mf.Entries) != 1 || mf.Entries[0].Password != "mysqlsecret" {
		t.Fatalf("entries = %+v", mf.Entries)
	}
}

func TestMongoCredentialURIPreservesHostList(t *testing.T) {
	uri := mongoCredentialURI(CredentialTarget{
		Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
		MongoHosts: "mongo1:27017,mongo2:27018",
		Database:   "appdb", User: "app_user", Password: "secret",
		MongoOptions: "authSource=admin&replicaSet=rs0",
	})
	for _, want := range []string{
		"mongodb://app_user:secret@mongo1:27017,mongo2:27018/appdb",
		"authSource=admin",
		"replicaSet=rs0",
	} {
		if !strings.Contains(uri, want) {
			t.Fatalf("URI missing %q: %s", want, uri)
		}
	}
}

func TestSaveMongoCredentialRotates(t *testing.T) {
	home := t.TempDir()
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	target := CredentialTarget{
		Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
		Database: "analytics", User: "analyst", Password: "first",
		MongoOptions: "authSource=analytics&replicaSet=rs0",
	}
	source, err := resolver.SaveCredential(target)
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != domain.SourceMongoFile || source.Path != "~/.mongo.analyst-in-analytics" {
		t.Fatalf("source = %+v", source)
	}
	path := filepath.Join(home, ".mongo.analyst-in-analytics")
	mf := ParseMongoFile(path)
	if mf.Entry == nil || mf.Entry.Password != "first" {
		t.Fatalf("entry = %+v, err=%q", mf.Entry, mf.Err)
	}
	target.Password = "second"
	if _, err := resolver.SaveCredential(target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "#mongodb://analyst:first@") {
		t.Fatalf("old URI not commented:\n%s", data)
	}
	mf = ParseMongoFile(path)
	if mf.Entry == nil || mf.Entry.Password != "second" {
		t.Fatalf("rotated entry = %+v, err=%q", mf.Entry, mf.Err)
	}
	if !strings.Contains(mf.Entry.Selector, "replicaSet=rs0") {
		t.Fatalf("selector = %q", mf.Entry.Selector)
	}
}

func TestSaveCredentialRejectsLineBreaks(t *testing.T) {
	home := t.TempDir()
	resolver := NewResolverAt(home, filepath.Join(home, ".pgpass"))
	_, err := resolver.SaveCredential(CredentialTarget{
		Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro", Password: "bad\npassword",
	})
	if err == nil {
		t.Fatal("line-break password was accepted")
	}
	if _, statErr := os.Stat(filepath.Join(home, ".pgpass")); !os.IsNotExist(statErr) {
		t.Fatal("credential file was written for a rejected password")
	}
}
