package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestModeSwitchFlow(t *testing.T) {
	m := workspace(t)
	if m.session == nil || !m.session.ReadOnly() {
		t.Fatal("workspace did not start read-only")
	}
	if !strings.Contains(StripANSI(m.View().Content), "r/o") {
		t.Fatalf("r/o badge missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	if m.modal != modalMode {
		t.Fatalf("modal = %v", m.modal)
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{"Enable read-write mode", "Writes will run on prod-ro", "Enter enable"} {
		if !strings.Contains(view, want) {
			t.Fatalf("confirm missing %q:\n%s", want, view)
		}
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.session.ReadOnly() {
		t.Fatal("session stayed read-only")
	}
	view = StripANSI(m.View().Content)
	if !strings.Contains(view, "r/w") || !strings.Contains(view, "read-write mode on") {
		t.Fatalf("read-write state missing:\n%s", view)
	}

	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "ctrl+w")
	if cmd == nil {
		t.Fatal("switching back did not start")
	}
	m = runCmd(t, m, cmd)
	if !m.session.ReadOnly() {
		t.Fatal("session did not return to read-only")
	}
	if !strings.Contains(StripANSI(m.View().Content), "r/o") {
		t.Fatalf("r/o badge missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestGoldenModeReadWrite(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	assertGolden(t, "workspace-readwrite.golden", m.View().Content)
}

func TestRootLaunchWarningAtConnect(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	m.rootLaunch = true
	m = connectSelected(t, selectLabel(t, m, "prod-ro"))
	if !strings.Contains(StripANSI(m.View().Content), "running as root") {
		t.Fatalf("root warning missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	if !strings.Contains(StripANSI(m.View().Content), "running as root") {
		t.Fatalf("root switch warning missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestModeConfirmShowsPrivilegedWarning(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.mode = &fakeModeController{elevated: true}
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	if !m.elevated {
		t.Fatal("elevated account was not stored")
	}
	if !strings.Contains(StripANSI(m.View().Content), "privileged") {
		t.Fatalf("privileged tag missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "elevated rights") || !strings.Contains(view, "not a security boundary") {
		t.Fatalf("elevated warning missing:\n%s", view)
	}
	m, _ = press(t, m, "esc")
	if m.modal != modalNone || !m.session.ReadOnly() {
		t.Fatalf("cancel modal=%v readOnly=%v", m.modal, m.session.ReadOnly())
	}
}

func TestModeSwitchBlockedByResultsLease(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.query = &fakeQueryExecutor{pages: []domain.QueryPage{{
		Grid:    domain.Grid{Columns: []domain.Column{{Name: "id"}}, Rows: []domain.Row{{Cells: []domain.Cell{{Text: "1"}}}}},
		HasMore: true,
	}}}
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT 1")
	m, cmd := press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)
	if !m.session.ResultsOpen() {
		t.Fatal("results cursor is not open")
	}
	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "ctrl+w")
	if cmd != nil || m.modal == modalMode {
		t.Fatal("mode switch started while results were open")
	}
	if !strings.Contains(m.message, "results open") || !m.session.ReadOnly() {
		t.Fatalf("message=%q readOnly=%v", m.message, m.session.ReadOnly())
	}
	m, _ = press(t, m, "esc")
}

func TestMongoGuardOnlyNoticeAndSwitch(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := connectSelected(t, selectLabel(t, e.model(t), "analytics"))
	if !strings.Contains(StripANSI(m.View().Content), "Read-only is enforced by the tool") {
		t.Fatalf("guard-only notice missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	if m.modal != modalMode {
		t.Fatal("read-write confirmation did not open")
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.session.ReadOnly() || !strings.Contains(StripANSI(m.View().Content), "r/w") {
		t.Fatalf("Mongo mode did not switch:\n%s", StripANSI(m.View().Content))
	}
}

func TestTreeWOpensModeConfirmAndEAccepts(t *testing.T) {
	m := workspace(t)
	if m.session == nil || !m.session.ReadOnly() {
		t.Fatal("workspace did not start read-only")
	}
	m, _ = press(t, m, "w")
	if m.modal != modalMode {
		t.Fatalf("w modal = %v", m.modal)
	}
	m, cmd := press(t, m, "e")
	if m.modal != modalNone {
		t.Fatalf("e did not close the mode modal: %v", m.modal)
	}
	if cmd == nil {
		t.Fatal("e did not start the read-write switch")
	}
	m = runCmd(t, m, cmd)
	if m.session.ReadOnly() {
		t.Fatal("session stayed read-only")
	}
}
