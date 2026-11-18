package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

func TestGolden(t *testing.T) {
	e := newTestEnv(t)
	empty := e.model(t)
	assertGolden(t, "picker-empty.golden", empty.View().Content)

	seedStandard(t, e)
	m := e.model(t)
	assertGolden(t, "picker.golden", m.View().Content)

	m = selectLabel(t, m, "prod-ro")
	m = connectSelected(t, m)
	assertGolden(t, "workspace.golden", m.View().Content)

	m, _ = press(t, m, "2")
	assertGolden(t, "workspace-details.golden", m.View().Content)

	m, _ = press(t, m, "3")
	assertGolden(t, "workspace-privileges.golden", m.View().Content)

	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-query-focus.golden", m.View().Content)

	m = selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-main-focus.golden", m.View().Content)
}

func TestGoldenModalsAndTooSmall(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)

	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1.internal:5432/orders")
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "prod-ro")
	assertGolden(t, "new-connection.golden", m.View().Content)
	assertNoSecrets(t, m.View().Content, "pgsecret", "mongosecret")

	m, _ = press(t, m, "esc")
	m = selectLabel(t, e.model(t), "billing")
	m, _ = press(t, m, "enter")
	assertGolden(t, "password.golden", m.View().Content)

	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "up")
	m, _ = press(t, m, "i")
	assertGolden(t, "import.golden", m.View().Content)

	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db9.internal:5432/orders")
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "prod-ro2")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalNew || !m.newNeedsPassword {
		t.Fatalf("expected inline password field, modal=%v needs=%v err=%q", m.modal, m.newNeedsPassword, m.newErr)
	}
	m = typeString(t, m, "hunter2")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave {
		t.Fatalf("expected save prompt, modal=%v err=%q", m.modal, m.newErr)
	}
	assertGolden(t, "save-profile.golden", m.View().Content)
	assertNoSecrets(t, m.View().Content, "hunter2", "pgsecret")

	next, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	m = next.(Model)
	assertGolden(t, "too-small.golden", m.View().Content)

	narrow, _ := e.model(t).Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	assertGolden(t, "too-small-narrow.golden", narrow.(Model).View().Content)
}

func TestGoldenStorageUnreadable(t *testing.T) {
	e := newTestEnv(t)
	if err := os.MkdirAll(filepath.Dir(e.store.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.store.Path(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "picker-storage-error.golden", e.model(t).View().Content)
}

func TestGoldenWide(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(Model)
	m = selectLabel(t, m, "prod-ro")
	m = connectSelected(t, m)
	assertGolden(t, "workspace-wide.golden", m.View().Content)
}

func TestTooSmallRecovers(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	m = next.(Model)
	if !strings.Contains(m.View().Content, "Terminal too small") {
		t.Fatal("expected too-small state")
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	if strings.Contains(m.View().Content, "Terminal too small") {
		t.Fatal("expected recovery at 80x24")
	}
}

func TestTooSmallMessageIsOneLine(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = next.(Model)
	view := m.View().Content
	if strings.Contains(view, "\n") {
		t.Fatalf("too-small view must be one line, got %q", view)
	}
	if lipgloss.Width(view) > m.width {
		t.Fatalf("too-small view width = %d, want <= %d", lipgloss.Width(view), m.width)
	}
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("too-small view lost its message: %q", view)
	}
}

func TestTooSmallKeyGuard(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	m = next.(Model)

	m, cmd := press(t, m, "enter")
	if cmd != nil || m.screen != ScreenPicker || m.modal != modalNone {
		t.Fatal("non-quit keys must be inert below the 80x24 floor")
	}
	m, _ = press(t, m, "down")
	if m.pickerSel != 0 {
		t.Fatalf("picker selection moved while too small: %d", m.pickerSel)
	}
	_, cmd = press(t, m, "q")
	if cmd == nil {
		t.Fatal("q must still quit while too small")
	}
}

func TestPickerEmptyState(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	view := m.View().Content
	for _, want := range []string{"No profiles yet", "Press n", "i to import", "~/.pgpass"} {
		if !strings.Contains(view, want) {
			t.Fatalf("empty picker missing %q:\n%s", want, view)
		}
	}
}

func TestPickerNavigationAndProfileSwitch(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	// profiles sort by label: analytics, billing, prod-ro.
	m = selectLabel(t, m, "billing")
	if p, _ := m.currentPickerProfile(); p.Label != "billing" {
		t.Fatalf("selected = %q", p.Label)
	}
	m, _ = press(t, m, "up")
	m, _ = press(t, m, "up")
	m, _ = press(t, m, "up")
	if m.pickerSel != 0 {
		t.Fatalf("up clamp = %d, want 0", m.pickerSel)
	}
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	if m.pickerSel != len(m.profiles)-1 {
		t.Fatalf("down clamp = %d, want %d", m.pickerSel, len(m.profiles)-1)
	}
	m, _ = press(t, m, "up")
	m = connectSelected(t, m)
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "billing") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	if !strings.Contains(StripANSI(m.viewQuery(80)), "SQL") {
		t.Fatalf("MySQL query prompt = %q", StripANSI(m.viewQuery(80)))
	}
}

func TestMongoProfileUsesMQL(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "analytics")
	m = connectSelected(t, m)
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "analytics") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	if !strings.Contains(StripANSI(m.viewQuery(80)), "MQL") {
		t.Fatalf("query prompt = %q", StripANSI(m.viewQuery(80)))
	}
}

func TestCyrillicAliasesOpenFlows(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "т")
	if m.modal != modalNew {
		t.Fatalf("т did not open new connection: %v", m.modal)
	}
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "ш")
	if m.modal != modalImport {
		t.Fatalf("ш did not open import: %v", m.modal)
	}
}

func TestMoreAndHelpMessages(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "m")
	if m.message != "more — story 4" {
		t.Fatalf("m message = %q", m.message)
	}
	m, _ = press(t, m, "?")
	if m.message != "help — story 9" {
		t.Fatalf("? message = %q", m.message)
	}
}

func TestCyrillicThroughUpdate(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	next, _ := m.Update(tea.KeyPressMsg{Text: "т", Code: 'т'})
	m = next.(Model)
	if m.modal != modalNew {
		t.Fatalf("cyrillic key press was not routed: %v", m.modal)
	}
}

func TestChordStubs(t *testing.T) {
	m := workspace(t)

	m, _ = press(t, m, "ctrl+ч")
	if !m.pendingChord {
		t.Fatal("ctrl+ч should start a chord")
	}
	m, _ = press(t, m, "ы")
	if m.pendingChord {
		t.Fatal("chord should resolve")
	}
	if m.message != "not available in this story" {
		t.Fatalf("chord message = %q", m.message)
	}

	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "?")
	if m.message != "cancelled" {
		t.Fatalf("unknown final = %q, want cancelled", m.message)
	}
}

func TestChordExecute(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+x")
	if m.message != "execute — story 5" {
		t.Fatalf("Ctrl+X Ctrl+X message = %q", m.message)
	}

	for _, final := range []string{"ц", "у"} {
		m, _ = press(t, m, "ctrl+ч")
		m, _ = press(t, m, final)
		if m.pendingChord {
			t.Fatalf("chord %q did not resolve", final)
		}
		if m.message != "not available in this story" {
			t.Fatalf("chord %q message = %q", final, m.message)
		}
	}
}

func TestChordTimeout(t *testing.T) {
	m := workspace(t)
	m, cmd := press(t, m, "ctrl+x")
	if !m.pendingChord || cmd == nil {
		t.Fatal("Ctrl+X must arm a chord with a timeout command")
	}
	if !strings.Contains(StripANSI(m.viewBottom(80)), "Ctrl+X …") {
		t.Fatal("pending chord must be visible in the bottom line")
	}
	seq := m.chordSeq
	next, _ := m.Update(chordTimeoutMsg{seq: seq})
	m = next.(Model)
	if m.pendingChord {
		t.Fatal("chord should expire")
	}
	if m.message != "cancelled" {
		t.Fatalf("timeout message = %q", m.message)
	}

	m, _ = press(t, m, "ctrl+x")
	stale := chordTimeoutMsg{seq: m.chordSeq - 1}
	next, _ = m.Update(stale)
	m = next.(Model)
	if !m.pendingChord {
		t.Fatal("stale timeout must not cancel a new chord")
	}
}

func TestTabsFocusDisconnect(t *testing.T) {
	m := workspace(t)

	m, _ = press(t, m, "2")
	if m.tab != 1 {
		t.Fatalf("tab = %d, want 1", m.tab)
	}
	m, _ = press(t, m, "]")
	if m.tab != 2 {
		t.Fatalf("tab = %d, want 2", m.tab)
	}
	m, _ = press(t, m, "tab")
	if m.focus != FocusMain {
		t.Fatalf("focus = %v, want main", m.focus)
	}
	m, _ = press(t, m, "ctrl+d")
	if m.screen != ScreenPicker {
		t.Fatalf("screen = %v, want picker", m.screen)
	}
	_, cmd := press(t, m, "q")
	if cmd == nil {
		t.Fatal("q at picker should quit")
	}
}

func TestWorkspaceQuitDoesNotExit(t *testing.T) {
	m := workspace(t)
	m, cmd := press(t, m, "q")
	if cmd != nil {
		t.Fatal("q in the workspace must not quit")
	}
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %v, want workspace", m.screen)
	}
	if m.message != "q quits only at the picker" {
		t.Fatalf("message = %q", m.message)
	}
}

func TestTabSelection(t *testing.T) {
	m := workspace(t)
	cases := []struct {
		key  string
		want int
	}{
		{"3", 2},
		{"[", 1},
		{"[", 0},
		{"[", 2},
		{"]", 0},
		{"2", 1},
		{"1", 0},
	}
	for _, tc := range cases {
		m, _ = press(t, m, tc.key)
		if m.tab != tc.want {
			t.Fatalf("after %q tab = %d, want %d", tc.key, m.tab, tc.want)
		}
	}
}

func TestFocusCycle(t *testing.T) {
	m := workspace(t)
	tabOrder := []Focus{FocusMain, FocusQuery, FocusTree, FocusMain}
	for _, want := range tabOrder {
		m, _ = press(t, m, "tab")
		if m.focus != want {
			t.Fatalf("tab focus = %v, want %v", m.focus, want)
		}
	}
	for _, want := range []Focus{FocusTree, FocusQuery, FocusMain, FocusTree} {
		m, _ = press(t, m, "shift+tab")
		if m.focus != want {
			t.Fatalf("shift+tab focus = %v, want %v", m.focus, want)
		}
	}
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	if !strings.Contains(StripANSI(m.View().Content), "Ctrl+X chord") {
		t.Fatal("query focus must be visible in the bottom line")
	}
}

func TestTreeNavigationBounds(t *testing.T) {
	m := workspace(t)
	if m.treeSel != 2 {
		t.Fatalf("initial tree selection = %d, want 2", m.treeSel)
	}

	m, _ = press(t, m, "tab")
	before := m.treeSel
	m, _ = press(t, m, "down")
	if m.treeSel != before {
		t.Fatal("tree must not move while another pane is focused")
	}

	m, _ = press(t, m, "shift+tab")
	if m.focus != FocusTree {
		t.Fatalf("focus = %v, want tree", m.focus)
	}
	for i := 0; i < 20; i++ {
		m, _ = press(t, m, "down")
	}
	if m.treeSel != len(fakeTree)-1 {
		t.Fatalf("down clamp = %d, want %d", m.treeSel, len(fakeTree)-1)
	}
	for i := 0; i < 20; i++ {
		m, _ = press(t, m, "up")
	}
	if m.treeSel != 0 {
		t.Fatalf("up clamp = %d, want 0", m.treeSel)
	}
}

func TestMessageClearsOnNextKey(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "?")
	if m.message == "" {
		t.Fatal("? should set a message")
	}
	m, _ = press(t, m, "x")
	if m.message != "" {
		t.Fatalf("message did not clear: %q", m.message)
	}
}

func TestReconnectResetsWorkspaceState(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "analytics")
	m = connectSelected(t, m)
	m, _ = press(t, m, "3")
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "down")

	m, _ = press(t, m, "ctrl+d")
	if m.screen != ScreenPicker {
		t.Fatal("expected picker")
	}
	if p, _ := m.currentPickerProfile(); p.Label != "analytics" {
		t.Fatalf("picker selection = %q, want disconnected profile", p.Label)
	}

	m = connectSelected(t, m)
	if m.tab != 0 || m.treeSel != 2 || m.focus != FocusTree {
		t.Fatalf("workspace not reset: tab=%d tree=%d focus=%v", m.tab, m.treeSel, m.focus)
	}
}

func TestRawSelectionAndFocusStyles(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	picker := m.View().Content
	first := m.profiles[m.pickerSel]
	selectedProfile := m.styles.Select.Render(Pad(profileFields(first, false), 80))
	if !strings.Contains(picker, selectedProfile) {
		t.Fatal("selected picker row lacks the Select style")
	}

	m = selectLabel(t, m, "prod-ro")
	m = connectSelected(t, m)
	view := m.View().Content
	if !strings.Contains(view, m.styles.Focused.Render("┌")) {
		t.Fatal("focused tree frame lacks the Focused style")
	}
	if !strings.Contains(view, m.styles.Focused.Render(" objects 3 · preview ")) {
		t.Fatal("focused tree title lacks the Focused style")
	}
	activeTab := m.styles.Accent.Bold(true).Underline(true).Render("1 Data")
	if !strings.Contains(view, activeTab) {
		t.Fatal("active tab lacks bold/underline accent styling")
	}
	selectedTree := m.styles.Select.Render(Pad(fakeTree[2].line(), 24))
	if !strings.Contains(view, selectedTree) {
		t.Fatal("selected tree row lacks the Select style")
	}
}

func TestUpdateRoutesKeyPress(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	next, _ := m.Update(tea.KeyPressMsg{Text: "n", Code: 'n'})
	m = next.(Model)
	if m.modal != modalNew {
		t.Fatalf("key press was not routed: %v", m.modal)
	}
}

func TestAltScreenAndBackground(t *testing.T) {
	e := newTestEnv(t)
	v := e.model(t).View()
	if !v.AltScreen {
		t.Fatal("shell must use the alternate screen")
	}
	if v.BackgroundColor == nil {
		t.Fatal("shell must set the Nightshift base background")
	}
}

func TestStorageUnreadableKeepsPickerUsable(t *testing.T) {
	e := newTestEnv(t)
	if err := os.MkdirAll(filepath.Dir(e.store.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.store.Path(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := e.model(t)
	if !m.storageErr {
		t.Fatal("storage error not recorded")
	}
	view := m.View().Content
	if !strings.Contains(view, "Profile storage is unreadable. Showing no profiles.") {
		t.Fatalf("missing storage copy:\n%s", view)
	}
	m, _ = press(t, m, "n")
	if m.modal != modalNew {
		t.Fatal("n must still open the new-connection modal")
	}
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "i")
	if m.modal != modalImport {
		t.Fatal("i must still open the import modal")
	}
}

func TestImportFlowCreatesProfiles(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "db1.internal:5432:orders:app_ro:pgsecret\n", 0o600)
	e.write(t, ".mongo.analyst", "mongodb://analyst:mongosecret@mongo1:27017/analytics\n", 0o600)
	m := e.model(t)
	m, _ = press(t, m, "i")
	if m.modal != modalImport {
		t.Fatal("import modal not open")
	}
	if len(m.importEntries) < 2 {
		t.Fatalf("import entries = %+v", m.importEntries)
	}
	// pgpass entries are selected by default; mongo files are not.
	for _, entry := range m.importEntries {
		if entry.Kind == "pgpass" && entry.Selectable && !m.importSel[entry.ID] {
			t.Fatalf("pgpass entry not selected by default: %+v", entry)
		}
		if entry.Kind == "mongo-file" && m.importSel[entry.ID] {
			t.Fatalf("mongo entry selected by default: %+v", entry)
		}
	}
	m, _ = press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatalf("import did not close: %v notice=%q", m.modal, m.importNotice)
	}
	views, err := e.svc.Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 {
		t.Fatalf("profiles = %+v", views)
	}
	if !strings.Contains(m.message, "imported") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestImportModalSpaceTogglesMongo(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)
	m, _ = press(t, m, "i")
	// Move to the mongo entry.
	for i := 0; i < len(m.importEntries); i++ {
		entry, _ := m.currentImportEntry()
		if entry.Kind == "mongo-file" {
			break
		}
		m, _ = press(t, m, "down")
	}
	entry, _ := m.currentImportEntry()
	if entry.Kind != "mongo-file" {
		t.Fatalf("cursor on %+v", entry)
	}
	m, _ = press(t, m, "space")
	if !m.importSel[entry.ID] {
		t.Fatal("space did not select the mongo entry")
	}
	m, _ = press(t, m, "space")
	if m.importSel[entry.ID] {
		t.Fatal("space did not deselect the mongo entry")
	}
}

func TestPasswordPromptNeverPersistsSecret(t *testing.T) {
	e := newTestEnv(t)
	p := domain.Profile{
		Label: "billing", Engine: domain.EngineMySQL, Host: "mysql1", Port: 3306,
		Database: "billing", User: "billing_ro",
	}
	e.seed(t, p)
	m := e.model(t)
	m, _ = press(t, m, "enter")
	if m.modal != modalPassword {
		t.Fatalf("password modal not open: %v", m.modal)
	}
	if !strings.Contains(StripANSI(m.View().Content), "Password for billing") {
		t.Fatalf("missing password title:\n%s", StripANSI(m.View().Content))
	}
	m = typeString(t, m, "hunter2")
	if strings.Contains(StripANSI(m.View().Content), "hunter2") {
		t.Fatal("password echoed in the modal")
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.screen != ScreenWorkspace {
		t.Fatalf("not connected: message=%q", m.message)
	}
	if e.connector.password != "hunter2" {
		t.Fatalf("password not passed: %q", e.connector.password)
	}
	views, _ := e.svc.Profiles(context.Background())
	data, err := os.ReadFile(e.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Fatal("secret persisted to profiles.json")
	}
	if len(views) != 1 {
		t.Fatalf("profiles = %+v", views)
	}
	assertNoSecrets(t, m.View().Content, "hunter2")

	// Each connect prompts again; nothing was stored from the first attempt.
	m, _ = press(t, m, "ctrl+d")
	m, _ = press(t, m, "enter")
	if m.modal != modalPassword {
		t.Fatalf("reconnect did not prompt again: %v", m.modal)
	}
}

func TestWrongPasswordShowsRowErrorThenPromptsAgain(t *testing.T) {
	e := newTestEnv(t)
	p := domain.Profile{
		Label: "billing", Engine: domain.EngineMySQL, Host: "mysql1", Port: 3306,
		Database: "billing", User: "billing_ro",
	}
	e.seed(t, p)
	e.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed"}
	m := e.model(t)
	m, _ = press(t, m, "enter")
	m = typeString(t, m, "wrong")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalNone || m.screen != ScreenPicker {
		t.Fatalf("wrong password did not return to picker: modal=%v screen=%v", m.modal, m.screen)
	}
	key := m.profiles[0].Profile.Key()
	if !strings.Contains(m.rowErrors[key], "authentication failed") {
		t.Fatalf("row error = %q", m.rowErrors[key])
	}
	if !strings.Contains(StripANSI(m.View().Content), "authentication failed") {
		t.Fatal("row error not rendered")
	}
	m, _ = press(t, m, "enter")
	if m.modal != modalPassword {
		t.Fatalf("retry did not prompt again: %v", m.modal)
	}
	e.connector.err = nil
	m = typeString(t, m, "right")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.screen != ScreenWorkspace {
		t.Fatalf("retry failed: %q", m.message)
	}
}

func TestConnectErrorShowsRowErrorAndRetry(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed"}
	m := selectLabel(t, e.model(t), "prod-ro")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.screen != ScreenPicker {
		t.Fatal("failure must keep the picker")
	}
	key := m.profiles[m.pickerSel].Profile.Key()
	rowErr := m.rowErrors[key]
	if !strings.Contains(rowErr, "prod-ro") || !strings.Contains(rowErr, "~/.pgpass") {
		t.Fatalf("row error = %q", rowErr)
	}
	if !strings.Contains(StripANSI(m.View().Content), "error") {
		t.Fatal("row error not rendered")
	}
	assertNoSecrets(t, m.View().Content, "pgsecret")

	// Enter retries: the row error is cleared before the attempt.
	e.connector.err = nil
	m, cmd = press(t, m, "enter")
	if m.rowErrors[key] != "" {
		t.Fatal("retry did not clear the row error")
	}
	m = runCmd(t, m, cmd)
	if m.screen != ScreenWorkspace {
		t.Fatalf("retry failed: %q", m.message)
	}
}

func TestConnectCancelClosesStaleSession(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m, cmd := press(t, m, "enter")
	if !m.connecting {
		t.Fatal("not connecting")
	}
	if !strings.Contains(StripANSI(m.View().Content), "connecting…") {
		t.Fatal("connecting row not rendered")
	}
	m, _ = press(t, m, "n")
	if m.modal != modalNone {
		t.Fatal("new connection must not open while connecting")
	}
	m, _ = press(t, m, "i")
	if m.modal != modalNone {
		t.Fatal("import must not open while connecting")
	}
	m, _ = press(t, m, "esc")
	if m.connecting {
		t.Fatal("Esc did not cancel")
	}
	if m.message != "connection cancelled" {
		t.Fatalf("message = %q", m.message)
	}
	if strings.Contains(StripANSI(m.View().Content), "connecting…") {
		t.Fatal("connecting state not restored after Esc")
	}
	m = runCmd(t, m, cmd)
	if len(e.connector.conns) != 1 || !e.connector.conns[0].closed {
		t.Fatal("stale successful connection was not closed")
	}
	if m.screen != ScreenPicker {
		t.Fatal("cancel must keep the picker")
	}
}

func TestNewConnectionRejectsPasswordURI(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app:secret@db1:5432/orders")
	m, _ = press(t, m, "enter")
	if m.newErr != "Could not connect: the URI must not contain a password." {
		t.Fatalf("error = %q", m.newErr)
	}
	if m.modal != modalNew {
		t.Fatal("form must stay open")
	}
	assertNoSecrets(t, m.newErr, "secret")
	assertNoSecrets(t, StripANSI(m.View().Content), "secret")
}

func TestNewConnectionPromptThenSave(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalNew || !m.newNeedsPassword {
		t.Fatalf("expected password field, modal=%v needs=%v err=%q", m.modal, m.newNeedsPassword, m.newErr)
	}
	if p, _ := m.svc.PrepareNewConnection(context.Background(), m.newURI.Value(), m.newLabel.Value()); p.Label != "orders@db1" {
		t.Fatalf("default label = %q", p.Label)
	}
	m = typeString(t, m, "hunter2")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave {
		t.Fatalf("save prompt missing: modal=%v err=%q", m.modal, m.newErr)
	}
	if m.saveProfile.Label != "orders@db1" {
		t.Fatalf("save label = %q", m.saveProfile.Label)
	}
	m, _ = press(t, m, "enter")
	if m.screen != ScreenWorkspace {
		t.Fatalf("save did not open workspace: %q", m.message)
	}
	lines, err := os.ReadFile(e.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(lines), "hunter2") {
		t.Fatal("save prompt persisted the secret")
	}
	var doc struct {
		Profiles []map[string]any `json:"profiles"`
	}
	if err := json.Unmarshal(lines, &doc); err != nil {
		t.Fatal(err)
	}
	for _, p := range doc.Profiles {
		for key := range p {
			if strings.Contains(strings.ToLower(key), "pass") {
				t.Fatalf("profile stored a password field: %v", p)
			}
		}
	}
}

func TestNewConnectionMatchedSourceNeedsNoPrompt(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "db1:5432:orders:app_ro:pgsecret\n", 0o600)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave {
		t.Fatalf("expected save prompt without a password field: modal=%v needs=%v err=%q", m.modal, m.newNeedsPassword, m.newErr)
	}
	if m.saveProfile.Source == nil || m.saveProfile.Source.Kind != domain.SourcePgpass {
		t.Fatalf("matched source not attached: %+v", m.saveProfile.Source)
	}
	if e.connector.password != "pgsecret" {
		t.Fatalf("source password not used: %q", e.connector.password)
	}
	m, _ = press(t, m, "enter")
	if m.screen != ScreenWorkspace {
		t.Fatalf("save did not open workspace: %q", m.message)
	}
	data, err := os.ReadFile(e.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "pgsecret") {
		t.Fatal("matched source password persisted")
	}
}

func TestNewConnectionSkipSave(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	m = typeString(t, m, "hunter2")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave {
		t.Fatalf("modal = %v", m.modal)
	}
	m, _ = press(t, m, "esc")
	if m.screen != ScreenWorkspace {
		t.Fatalf("skip did not open workspace: %q", m.message)
	}
	views, _ := e.svc.Profiles(context.Background())
	if len(views) != 0 {
		t.Fatalf("skip saved a profile: %+v", views)
	}
}

func TestNewConnectionPasteIntoURI(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "n")
	next, _ := m.Update(tea.PasteMsg{Content: "postgres://app_ro@db1:5432/orders"})
	m = next.(Model)
	if m.newURI.Value() != "postgres://app_ro@db1:5432/orders" {
		t.Fatalf("pasted value = %q", m.newURI.Value())
	}
}

func TestSanitizesSourceDerivedText(t *testing.T) {
	e := newTestEnv(t)
	p := domain.Profile{
		Label: "prod\x1b[31m-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	e.seed(t, p)
	m := e.model(t)
	view := m.View().Content
	if strings.Contains(view, "\x1b[31m") {
		t.Fatal("injected ANSI sequence escaped into the picker")
	}
	if !strings.Contains(StripANSI(view), "prod-ro") {
		t.Fatalf("label not rendered: %q", StripANSI(view))
	}
}

func TestGoldensContainNoSecrets(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	secrets := []string{"pgsecret", "mongosecret", "hunter2", "sup3r", "password=", "user:pass"}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range secrets {
			if strings.Contains(strings.ToLower(string(data)), strings.ToLower(secret)) {
				t.Fatalf("%s contains secret %q", path, secret)
			}
		}
	}
}

func TestDisconnectClosesSession(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	if len(e.connector.conns) != 1 {
		t.Fatal("no connection opened")
	}
	m, _ = press(t, m, "ctrl+d")
	if !e.connector.conns[0].closed {
		t.Fatal("disconnect did not close the connection")
	}
}

func TestImportModalEmptyState(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "i")
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "No ~/.pgpass or ~/.mongo.* files found") {
		t.Fatalf("missing empty state:\n%s", view)
	}
}

func TestImportMalformedRowsCarryTags(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "malformed\n", 0o600)
	e.write(t, ".mongo.bad", "mongodb://mongo1:notaport/analytics\n", 0o600)
	m := e.model(t)
	m, _ = press(t, m, "i")
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "warning") {
		t.Fatalf("malformed pgpass has no warning tag:\n%s", view)
	}
	if !strings.Contains(view, "error") || !strings.Contains(view, "~/.mongo.bad") {
		t.Fatalf("malformed mongo has no error tag:\n%s", view)
	}
	m, _ = press(t, m, "enter")
	if m.modal != modalImport {
		t.Fatal("import modal must stay open when nothing selectable remains")
	}
	if !strings.Contains(m.importNotice, "skipped") {
		t.Fatalf("import notice did not report skipped rows: %q", m.importNotice)
	}
}

func TestStaleConnectResultClosesSession(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m, cmd := press(t, m, "enter")
	if !m.connecting || cmd == nil {
		t.Fatal("connection did not start")
	}
	done := cmd() // obtain a successful result without applying it
	m, _ = press(t, m, "esc")
	if m.connecting {
		t.Fatal("Esc did not cancel")
	}
	next, _ := m.Update(done)
	m = next.(Model)
	if m.screen != ScreenPicker {
		t.Fatalf("stale result opened workspace: %v", m.screen)
	}
	if len(e.connector.conns) != 1 || !e.connector.conns[0].closed {
		t.Fatal("stale successful session was not closed")
	}
}

func TestNewConnectionCancelClosesSession(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "db1:5432:orders:app_ro:pgsecret\n", 0o600)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	if !m.newBusy || cmd == nil {
		t.Fatalf("new connection did not start: busy=%v cmd=%v", m.newBusy, cmd)
	}
	done := cmd()
	m, _ = press(t, m, "esc")
	if m.modal != modalNone || m.connecting {
		t.Fatalf("cancel left modal=%v connecting=%v", m.modal, m.connecting)
	}
	next, _ := m.Update(done)
	m = next.(Model)
	if m.screen != ScreenPicker || m.modal != modalNone {
		t.Fatalf("stale new result opened %v/%v", m.screen, m.modal)
	}
	if len(e.connector.conns) != 1 || !e.connector.conns[0].closed {
		t.Fatal("stale new-connection session was not closed")
	}
}

func TestPasswordModalCancelClosesSession(t *testing.T) {
	e := newTestEnv(t)
	p := domain.Profile{
		Label: "billing", Engine: domain.EngineMySQL, Host: "mysql1", Port: 3306,
		Database: "billing", User: "billing_ro",
	}
	e.seed(t, p)
	m := e.model(t)
	m, _ = press(t, m, "enter")
	if m.modal != modalPassword {
		t.Fatalf("password modal = %v", m.modal)
	}
	m = typeString(t, m, "hunter2")
	m, cmd := press(t, m, "enter")
	if !m.passwordBusy || cmd == nil {
		t.Fatal("password connect did not start")
	}
	done := cmd()
	m, _ = press(t, m, "esc")
	next, _ := m.Update(done)
	m = next.(Model)
	if m.screen != ScreenPicker || m.modal != modalNone {
		t.Fatalf("stale password result opened %v/%v", m.screen, m.modal)
	}
	if len(e.connector.conns) != 1 || !e.connector.conns[0].closed {
		t.Fatal("stale password session was not closed")
	}
}

func TestPickerShowsSourceHealth(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "db1:5432:orders:app_ro:secret\n", 0o644)
	e.write(t, ".pgpass.good", "db9:5432:orders:app_ro:secret\n", 0o600)
	e.write(t, ".mongo.bad", "mongodb://mongo1:notaport/db\n", 0o600)
	e.seed(t,
		domain.Profile{Label: "insecure", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432, Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass"}},
		domain.Profile{Label: "missing", Engine: domain.EnginePostgreSQL, Host: "db2", Port: 5432, Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.missing"}},
		domain.Profile{Label: "stale", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432, Source: &domain.Source{Kind: domain.SourcePgpass, Path: "~/.pgpass.good"}},
		domain.Profile{Label: "malformed", Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017, Source: &domain.Source{Kind: domain.SourceMongoFile, Path: "~/.mongo.bad"}},
	)
	view := StripANSI(e.model(t).View().Content)
	for _, want := range []string{"ignored", "missing", "stale", "error"} {
		if !strings.Contains(view, want) {
			t.Fatalf("source health %q missing:\n%s", want, view)
		}
	}
}

func TestTooSmallKeepsPickerQuitRuleAndCyrillicForceQuit(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	m = next.(Model)
	m, cmd := press(t, m, "q")
	if cmd != nil {
		t.Fatal("q in a too-small workspace must not quit")
	}
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %v", m.screen)
	}
	_, cmd = press(t, m, "ctrl+с")
	if cmd == nil {
		t.Fatal("Cyrillic Ctrl+С must force quit while too small")
	}
}

func TestPickerFooterSurvivesErrorsAndScroll(t *testing.T) {
	e := newTestEnv(t)
	profiles := make([]domain.Profile, 0, 20)
	for i := 0; i < 20; i++ {
		profiles = append(profiles, domain.Profile{
			Label: fmt.Sprintf("p%02d", i), Engine: domain.EnginePostgreSQL,
			Host: "h", Port: 5432, Database: "d", User: "u",
		})
	}
	e.seed(t, profiles...)
	m := e.model(t)
	for i := 0; i < 10; i++ {
		m.rowErrors[m.profiles[i].Profile.Key()] = "connection failed"
	}
	view := m.View().Content
	if lines := strings.Split(view, "\n"); len(lines) != 24 {
		t.Fatalf("view height = %d, want 24", len(lines))
	}
	if !strings.Contains(StripANSI(view), "q quit") {
		t.Fatalf("footer was clipped:\n%s", view)
	}
}

func TestNewConnectionAuthFailureAllowsPasswordOverride(t *testing.T) {
	e := newTestEnv(t)
	e.write(t, ".pgpass", "db1:5432:orders:app_ro:wrong\n", 0o600)
	e.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed"}
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if !m.newNeedsPassword || !m.newIgnoreSource {
		t.Fatalf("auth failure did not offer an override: needs=%v ignore=%v err=%q", m.newNeedsPassword, m.newIgnoreSource, m.newErr)
	}
	e.connector.err = nil
	m = typeString(t, m, "right")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave || m.saveProfile.Source != nil {
		t.Fatalf("save prompt = %v source=%+v", m.modal, m.saveProfile.Source)
	}
	m, _ = press(t, m, "enter")
	if m.screen != ScreenWorkspace {
		t.Fatalf("save did not open workspace: %q", m.message)
	}
}

func TestNewConnectionURIEditClearsPassword(t *testing.T) {
	e := newTestEnv(t)
	m := e.model(t)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if !m.newNeedsPassword {
		t.Fatalf("password field missing: %v %q", m.modal, m.newErr)
	}
	m = typeString(t, m, "secret")
	m, _ = press(t, m, "shift+tab")
	m, _ = press(t, m, "shift+tab")
	m = typeString(t, m, "x")
	if m.newNeedsPassword || m.newIgnoreSource || m.newPassword.Value() != "" {
		t.Fatalf("URI edit kept the password: needs=%v ignore=%v value=%q", m.newNeedsPassword, m.newIgnoreSource, m.newPassword.Value())
	}
}

type failingSaveStore struct {
	inner ports.ProfileStore
}

func (f failingSaveStore) Load(ctx context.Context) ([]domain.Profile, error) {
	return f.inner.Load(ctx)
}

func (f failingSaveStore) Save(ctx context.Context, profiles []domain.Profile) error {
	return os.ErrPermission
}

func TestSaveFailureKeepsWorkspaceOpen(t *testing.T) {
	e := newTestEnv(t)
	svc := app.NewService(failingSaveStore{inner: e.store}, e.resolver, e.connector)
	m := New(svc)
	m, _ = press(t, m, "n")
	m = typeString(t, m, "postgres://app_ro@db1:5432/orders")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	m = typeString(t, m, "hunter2")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalSave {
		t.Fatalf("save prompt missing: %v %q", m.modal, m.newErr)
	}
	m, _ = press(t, m, "enter")
	if m.screen != ScreenWorkspace {
		t.Fatalf("save failure did not keep the workspace: %q", m.message)
	}
	if !strings.Contains(m.message, "Could not save") {
		t.Fatalf("message = %q", m.message)
	}
}
