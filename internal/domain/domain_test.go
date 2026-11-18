package domain

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEngineMetadata(t *testing.T) {
	cases := []struct {
		engine Engine
		name   string
		port   int
		scheme string
	}{
		{EnginePostgreSQL, "PostgreSQL", 5432, "postgres"},
		{EngineMySQL, "MySQL", 3306, "mysql"},
		{EngineMongoDB, "MongoDB", 27017, "mongodb"},
	}
	for _, tc := range cases {
		if tc.engine.DisplayName() != tc.name {
			t.Errorf("%s display = %q", tc.engine, tc.engine.DisplayName())
		}
		if tc.engine.DefaultPort() != tc.port {
			t.Errorf("%s port = %d", tc.engine, tc.engine.DefaultPort())
		}
		if tc.engine.Scheme() != tc.scheme {
			t.Errorf("%s scheme = %q", tc.engine, tc.engine.Scheme())
		}
		if !tc.engine.Valid() {
			t.Errorf("%s should be valid", tc.engine)
		}
	}
	if Engine("sqlite").Valid() {
		t.Fatal("sqlite must not be valid")
	}
	if e, ok := ParseScheme("postgresql"); !ok || e != EnginePostgreSQL {
		t.Fatalf("postgresql scheme = %v, %v", e, ok)
	}
}

func TestProfileAllowNoTLSAliases(t *testing.T) {
	for _, key := range []string{"allow_no_tls", "allowNoTLS", "notls"} {
		data := []byte(`{"label":"x","engine":"postgresql","host":"h","port":5432,` +
			`"database":"d","user":"u","` + key + `":true}`)
		var p Profile
		if err := json.Unmarshal(data, &p); err != nil {
			t.Fatal(err)
		}
		if !p.AllowNoTLS {
			t.Fatalf("%s did not enable AllowNoTLS", key)
		}
	}
	var p Profile
	if err := json.Unmarshal([]byte(`{"allow_no_tls":false,"notls":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.AllowNoTLS {
		t.Fatal("an enabled alias must win over a false canonical key")
	}
}

func TestProfileKeyIsDeterministicAndSourceAware(t *testing.T) {
	base := Profile{Engine: EnginePostgreSQL, Host: "DB1", Port: 5432, Database: "orders", User: "app_ro"}
	if base.Key() != base.Key() {
		t.Fatal("key is not deterministic")
	}
	other := base
	other.Host = "db1"
	if base.Key() != other.Key() {
		t.Fatal("host case must not change identity")
	}
	mongo := Profile{
		Engine: EngineMongoDB, Host: "mongo1", Port: 27017, Database: "analytics", User: "analyst",
		Source: &Source{Kind: SourceMongoFile, Path: "~/.mongo.analyst"},
	}
	changed := mongo
	changed.Host = "mongo2"
	changed.Database = "other"
	if mongo.Key() != changed.Key() {
		t.Fatal("mongo identity must follow the source file")
	}
}

func TestRedactedURIHasNoPassword(t *testing.T) {
	p := Profile{Engine: EnginePostgreSQL, Host: "db1", Port: 5432, Database: "orders", User: "app_ro"}
	uri := p.RedactedURI()
	if uri != "postgres://app_ro@db1:5432/orders" {
		t.Fatalf("uri = %q", uri)
	}
	if strings.Contains(uri, ":pass") || strings.Contains(uri, "***") {
		t.Fatalf("redacted URI looks wrong: %q", uri)
	}
}

func TestNormalizePromptSourceAndLabel(t *testing.T) {
	p := Profile{
		Engine:   EngineMySQL,
		Host:     " mysql1 ",
		Port:     3306,
		Database: "billing",
		User:     "billing_ro",
		Source:   &Source{Kind: SourcePrompt},
	}
	n := p.Normalize()
	if n.Source != nil {
		t.Fatal("prompt source should normalize to nil")
	}
	if n.Label != "billing@mysql1" {
		t.Fatalf("label = %q", n.Label)
	}
	if n.Host != "mysql1" {
		t.Fatalf("host = %q", n.Host)
	}
}

func TestValidate(t *testing.T) {
	good := Profile{Label: "x", Engine: EnginePostgreSQL, Host: "h", Port: 5432}
	if err := good.Validate(); err != nil {
		t.Fatalf("good profile invalid: %v", err)
	}
	bad := good
	bad.Port = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("port 0 must be invalid")
	}
	bad = good
	bad.Source = &Source{Kind: SourcePgpass}
	if err := bad.Validate(); err == nil {
		t.Fatal("source without path must be invalid")
	}
}

func TestRedactNeverLeaksUserinfoOrPasswordKV(t *testing.T) {
	in := "postgres://app:sup3r@db1:5432/orders?password=hunter2&sslmode=disable"
	got := RedactSecret(Redact(in), "sup3r")
	for _, secret := range []string{"sup3r", "hunter2"} {
		if strings.Contains(got, secret) {
			t.Fatalf("redaction leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "app:***@db1") {
		t.Fatalf("userinfo not redacted: %s", got)
	}
}

func TestRedactSecretCoversEncodedForms(t *testing.T) {
	secret := "p@ss word/with:chars"
	raw := "connect failed for " + secret + " and " + url.QueryEscape(secret) + " and " + url.PathEscape(secret)
	got := RedactSecret(raw, secret)
	for _, form := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
		if strings.Contains(got, form) {
			t.Fatalf("redaction left %q in %q", form, got)
		}
	}
}

func TestKindOf(t *testing.T) {
	err := &Error{Kind: KindTimeout, Message: "timed out"}
	if KindOf(err) != KindTimeout {
		t.Fatalf("kind = %v", KindOf(err))
	}
	if KindOf(nil) != KindEngine {
		t.Fatal("nil should be engine kind")
	}
}

func TestLastUsedTimestampIsNotPersistedAsSecret(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	p := Profile{Label: "x", Engine: EngineMongoDB, Host: "h", Port: 27017, LastUsed: &now}
	if p.LastUsed == nil || !p.LastUsed.Equal(now) {
		t.Fatal("last used lost")
	}
}

func TestCatalogAccessAndIdentity(t *testing.T) {
	if AccessFrom(false, true) != AccessWrite {
		t.Fatal("write must map to rw")
	}
	if AccessFrom(true, false) != AccessRead {
		t.Fatal("read must map to ro")
	}
	if AccessFrom(false, false) != AccessNone {
		t.Fatal("no access must map to —")
	}
	for _, kind := range []ObjectKind{KindTable, KindCollection, KindView} {
		if !kind.AccessMarker() {
			t.Fatalf("%s must carry a marker", kind)
		}
	}
	for _, kind := range []ObjectKind{KindSequence, KindDatabase, KindSchema} {
		if kind.AccessMarker() {
			t.Fatalf("%s must not carry a marker", kind)
		}
	}
	a := ObjectRef{Engine: EnginePostgreSQL, Kind: KindTable, Database: "orders", Schema: "public", Name: "customers"}
	b := a
	if a.Key() != b.Key() {
		t.Fatal("key must be deterministic")
	}
	b.Database = "billing"
	if a.Key() == b.Key() {
		t.Fatal("database must change identity")
	}
}
