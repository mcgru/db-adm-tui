package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/history"
)

func TestCredentialAndANSILeakAudit(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	historyDir := filepath.Join(e.home, "state", "db-adm-tui")
	historyStore := history.NewStore(historyDir)
	e.svc.SetHistory(historyStore)
	malicious := "ok\x1b]8;;https://evil.example\x07click\x1b]8;;\x07\x07\u202e"
	e.connector.data = &fakeDataReader{total: 1, pageFn: func(domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{Grid: domain.Grid{
			Columns: []domain.Column{{Name: "payload"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: malicious, Kind: domain.CellText}}}},
		}}, nil
	}}
	e.connector.query = &fakeQueryExecutor{err: errors.New("engine failed with pgsecret \x1b]8;;https://evil.example\x07boom\x1b]8;;\x07")}

	m := connectSelected(t, selectLabel(t, e.model(t), "prod-ro"))
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "down")
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "down")
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	dataView := StripANSI(m.View().Content)
	for _, bad := range []string{"evil.example", "\x07", "\u202e"} {
		if strings.Contains(dataView, bad) {
			t.Fatalf("data view leaked %q:\n%s", bad, dataView)
		}
	}
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT 1")
	m, cmd = press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)

	view := StripANSI(m.View().Content)
	for _, bad := range []string{"evil.example", "\x07", "\u202e"} {
		if strings.Contains(view, bad) {
			t.Fatalf("view leaked %q:\n%s", bad, view)
		}
	}
	if strings.Contains(view, "pgsecret") {
		t.Fatalf("view leaked the connection secret:\n%s", view)
	}

	historyFiles, err := filepath.Glob(filepath.Join(historyDir, "commands.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(historyFiles) == 0 {
		t.Fatal("history was not written")
	}
	for _, path := range historyFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{"pgsecret", "evil.example", "\x07"} {
			if strings.Contains(string(data), bad) {
				t.Fatalf("%s leaked %q", path, bad)
			}
		}
	}
	if data, err := os.ReadFile(e.store.Path()); err == nil && strings.Contains(string(data), "pgsecret") {
		t.Fatal("profiles.json leaked the secret")
	}
	if strings.Contains(strings.Join(os.Args, " "), "pgsecret") {
		t.Fatal("argv leaked the secret")
	}
}

func TestANSIInPickerFieldsIsSanitized(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	profiles := sampleProfiles()
	profiles[0].Label = "prod\x1b]8;;https://evil.example\x07-ro\x1b]8;;\x07"
	if err := e.store.Save(context.Background(), profiles); err != nil {
		t.Fatal(err)
	}
	view := StripANSI(e.model(t).View().Content)
	for _, bad := range []string{"evil.example", "\x07"} {
		if strings.Contains(view, bad) {
			t.Fatalf("picker leaked %q:\n%s", bad, view)
		}
	}
}
