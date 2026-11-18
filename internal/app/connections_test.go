package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
	"github.com/mcgru/db-adm-tui/internal/profile"
)

type fakeConn struct {
	closed bool
}

func (c *fakeConn) Ping(ctx context.Context) error { return nil }
func (c *fakeConn) Close() error                   { c.closed = true; return nil }

type fakeConnector struct {
	err      error
	password string
	profile  domain.Profile
	conns    []*fakeConn
}

func (f *fakeConnector) Connect(ctx context.Context, p domain.Profile, password string) (ports.SessionBundle, error) {
	f.password = password
	f.profile = p
	if f.err != nil {
		return ports.SessionBundle{}, f.err
	}
	conn := &fakeConn{}
	f.conns = append(f.conns, conn)
	return ports.SessionBundle{Session: conn, Catalog: fakeCatalog{database: p.Database}}, nil
}
func (f *fakeConnector) Close() error { return nil }

type fakeCatalog struct {
	database string
}

func (c fakeCatalog) Databases(ctx context.Context) ([]domain.Node, error) {
	return []domain.Node{{
		Ref:        domain.ObjectRef{Kind: domain.KindDatabase, Database: c.database, Name: c.database},
		Name:       c.database,
		Kind:       domain.KindDatabase,
		Expandable: true,
	}}, nil
}
func (c fakeCatalog) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	return []domain.Node{{
		Ref:        domain.ObjectRef{Kind: domain.KindSchema, Database: ref.Database, Schema: "public", Name: "public"},
		Name:       "public",
		Kind:       domain.KindSchema,
		Expandable: true,
	}}, nil
}
func (c fakeCatalog) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	return nil, nil
}
func (c fakeCatalog) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	return "", nil
}

type fixture struct {
	home      string
	store     *profile.Store
	resolver  *profile.Resolver
	connector *fakeConnector
	svc       *Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PGPASSFILE", "")
	storeDir := filepath.Join(home, ".config", "db-adm-tui")
	store := profile.NewStore(storeDir)
	resolver := profile.NewResolverAt(home, filepath.Join(home, ".pgpass"))
	connector := &fakeConnector{}
	svc := NewService(store, resolver, connector)
	svc.SetClock(func() time.Time {
		return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	})
	return &fixture{home: home, store: store, resolver: resolver, connector: connector, svc: svc}
}

func (f *fixture) write(t *testing.T, name, content string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(f.home, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func (f *fixture) seed(t *testing.T, profiles ...domain.Profile) {
	t.Helper()
	if err := f.store.Save(context.Background(), profiles); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverHonorsPGPASSFILE(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := filepath.Join(home, "custom.pass")
	if err := os.WriteFile(custom, []byte("db1:5432:orders:app_ro:secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(custom, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PGPASSFILE", custom)
	resolver, err := profile.NewResolver()
	if err != nil {
		t.Fatal(err)
	}
	store := profile.NewStore(filepath.Join(home, ".config", "db-adm-tui"))
	svc := NewService(store, resolver, &fakeConnector{})
	d := svc.Discover()
	if len(d.Entries) != 1 {
		t.Fatalf("entries = %+v", d.Entries)
	}
	if d.Entries[0].Source != "~/custom.pass" {
		t.Fatalf("source = %q", d.Entries[0].Source)
	}
	if strings.Contains(d.Entries[0].Label, "secret") {
		t.Fatal("discovery leaked a secret")
	}
}

func TestProfilesEmptyAndStorageError(t *testing.T) {
	f := newFixture(t)
	views, err := f.svc.Profiles(context.Background())
	if err != nil || len(views) != 0 {
		t.Fatalf("views = %+v, %v", views, err)
	}
	if err := os.MkdirAll(filepath.Dir(f.store.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.store.Path(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	views, err = f.svc.Profiles(context.Background())
	if err == nil || len(views) != 0 {
		t.Fatalf("corrupt views = %+v, %v", views, err)
	}
}

func TestDiscoverPgpassAndMongo(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", strings.Join([]string{
		"db1:5432:orders:app_ro:first",
		"db2:5432:billing:billing_ro:second",
		"malformed",
	}, "\n"), 0o600)
	f.write(t, ".mongo.analyst", "mongodb://analyst:hunter2@mongo1:27017/analytics\n", 0o600)
	f.write(t, ".mongo.bad", "mongodb://mongo1:notaport/analytics\n", 0o600)

	d := f.svc.Discover()
	if d.Empty {
		t.Fatal("discovery must not be empty")
	}
	if len(d.Status) != 0 {
		t.Fatalf("status = %v", d.Status)
	}
	var pgpass, mongo, warnings int
	for _, entry := range d.Entries {
		switch entry.Kind {
		case "pgpass":
			if entry.Warning != "" {
				warnings++
				continue
			}
			pgpass++
		case "mongo-file":
			mongo++
		}
		if strings.Contains(entry.Label+entry.Source+entry.Warning+entry.Error, "hunter2") {
			t.Fatalf("discovery leaked a password: %+v", entry)
		}
	}
	if pgpass != 2 || mongo != 2 || warnings != 1 {
		t.Fatalf("pgpass=%d mongo=%d warnings=%d entries=%+v", pgpass, mongo, warnings, d.Entries)
	}
}

func TestDiscoverReportsInsecurePgpass(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o644)
	d := f.svc.Discover()
	if len(d.Entries) != 0 {
		t.Fatalf("entries = %+v", d.Entries)
	}
	if len(d.Status) != 1 || !strings.Contains(d.Status[0], "ignored") || !strings.Contains(d.Status[0], "0600") {
		t.Fatalf("status = %v", d.Status)
	}
	if strings.Contains(strings.Join(d.Status, " "), "secret") {
		t.Fatal("discovery status leaked the password")
	}
}

func TestImportCreatesAndRefreshesProfiles(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o600)
	f.write(t, ".mongo.analyst", "mongodb://analyst:hunter2@mongo1:27017/analytics?authSource=admin\n", 0o600)
	d := f.svc.Discover()
	var selected []ImportEntry
	for _, entry := range d.Entries {
		if entry.Selectable {
			selected = append(selected, entry)
		}
	}
	summary, err := f.svc.Import(context.Background(), selected)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Created != 2 || summary.Updated != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	views, err := f.svc.Profiles(context.Background())
	if err != nil || len(views) != 2 {
		t.Fatalf("views = %+v, %v", views, err)
	}
	for _, view := range views {
		if view.Profile.Source == nil {
			t.Fatalf("profile lost source: %+v", view.Profile)
		}
		if view.Profile.Engine == domain.EngineMongoDB {
			selector := view.Profile.Source.Selector
			if !strings.Contains(selector, "authSource=admin") || strings.Contains(selector, "hunter2") {
				t.Fatalf("mongo selector = %q", selector)
			}
		}
	}
	data, err := os.ReadFile(f.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret", "hunter2"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("profiles.json leaked %q", secret)
		}
	}

	// Re-import updates fields and preserves identity/created time.
	created := map[string]time.Time{}
	for _, view := range views {
		created[view.Profile.Key()] = view.Profile.CreatedAt
	}
	f.write(t, ".mongo.analyst", "mongodb://analyst:newpass@mongo2:27017/archive\n", 0o600)
	d = f.svc.Discover()
	selected = nil
	for _, entry := range d.Entries {
		if entry.Selectable {
			selected = append(selected, entry)
		}
	}
	summary, err = f.svc.Import(context.Background(), selected)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Updated < 1 {
		t.Fatalf("summary = %+v", summary)
	}
	views, _ = f.svc.Profiles(context.Background())
	for _, view := range views {
		if view.Profile.Engine == domain.EngineMongoDB {
			if view.Profile.Host != "mongo2" || view.Profile.Database != "archive" {
				t.Fatalf("mongo not refreshed: %+v", view.Profile)
			}
			if !strings.Contains(view.Profile.Source.Selector, "mongo2") || strings.Contains(view.Profile.Source.Selector, "authSource") {
				t.Fatalf("mongo selector not refreshed: %q", view.Profile.Source.Selector)
			}
			if !view.Profile.CreatedAt.Equal(created[view.Profile.Key()]) {
				t.Fatalf("created time not preserved: %v != %v", view.Profile.CreatedAt, created[view.Profile.Key()])
			}
		}
	}
}

func TestImportPreservesExistingLabel(t *testing.T) {
	f := newFixture(t)
	existing := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, existing)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o600)
	d := f.svc.Discover()
	var selected []ImportEntry
	for _, entry := range d.Entries {
		if entry.Selectable {
			selected = append(selected, entry)
		}
	}
	if _, err := f.svc.Import(context.Background(), selected); err != nil {
		t.Fatal(err)
	}
	views, err := f.svc.Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].Profile.Label != "prod-ro" {
		t.Fatalf("label not preserved: %+v", views)
	}
}

func TestConnectPromptRequiredAndNeverStoresSecret(t *testing.T) {
	f := newFixture(t)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	if _, err := f.svc.Connect(context.Background(), p, Credential{}); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("err = %v", err)
	}
	result, err := f.svc.Connect(context.Background(), p, Credential{Password: "hunter2", Provided: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Session == nil {
		t.Fatal("no session")
	}
	if f.connector.password != "hunter2" {
		t.Fatalf("password passed = %q", f.connector.password)
	}
	if result.Profile.LastUsed == nil {
		t.Fatal("last_used not set in memory")
	}
	if err := result.Session.Close(); err != nil {
		t.Fatal(err)
	}
	// Unsaved profiles are not persisted.
	views, _ := f.svc.Profiles(context.Background())
	if len(views) != 0 {
		t.Fatalf("unsaved profile persisted: %+v", views)
	}
}

func TestConnectReadsSourceAndUpdatesLastUsed(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:first\ndb1:5432:billing:app_ro:billingpass\n", 0o600)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	if f.connector.password != "first" {
		t.Fatalf("password = %q", f.connector.password)
	}
	if result.Profile.LastUsed == nil {
		t.Fatal("last_used missing")
	}
	views, err := f.svc.Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].Profile.LastUsed == nil {
		t.Fatalf("last_used not saved: %+v", views)
	}
	_ = result.Session.Close()
}

func TestConnectFailureNamesProfileAndSourceWithoutSecret(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:sup3r\n", 0o600)
	mysql := domain.Profile{
		Label: "prod-ro", Engine: domain.EngineMySQL, Host: "db1", Port: 3306,
		Database: "orders", User: "app_ro",
	}
	f.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed for sup3r"}
	_, err := f.svc.Connect(context.Background(), mysql, Credential{Password: "sup3r", Provided: true})
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, "prod-ro") {
		t.Fatalf("error does not name profile: %q", message)
	}
	if !strings.Contains(message, "authentication failed") {
		t.Fatalf("error lost cause: %q", message)
	}
	if strings.Contains(message, "sup3r") {
		t.Fatalf("error leaked secret: %q", message)
	}
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.missing"},
	}
	_, err = f.svc.Connect(context.Background(), p, Credential{})
	if err == nil || !strings.Contains(err.Error(), "~/.missing") {
		t.Fatalf("source error = %v", err)
	}
	if domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("source error kind = %v", domain.KindOf(err))
	}
}

func TestPrepareNewConnectionMatchesSources(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o600)
	f.write(t, ".mongo.analyst", "mongodb://analyst:pw@mongo1:27017/analytics\n", 0o600)

	pg, err := f.svc.PrepareNewConnection(context.Background(), "postgres://app_ro@db1:5432/orders", "prod-ro")
	if err != nil {
		t.Fatal(err)
	}
	if pg.Source == nil || pg.Source.Kind != domain.SourcePgpass || pg.Source.Path != "~/.pgpass" {
		t.Fatalf("pg source = %+v", pg.Source)
	}
	mongo, err := f.svc.PrepareNewConnection(context.Background(), "mongodb://analyst@mongo1:27017/analytics", "")
	if err != nil {
		t.Fatal(err)
	}
	if mongo.Source == nil || mongo.Source.Kind != domain.SourceMongoFile || mongo.Source.Path != "~/.mongo.analyst" {
		t.Fatalf("mongo source = %+v", mongo.Source)
	}
	mysql, err := f.svc.PrepareNewConnection(context.Background(), "mysql://billing_ro@mysql1/billing", "")
	if err != nil {
		t.Fatal(err)
	}
	if mysql.Source != nil {
		t.Fatalf("mysql must not match a source: %+v", mysql.Source)
	}
}

func TestSaveProfileNeverStoresSecret(t *testing.T) {
	f := newFixture(t)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	if err := f.svc.SaveProfile(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(f.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"password", "hunter2", "secret"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("profiles.json contains %q", forbidden)
		}
	}
	views, _ := f.svc.Profiles(context.Background())
	if len(views) != 1 || views[0].Profile.Label != "prod-ro" {
		t.Fatalf("views = %+v", views)
	}
}

type failingStore struct {
	inner ports.ProfileStore
}

func (f failingStore) Load(ctx context.Context) ([]domain.Profile, error) {
	return f.inner.Load(ctx)
}

func (f failingStore) Save(ctx context.Context, profiles []domain.Profile) error {
	return errors.New("disk full")
}

func TestImportAndSaveRefuseCorruptStore(t *testing.T) {
	f := newFixture(t)
	if err := os.MkdirAll(filepath.Dir(f.store.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.store.Path(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := ImportEntry{
		ID: "one", Kind: "pgpass", Source: "~/.pgpass", Engine: domain.EnginePostgreSQL,
		Host: "db1", Port: 5432, Database: "orders", User: "app_ro", Label: "prod-ro", Selectable: true,
	}
	if _, err := f.svc.Import(context.Background(), []ImportEntry{entry}); err == nil {
		t.Fatal("import must refuse an unreadable store")
	}
	if _, err := os.Stat(f.store.Path()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(f.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{bad" {
		t.Fatalf("corrupt store was overwritten: %q", data)
	}
	p := domain.Profile{Label: "x", Engine: domain.EnginePostgreSQL, Host: "h", Port: 5432}
	if err := f.svc.SaveProfile(context.Background(), p); err == nil {
		t.Fatal("save must refuse an unreadable store")
	}
	data, _ = os.ReadFile(f.store.Path())
	if string(data) != "{bad" {
		t.Fatalf("corrupt store was overwritten by save: %q", data)
	}
}

func TestImportCountsUniqueProfiles(t *testing.T) {
	f := newFixture(t)
	entry := ImportEntry{
		ID: "one", Kind: "pgpass", Source: "~/.pgpass", Engine: domain.EnginePostgreSQL,
		Host: "db1", Port: 5432, Database: "orders", User: "app_ro", Label: "prod-ro", Selectable: true,
	}
	second := entry
	second.ID = "two"
	summary, err := f.svc.Import(context.Background(), []ImportEntry{entry, second})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Created != 1 || summary.Updated != 0 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestConnectUsesFreshMongoSelector(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".mongo.analyst", "mongodb://analyst:first@mongo1:27017/analytics?authSource=admin\n", 0o600)
	p := domain.Profile{
		Label: "analytics", Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
		Database: "analytics", User: "analyst",
		Source: &domain.Source{Kind: domain.SourceMongoFile, Path: "~/.mongo.analyst"},
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	if f.connector.profile.Source == nil || !strings.Contains(f.connector.profile.Source.Selector, "authSource=admin") {
		t.Fatalf("first selector = %+v", f.connector.profile.Source)
	}
	_ = result.Session.Close()

	f.write(t, ".mongo.analyst", "mongodb://analyst:second@mongo1:27017/analytics?tls=true\n", 0o600)
	result, err = f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	selector := f.connector.profile.Source.Selector
	if !strings.Contains(selector, "tls=true") || strings.Contains(selector, "authSource") {
		t.Fatalf("stale selector = %q", selector)
	}
	if f.connector.password != "second" {
		t.Fatalf("password = %q", f.connector.password)
	}
	_ = result.Session.Close()
}

func TestConnectWarnsWhenLastUsedCannotBeSaved(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o600)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, p)
	svc := NewService(failingStore{inner: f.store}, f.resolver, f.connector)
	result, err := svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Session == nil {
		t.Fatal("connection must stay open")
	}
	if !strings.Contains(result.Warning, "connection is still open") {
		t.Fatalf("warning = %q", result.Warning)
	}
	_ = result.Session.Close()
}

func TestSessionSwitchesPostgresDatabase(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:first\ndb1:5432:billing:app_ro:billingpass\n", 0o600)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Session.CurrentDatabase(); got != "orders" {
		t.Fatalf("database = %q", got)
	}
	if _, err := result.Session.Children(context.Background(), domain.ObjectRef{
		Kind: domain.KindDatabase, Database: "billing", Name: "billing",
	}); err != nil {
		t.Fatal(err)
	}
	if got := result.Session.CurrentDatabase(); got != "billing" {
		t.Fatalf("switched database = %q", got)
	}
	if len(f.connector.conns) != 2 || !f.connector.conns[0].closed {
		t.Fatalf("expected one old closed connection and one live: %+v", f.connector.conns)
	}
	if f.connector.profile.Database != "billing" {
		t.Fatalf("connector profile = %+v", f.connector.profile)
	}
	_ = result.Session.Close()
}

func TestSessionSwitchRereadsSourceAndUsesNewPassword(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:first\n", 0o600)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:first\ndb1:5432:billing:app_ro:second\n", 0o600)
	if _, err := result.Session.Children(context.Background(), domain.ObjectRef{
		Kind: domain.KindDatabase, Database: "billing", Name: "billing",
	}); err != nil {
		t.Fatal(err)
	}
	if f.connector.password != "second" {
		t.Fatalf("switch password = %q", f.connector.password)
	}
	_ = result.Session.Close()
}

func TestSessionSwitchFailureMarksDead(t *testing.T) {
	f := newFixture(t)
	f.write(t, ".pgpass", "db1:5432:orders:app_ro:first\ndb1:5432:billing:app_ro:billingpass\n", 0o600)
	p := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
		Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{})
	if err != nil {
		t.Fatal(err)
	}
	f.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed"}
	if _, err := result.Session.Children(context.Background(), domain.ObjectRef{
		Kind: domain.KindDatabase, Database: "billing", Name: "billing",
	}); err == nil {
		t.Fatal("switch failure must return an error")
	}
	if result.Session.Alive() {
		t.Fatal("failed switch must leave the session dead")
	}
}

func TestSessionSwitchUsesRetainedPromptPassword(t *testing.T) {
	f := newFixture(t)
	p := domain.Profile{
		Label: "prompt-pg", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	f.seed(t, p)
	result, err := f.svc.Connect(context.Background(), p, Credential{Password: "entered", Provided: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := result.Session.Children(context.Background(), domain.ObjectRef{
		Kind: domain.KindDatabase, Database: "billing", Name: "billing",
	}); err != nil {
		t.Fatal(err)
	}
	if result.Session.CurrentDatabase() != "billing" {
		t.Fatalf("database = %q", result.Session.CurrentDatabase())
	}
	if f.connector.password != "entered" {
		t.Fatalf("switch password = %q", f.connector.password)
	}
	_ = result.Session.Close()
}
