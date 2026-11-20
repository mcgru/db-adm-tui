package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMongoBytesLastURIWins(t *testing.T) {
	data := strings.Join([]string{
		"# connection file",
		"mongosh \"mongodb://first:one@mongo1:27017/analytics\"",
		"",
		"mongo 'mongodb://analyst:two@mongo2:27018/archive?authSource=admin'",
	}, "\n")
	mf := ParseMongoBytes("~/.mongo.analyst", data)
	if mf.Err != "" || mf.Entry == nil {
		t.Fatalf("parse = %+v", mf)
	}
	entry := mf.Entry
	if entry.Host != "mongo2" || entry.Port != 27018 || entry.Database != "archive" || entry.User != "analyst" {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Password != "two" {
		t.Fatalf("password lost")
	}
	if !strings.Contains(entry.Selector, "authSource=admin") || !strings.Contains(entry.Selector, "mongo2:27018") {
		t.Fatalf("selector lost hosts/options: %q", entry.Selector)
	}
	if strings.Contains(entry.Selector, "two") {
		t.Fatalf("selector leaked the password: %q", entry.Selector)
	}
}

func TestParseMongoBytesKeepsReplicaSetAndOptions(t *testing.T) {
	mf := ParseMongoBytes("~/.mongo.analyst", "mongodb://analyst:pw@mongo1:27017,mongo2:27018/analytics?authSource=admin&replicaSet=rs0\n")
	if mf.Err != "" || mf.Entry == nil {
		t.Fatalf("parse = %+v", mf)
	}
	if mf.Entry.Host != "mongo1" || mf.Entry.Port != 27017 || mf.Entry.Database != "analytics" {
		t.Fatalf("entry = %+v", mf.Entry)
	}
	for _, want := range []string{"mongo2:27018", "authSource=admin", "replicaSet=rs0"} {
		if !strings.Contains(mf.Entry.Selector, want) {
			t.Fatalf("selector lost %q: %q", want, mf.Entry.Selector)
		}
	}
	if strings.Contains(mf.Entry.Selector, "pw") {
		t.Fatalf("selector leaked password: %q", mf.Entry.Selector)
	}
}

func TestParseMongoHostsAndSelector(t *testing.T) {
	hosts, err := ParseMongoHosts("mongo1:27017, mongo2:27018,mongo3")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(hosts, ",") != "mongo1:27017,mongo2:27018,mongo3:27017" {
		t.Fatalf("hosts = %+v", hosts)
	}
	if _, err := ParseMongoHosts("mongo1:bad"); err == nil {
		t.Fatal("invalid port accepted")
	}
	selector := "mongodb://analyst@mongo1:27017,mongo2:27018/analytics?authSource=admin"
	if got := strings.Join(MongoSelectorHosts(selector), ","); got != "mongo1:27017,mongo2:27018" {
		t.Fatalf("selector hosts = %q", got)
	}
}

func TestParseMongoBytesMalformedNamesFile(t *testing.T) {
	mf := ParseMongoBytes("~/.mongo.bad", "mongodb://mongo1:notaport/analytics\n")
	if mf.Err == "" || mf.Entry != nil {
		t.Fatalf("malformed parse = %+v", mf)
	}
	if strings.Contains(mf.Err, "mongodb://") {
		t.Fatalf("error leaked the URI: %q", mf.Err)
	}
}

func TestParseMongoBytesRejectsSRVExplicitly(t *testing.T) {
	mf := ParseMongoBytes("~/.mongo.srv", "mongodb+srv://u:p@cluster.example/db\n")
	if mf.Err != "unsupported scheme mongodb+srv" {
		t.Fatalf("err = %q", mf.Err)
	}
}

func TestParseMongoBytesNoURI(t *testing.T) {
	mf := ParseMongoBytes("~/.mongo.notes", "just notes\n# comment\n")
	if mf.Err != "no mongodb:// URI found" {
		t.Fatalf("err = %q", mf.Err)
	}
}

func TestParseMongoFileMissing(t *testing.T) {
	mf := ParseMongoFile(filepath.Join(t.TempDir(), ".mongo.missing"))
	if !mf.Missing || mf.Err == "" {
		t.Fatalf("missing = %+v", mf)
	}
}

func TestMongoFilesDiscovery(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".mongo.analyst"), "mongodb://u:p@h/db\n", 0o600)
	writeFile(t, filepath.Join(home, ".mongo"), "mongodb://u:p@h/db\n", 0o600)
	writeFile(t, filepath.Join(home, ".mongorc.js"), "ignored\n", 0o600)
	if err := os.Mkdir(filepath.Join(home, ".mongo.dir"), 0o700); err != nil {
		t.Fatal(err)
	}
	files, err := MongoFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	if filepath.Base(files[0]) != ".mongo" || filepath.Base(files[1]) != ".mongo.analyst" {
		t.Fatalf("files = %v", files)
	}
}

func TestMatchMongoIgnoresCaseOfHost(t *testing.T) {
	entry := MongoEntry{Host: "MONGO1", Port: 27017, Database: "db", User: "u"}
	if !MatchMongo(entry, "mongo1", 27017, "db", "u") {
		t.Fatal("host comparison should ignore case")
	}
	if MatchMongo(entry, "mongo1", 27017, "other", "u") {
		t.Fatal("database must match")
	}
}

func TestParseMongoBytesStripsPasswordQueryFromSelector(t *testing.T) {
	mf := ParseMongoBytes("~/.mongo.secret", "mongodb://u@h:27017/db?pwd=hunter2&authSource=admin\n")
	if mf.Err != "" || mf.Entry == nil {
		t.Fatalf("parse = %+v", mf)
	}
	if strings.Contains(mf.Entry.Selector, "hunter2") || strings.Contains(strings.ToLower(mf.Entry.Selector), "pwd=") {
		t.Fatalf("selector leaked a query secret: %q", mf.Entry.Selector)
	}
	if !strings.Contains(mf.Entry.Selector, "authSource=admin") {
		t.Fatalf("selector lost non-secret options: %q", mf.Entry.Selector)
	}
}

func TestParseMongoBytesMalformedLineInvalidatesFile(t *testing.T) {
	data := "mongodb://u:p@mongo1:27017/db\nmongodb://mongo1:notaport/db\n"
	mf := ParseMongoBytes("~/.mongo.bad", data)
	if mf.Err == "" || mf.Entry != nil {
		t.Fatalf("a malformed line must invalidate the file: %+v", mf)
	}
	if strings.Contains(mf.Err, "p@mongo1") {
		t.Fatalf("error leaked the URI: %q", mf.Err)
	}
}

func FuzzParseMongoBytes(f *testing.F) {
	for _, seed := range []string{
		"mongodb://user:pass@host:27017/db\n",
		"# comment\nmongodb://user@host/db?authSource=admin\n",
		"mongodb+srv://user@host/db\n",
		"not a mongo line\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		result := ParseMongoBytes("~/.mongo.fuzz", data)
		if result.Entry != nil {
			if strings.Contains(result.Entry.Selector, "password=") ||
				strings.Contains(result.Entry.Selector, "pwd=") {
				t.Fatalf("selector leaked a password: %q", result.Entry.Selector)
			}
		}
	})
}
