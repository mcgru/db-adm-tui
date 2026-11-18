package tui

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
	"github.com/mcgru/db-adm-tui/internal/profile"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	got = StripANSI(got)
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to create)", path, err)
	}
	if got != string(want) {
		t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

type fakeConn struct {
	closed bool
}

func (c *fakeConn) Ping(ctx context.Context) error { return nil }
func (c *fakeConn) Close() error                   { c.closed = true; return nil }

type fakeConnector struct {
	err      error
	password string
	conns    []*fakeConn
}

func (f *fakeConnector) Connect(ctx context.Context, p domain.Profile, password string) (ports.Connection, error) {
	f.password = password
	if f.err != nil {
		return nil, f.err
	}
	conn := &fakeConn{}
	f.conns = append(f.conns, conn)
	return conn, nil
}
func (f *fakeConnector) Close() error { return nil }

type testEnv struct {
	home      string
	store     *profile.Store
	resolver  *profile.Resolver
	connector *fakeConnector
	svc       *app.Service
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PGPASSFILE", "")
	store := profile.NewStore(filepath.Join(home, ".config", "db-adm-tui"))
	resolver := profile.NewResolverAt(home, filepath.Join(home, ".pgpass"))
	connector := &fakeConnector{}
	svc := app.NewService(store, resolver, connector)
	svc.SetClock(func() time.Time {
		return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	})
	return &testEnv{home: home, store: store, resolver: resolver, connector: connector, svc: svc}
}

func (e *testEnv) model(t *testing.T) Model {
	t.Helper()
	return New(e.svc)
}

func (e *testEnv) write(t *testing.T, name, content string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(e.home, name)
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

func (e *testEnv) seed(t *testing.T, profiles ...domain.Profile) {
	t.Helper()
	if err := e.store.Save(context.Background(), profiles); err != nil {
		t.Fatal(err)
	}
}

func sampleProfiles() []domain.Profile {
	created := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	last := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	return []domain.Profile{
		{
			Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1.internal", Port: 5432,
			Database: "orders", User: "app_ro",
			Source:    &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"},
			CreatedAt: created, UpdatedAt: created, LastUsed: &last,
		},
		{
			Label: "analytics", Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
			Database: "analytics", User: "analyst",
			Source:    &domain.Source{Kind: domain.SourceMongoFile, Path: "~/.mongo.analyst"},
			CreatedAt: created, UpdatedAt: created, LastUsed: &last,
		},
		{
			Label: "billing", Engine: domain.EngineMySQL, Host: "mysql1", Port: 3306,
			Database: "billing", User: "billing_ro",
			CreatedAt: created, UpdatedAt: created,
		},
	}
}

func seedStandard(t *testing.T, e *testEnv) {
	t.Helper()
	e.write(t, ".pgpass", "db1.internal:5432:orders:app_ro:pgsecret\n", 0o600)
	e.write(t, ".mongo.analyst", "mongodb://analyst:mongosecret@mongo1:27017/analytics\n", 0o600)
	e.seed(t, sampleProfiles()...)
}

func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "space":
		return tea.KeyPressMsg{Text: " ", Code: tea.KeySpace}
	case "f1":
		return tea.KeyPressMsg{Code: tea.KeyF1}
	}
	if strings.HasPrefix(s, "ctrl+") {
		runes := []rune(strings.TrimPrefix(s, "ctrl+"))
		if len(runes) == 1 {
			return tea.KeyPressMsg{Code: runes[0], Mod: tea.ModCtrl}
		}
		return tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl}
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return tea.KeyPressMsg{}
	}
	return tea.KeyPressMsg{Text: s, Code: runes[0]}
}

func press(t *testing.T, m Model, key string) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(keyMsg(key))
	mm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update(%q) returned %T", key, next)
	}
	return mm, cmd
}

func typeString(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		m, _ = press(t, m, string(r))
	}
	return m
}

func runCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	next, _ := m.Update(cmd())
	mm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return mm
}

// connectSelected presses Enter on the picker, answers a password prompt when
// present, and applies the asynchronous connection result.
func connectSelected(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := press(t, m, "enter")
	if m.modal == modalPassword {
		m = typeString(t, m, "hunter2")
		m, cmd = press(t, m, "enter")
	}
	if cmd == nil {
		t.Fatalf("no connect command: message=%q modal=%v", m.message, m.modal)
	}
	return runCmd(t, m, cmd)
}

func selectLabel(t *testing.T, m Model, label string) Model {
	t.Helper()
	for i := 0; i < 20; i++ {
		if p, ok := m.currentPickerProfile(); ok && p.Label == label {
			return m
		}
		before := m.pickerSel
		m, _ = press(t, m, "down")
		if m.pickerSel == before {
			break
		}
	}
	for i := 0; i < 20; i++ {
		if p, ok := m.currentPickerProfile(); ok && p.Label == label {
			return m
		}
		before := m.pickerSel
		m, _ = press(t, m, "up")
		if m.pickerSel == before {
			break
		}
	}
	t.Fatalf("profile %q not found", label)
	return m
}

func workspace(t *testing.T) Model {
	t.Helper()
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	if m.screen != ScreenWorkspace {
		t.Fatalf("expected workspace, message=%q modal=%v", m.message, m.modal)
	}
	return m
}

func assertNoSecrets(t *testing.T, text string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(text, secret) {
			t.Fatalf("view leaked %q:\n%s", secret, text)
		}
	}
}
