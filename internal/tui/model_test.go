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
	m = workspaceFrom(t, m)
	assertGolden(t, "workspace.golden", m.View().Content)

	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	assertGolden(t, "workspace-details.golden", m.View().Content)

	m, cmd = press(t, m, "2")
	m = settle(t, m, cmd)
	assertGolden(t, "workspace-indexes.golden", m.View().Content)

	m, cmd = press(t, m, "3")
	m = settle(t, m, cmd)
	assertGolden(t, "workspace-privileges.golden", m.View().Content)

	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-query-focus.golden", m.View().Content)

	m = selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-main-focus.golden", m.View().Content)
}

func TestGoldenModalsAndTooSmall(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := e.model(t)

	edit := selectLabel(t, m, "prod-ro")
	edit, _ = press(t, edit, "e")
	assertGolden(t, "edit-profile.golden", edit.View().Content)

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
	m = workspaceFrom(t, m)
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
	if cmd != nil {
		t.Fatal("plain q must not quit")
	}
	_, cmd = press(t, m, "ctrl+q")
	if cmd == nil {
		t.Fatal("Ctrl+Q must still quit while too small")
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
	if m.message != "m fetches the next page in the Data tab" {
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
	if m.message != "counting exact rows… Esc cancels" {
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
	if m.message != "enter a statement before running it" {
		t.Fatalf("Ctrl+X Ctrl+X message = %q", m.message)
	}

	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ц")
	if m.pendingChord {
		t.Fatal("chord did not resolve")
	}
	if m.modal != modalMode {
		t.Fatalf("toggle mode modal = %v", m.modal)
	}
	m, _ = press(t, m, "esc")

	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "у")
	if m.modal != modalStatement {
		t.Fatalf("edit chord modal = %v", m.modal)
	}
	m, cmd := press(t, m, "esc")
	m = runCmd(t, m, cmd)
	if m.modal != modalNone {
		t.Fatalf("statement modal did not close: %v", m.modal)
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
	if m.details.tab != 1 {
		t.Fatalf("tab = %d, want 1", m.details.tab)
	}
	m, _ = press(t, m, "]")
	if m.details.tab != 2 {
		t.Fatalf("tab = %d, want 2", m.details.tab)
	}
	m, _ = press(t, m, "tab")
	if m.focus != FocusMain {
		t.Fatalf("focus = %v, want main", m.focus)
	}
	m, _ = press(t, m, "ctrl+d")
	if m.screen != ScreenPicker {
		t.Fatalf("screen = %v, want picker", m.screen)
	}
	_, cmd := press(t, m, "ctrl+q")
	if cmd == nil {
		t.Fatal("Ctrl+Q at picker should quit")
	}
}

func TestEditProfileModalSaves(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	original := m.profiles[m.pickerSel].Profile
	m, _ = press(t, m, "e")
	if m.modal != modalEdit || m.editOriginalKey != original.Key() {
		t.Fatalf("edit modal = %v key=%q", m.modal, m.editOriginalKey)
	}
	if m.editHost.Value() != "db1.internal" || m.editPort.Value() != "5432" {
		t.Fatalf("form not prefilled: host=%q port=%q", m.editHost.Value(), m.editPort.Value())
	}
	m.editLabel.SetValue("prod-ro-2")
	m.editHost.SetValue("db2.internal")
	m, _ = press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatalf("modal stayed open: %v err=%q", m.modal, m.editErr)
	}
	if !strings.Contains(m.message, "saved prod-ro-2") {
		t.Fatalf("message = %q", m.message)
	}
	loaded, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 {
		t.Fatalf("profiles = %d, want 3", len(loaded))
	}
	var got *domain.Profile
	for i := range loaded {
		if loaded[i].Label == "prod-ro-2" {
			got = &loaded[i]
		}
	}
	if got == nil || got.Host != "db2.internal" || got.Port != 5432 {
		t.Fatalf("updated profile = %+v", got)
	}
	if got.Source == nil || got.Source.Kind != domain.SourcePgpass {
		t.Fatalf("source lost: %+v", got.Source)
	}
	if p, ok := m.currentPickerProfile(); !ok || p.Label != "prod-ro-2" {
		t.Fatalf("selection did not follow the profile: %+v", p)
	}
}

func TestEditProfileCancelKeepsProfile(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m, _ = press(t, m, "e")
	m.editHost.SetValue("changed.internal")
	m, _ = press(t, m, "esc")
	if m.modal != modalNone {
		t.Fatalf("modal = %v", m.modal)
	}
	loaded, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range loaded {
		if p.Label == "prod-ro" && p.Host != "db1.internal" {
			t.Fatalf("cancelled edit changed the profile: %+v", p)
		}
	}
}

func TestEditProfileRejectsInvalidPort(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m, _ = press(t, m, "e")
	m.editPort.SetValue("abc")
	m, _ = press(t, m, "enter")
	if m.modal != modalEdit || m.editErr != "port must be a number" {
		t.Fatalf("modal=%v err=%q", m.modal, m.editErr)
	}
	if m.editFocus != editPort {
		t.Fatalf("focus = %v, want port", m.editFocus)
	}
}

func TestEditProfileTogglesAllowNoTLS(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m, _ = press(t, m, "e")
	for i := 0; i < 5; i++ {
		m, _ = press(t, m, "tab")
	}
	if m.editFocus != editTLS {
		t.Fatalf("focus = %v, want TLS", m.editFocus)
	}
	m, _ = press(t, m, "space")
	if !m.editAllowNoTLS {
		t.Fatal("space did not enable the no-TLS fallback")
	}
	m, _ = press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatalf("modal stayed open: %v err=%q", m.modal, m.editErr)
	}
	loaded, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range loaded {
		if p.Label == "prod-ro" {
			if !p.AllowNoTLS {
				t.Fatalf("allow_no_tls not saved: %+v", p)
			}
			return
		}
	}
	t.Fatal("prod-ro not found")
}

func TestEditProfileKeyOnlyAtPicker(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "e")
	if m.modal != modalNone {
		t.Fatalf("edit modal opened in the workspace: %v", m.modal)
	}
}

func TestWorkspaceCtrlQQuits(t *testing.T) {
	m := workspace(t)
	m, cmd := press(t, m, "q")
	if cmd != nil {
		t.Fatal("plain q in the workspace must not quit")
	}
	_, cmd = press(t, m, "ctrl+q")
	if cmd == nil {
		t.Fatal("Ctrl+Q in the workspace should quit")
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
		{"[", 3},
		{"]", 0},
		{"2", 1},
		{"1", 0},
	}
	for _, tc := range cases {
		m, _ = press(t, m, tc.key)
		if m.details.tab != tc.want {
			t.Fatalf("after %q tab = %d, want %d", tc.key, m.details.tab, tc.want)
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
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "Enter run · Ctrl+Space complete") {
		t.Fatal("query focus must be visible in the bottom line")
	}
	if !strings.Contains(view, "▸ SQL") {
		t.Fatalf("query bar focus marker missing:\n%s", view)
	}
}

func TestTreeNavigationBounds(t *testing.T) {
	m := workspace(t)
	if m.tree.selected != 2 {
		t.Fatalf("initial tree selection = %d, want 2", m.tree.selected)
	}

	m, _ = press(t, m, "tab")
	before := m.tree.selected
	m, _ = press(t, m, "down")
	if m.tree.selected != before {
		t.Fatal("tree must not move while another pane is focused")
	}

	m, _ = press(t, m, "shift+tab")
	if m.focus != FocusTree {
		t.Fatalf("focus = %v, want tree", m.focus)
	}
	for i := 0; i < 20; i++ {
		m, _ = press(t, m, "down")
	}
	if m.tree.selected != len(m.tree.rows)-1 {
		t.Fatalf("down clamp = %d, want %d", m.tree.selected, len(m.tree.rows)-1)
	}
	for i := 0; i < 20; i++ {
		m, _ = press(t, m, "up")
	}
	if m.tree.selected != 0 {
		t.Fatalf("up clamp = %d, want 0", m.tree.selected)
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
	if m.details.tab != 0 || m.focus != FocusTree || m.tree.selected != 0 {
		t.Fatalf("workspace not reset: tab=%d tree=%d focus=%v", m.details.tab, m.tree.selected, m.focus)
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
	m = workspaceFrom(t, m)
	view := m.View().Content
	if !strings.Contains(view, m.styles.Focused.Render("┌")) {
		t.Fatal("focused tree frame lacks the Focused style")
	}
	if !strings.Contains(view, m.styles.Focused.Render(fmt.Sprintf(" objects %d ", m.tree.objectCount()))) {
		t.Fatal("focused tree title lacks the Focused style")
	}
	activeTab := m.styles.Accent.Bold(true).Underline(true).Render("4 Data")
	if !strings.Contains(view, activeTab) {
		t.Fatal("active tab lacks bold/underline accent styling")
	}
	selected := ""
	for _, row := range m.tree.rows {
		if row.Key == m.tree.selectedKey() {
			selected = treeRowText(row)
			break
		}
	}
	selectedTree := m.styles.Select.Render(Pad(selected, 24))
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
	e.write(t, ".mypass", "mysql1.internal:3306:billing:billing_ro:mysecret\n", 0o600)
	e.write(t, ".mongo.analyst", "mongodb://analyst:mongosecret@mongo1:27017/analytics\n", 0o600)
	m := e.model(t)
	m, _ = press(t, m, "i")
	if m.modal != modalImport {
		t.Fatal("import modal not open")
	}
	if len(m.importEntries) < 3 {
		t.Fatalf("import entries = %+v", m.importEntries)
	}
	// Password-file entries are selected by default; mongo files are not.
	for _, entry := range m.importEntries {
		if (entry.Kind == "pgpass" || entry.Kind == "mypass") && entry.Selectable && !m.importSel[entry.ID] {
			t.Fatalf("%s entry not selected by default: %+v", entry.Kind, entry)
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
	if len(views) != 2 {
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
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.screen != ScreenWorkspace || len(m.tree.roots) == 0 {
		t.Fatalf("save did not load workspace: %q roots=%v", m.message, m.tree.roots)
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
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.screen != ScreenWorkspace || len(m.tree.roots) == 0 {
		t.Fatalf("save did not load workspace: %q roots=%v", m.message, m.tree.roots)
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
	m, cmd = press(t, m, "esc")
	m = settle(t, m, cmd)
	if m.screen != ScreenWorkspace || len(m.tree.roots) == 0 {
		t.Fatalf("skip did not load workspace: %q roots=%v", m.message, m.tree.roots)
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
	if !strings.Contains(view, "No ~/.pgpass, ~/.mypass, or ~/.mongo.* files found") {
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

func TestTooSmallKeepsCtrlQAndCyrillicForceQuit(t *testing.T) {
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
	_, cmd = press(t, m, "ctrl+q")
	if cmd == nil {
		t.Fatal("Ctrl+Q must quit while too small")
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
	if !strings.Contains(StripANSI(view), "Ctrl+Q quit") {
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
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.screen != ScreenWorkspace || len(m.tree.roots) == 0 {
		t.Fatalf("save failure did not load workspace: %q roots=%v", m.message, m.tree.roots)
	}
	if !strings.Contains(m.message, "Could not save") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestTreeAccessMarkers(t *testing.T) {
	m := workspace(t)
	view := StripANSI(m.View().Content)
	for _, want := range []string{"customers  ro", "invoices  rw", "audit_log  —", "customer_view  ro"} {
		if !strings.Contains(view, want) {
			t.Fatalf("marker %q missing:\n%s", want, view)
		}
	}
	for _, row := range m.tree.rows {
		if row.Info == "" && row.Node.Kind == domain.KindSequence {
			if strings.Contains(treeRowText(row), "—") {
				t.Fatalf("sequence carries a marker: %q", treeRowText(row))
			}
		}
	}
}

func TestTreeAccessMarkerIsDimmed(t *testing.T) {
	m := workspace(t)
	raw := m.View().Content
	if !strings.Contains(raw, m.styles.Muted.Render("  rw")) {
		t.Fatalf("access marker is not rendered with the muted style:\n%q", raw)
	}
	if !strings.Contains(raw, m.styles.Text.Render("    · invoices")) {
		t.Fatalf("object name is not rendered with the text style:\n%q", raw)
	}
}

func TestDetailsStatisticsAndExactCount(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	for _, want := range []string{"Statistics", "~5 (catalog estimate)", "8.0 KB", "Ctrl+X s computes exact count", "Structure"} {
		if !strings.Contains(view, want) {
			t.Fatalf("statistics %q missing:\n%s", want, view)
		}
	}
	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "s")
	m = settle(t, m, cmd)
	view = StripANSI(m.View().Content)
	if !strings.Contains(view, "5 exact") {
		t.Fatalf("exact count missing:\n%s", view)
	}
	if m.message != "exact count 5" {
		t.Fatalf("message = %q", m.message)
	}
}

func TestExactCountCancelKeepsEstimate(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "s")
	if !m.details.exactLoading {
		t.Fatal("exact count did not start")
	}
	m, _ = press(t, m, "esc")
	if m.details.exactLoading || m.details.exactSet {
		t.Fatalf("cancel loading=%v set=%v", m.details.exactLoading, m.details.exactSet)
	}
	if m.message != "cancelled" {
		t.Fatalf("message = %q", m.message)
	}
	if !strings.Contains(StripANSI(m.View().Content), "~5 (catalog estimate)") {
		t.Fatal("estimate did not survive cancel")
	}
}

func TestExactCountFailureKeepsEstimate(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "s")
	key := m.details.node.Ref.Key()
	next, _ := m.Update(exactCountMsg{
		gen: m.exactGen, key: key,
		err: domain.NewError(domain.KindEngine, "count", "count failed"),
	})
	m = next.(Model)
	if m.details.exactLoading || m.details.exactSet {
		t.Fatalf("failure loading=%v set=%v", m.details.exactLoading, m.details.exactSet)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "count failed") || !strings.Contains(view, "~5 (catalog estimate)") {
		t.Fatalf("failure view:\n%s", view)
	}
}

func TestStatisticsErrorKeepsStructure(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	defaults := defaultFakeStats(domain.EnginePostgreSQL)
	e.connector.stats = &fakeStatsReader{
		statisticsErr: domain.NewError(domain.KindPermission, "statistics", "permission denied"),
		structure:     defaults.structure,
		indexes:       defaults.indexes,
	}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "error: permission denied") || !strings.Contains(view, "Structure") || !strings.Contains(view, "bigint") {
		t.Fatalf("statistics error hid structure:\n%s", view)
	}
}

func TestPrivilegesTabForSecurity(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "2")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	for _, want := range []string{"principal", "app_ro", "grants", "memberships", "attributes"} {
		if !strings.Contains(view, want) {
			t.Fatalf("security privileges %q missing:\n%s", want, view)
		}
	}
	if m.details.currentTab() != "Privileges" {
		t.Fatalf("tab = %q", m.details.currentTab())
	}
}

func TestPrivilegesTabForUser(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindUsers)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindUser)
	m, cmd = press(t, m, "2")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "principal") || !strings.Contains(view, "app_ro") {
		t.Fatalf("user privileges missing:\n%s", view)
	}
	if m.details.currentTab() != "Privileges" {
		t.Fatalf("tab = %q", m.details.currentTab())
	}
}

func TestPrivilegesTabError(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.privileges = &fakePrivilegeReader{
		err: domain.NewError(domain.KindPermission, "privileges", "permission denied"),
	}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	m, cmd := press(t, m, "3")
	m = settle(t, m, cmd)
	if !strings.Contains(StripANSI(m.View().Content), "error: permission denied") {
		t.Fatalf("privileges error missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestMongoDetailsStructureAndIndexes(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "analytics", nil)
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	for _, want := range []string{"sampled from 2 documents", "profile.email", "1/2 (50%)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("mongo structure %q missing:\n%s", want, view)
		}
	}
	m, cmd = press(t, m, "2")
	m = settle(t, m, cmd)
	view = StripANSI(m.View().Content)
	for _, want := range []string{"_id_", "btree", "8.0 KB", "Statistics", "Structure"} {
		if !strings.Contains(view, want) {
			t.Fatalf("mongo indexes %q missing:\n%s", want, view)
		}
	}
}

func TestSecurityLoadsPrincipals(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindUsers)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	foundUser := false
	for _, row := range m.tree.rows {
		if row.Info == "" && row.Node.Kind == domain.KindUser && row.Node.Name == "app_ro" {
			foundUser = true
		}
	}
	if !foundUser {
		t.Fatalf("user row missing:\n%s", StripANSI(m.View().Content))
	}
	m = selectKind(t, m, domain.KindUser)
	if view := StripANSI(m.View().Content); !strings.Contains(view, "scope") || !strings.Contains(view, "LOGIN") {
		t.Fatalf("user details missing scope/attributes:\n%s", view)
	}
	m = selectKind(t, m, domain.KindRoles)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	foundRole := false
	for _, row := range m.tree.rows {
		if row.Info == "" && row.Node.Kind == domain.KindRole && row.Node.Name == "reporting" {
			foundRole = true
		}
	}
	if !foundRole {
		t.Fatalf("role row missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestTreeLazyErrorRetries(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.catalog = &fakeCatalog{engine: domain.EnginePostgreSQL, childrenErrAll: true}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindDatabase)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if !strings.Contains(StripANSI(m.View().Content), "not permitted") {
		t.Fatalf("permission error missing:\n%s", StripANSI(m.View().Content))
	}
	e.connector.catalog.childrenErrAll = false
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	if !strings.Contains(StripANSI(m.View().Content), "public") {
		t.Fatalf("retry did not load children:\n%s", StripANSI(m.View().Content))
	}
}

func TestPostgresSchemasGrouping(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindDatabase)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	public, group := -1, -1
	for i, row := range m.tree.rows {
		if row.Info != "" {
			continue
		}
		if row.Node.Kind == domain.KindSchema && row.Node.Name == "public" {
			public = i
		}
		if row.Node.Kind == domain.KindSchemas {
			group = i
		}
	}
	if public < 0 || group < 0 || public > group {
		t.Fatalf("schemas not grouped after public: public=%d group=%d\n%s", public, group, StripANSI(m.View().Content))
	}
	m = selectKind(t, m, domain.KindSchemas)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "reporting") {
		t.Fatalf("schema group did not load children:\n%s", view)
	}
}

func TestPostgresDatabaseSwitch(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.write(t, ".pgpass", "db1.internal:5432:orders:app_ro:pgsecret\ndb1.internal:5432:billing:app_ro:billingpass\n", 0o600)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m, _ = press(t, m, "down") // billing
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if got := m.session.CurrentDatabase(); got != "billing" {
		t.Fatalf("database = %q", got)
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "billing") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	if len(e.connector.conns) != 2 || !e.connector.conns[0].closed {
		t.Fatalf("expected a switched connection: %+v", e.connector.conns)
	}
}

func TestPostgresSwitchFailureReturnsToPicker(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.write(t, ".pgpass", "db1.internal:5432:orders:app_ro:pgsecret\ndb1.internal:5432:billing:app_ro:billingpass\n", 0o600)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	e.connector.err = &domain.Error{Kind: domain.KindAuth, Message: "authentication failed"}
	m, _ = press(t, m, "down")
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if m.screen != ScreenPicker {
		t.Fatalf("screen = %v, want picker", m.screen)
	}
	if !strings.Contains(m.message, "authentication failed") {
		t.Fatalf("message = %q", m.message)
	}
	p, ok := m.currentPickerProfile()
	if !ok || p.Label != "prod-ro" {
		t.Fatalf("selected picker profile = %+v", p)
	}
	if !strings.Contains(m.rowErrors[p.Key()], "authentication failed") {
		t.Fatalf("row error = %q", m.rowErrors[p.Key()])
	}
	if !strings.Contains(StripANSI(m.View().Content), "authentication failed") {
		t.Fatal("picker row error not rendered")
	}
}

func TestMySQLAndMongoTrees(t *testing.T) {
	for _, tc := range []struct {
		label  string
		engine domain.Engine
	}{
		{"billing", domain.EngineMySQL},
		{"analytics", domain.EngineMongoDB},
	} {
		e := newTestEnv(t)
		seedStandard(t, e)
		m := selectLabel(t, e.model(t), tc.label)
		m = connectSelected(t, m)
		m = selectKind(t, m, domain.KindDatabase)
		m, cmd := press(t, m, "right")
		m = settle(t, m, cmd)
		view := StripANSI(m.View().Content)
		for _, want := range []string{"customers", "invoices", "customer_view"} {
			if !strings.Contains(view, want) {
				t.Fatalf("%s tree missing %q:\n%s", tc.label, want, view)
			}
		}
		if strings.Contains(view, "public") {
			t.Fatalf("%s tree must not show a schema:\n%s", tc.label, view)
		}
	}
}

func TestEmptyDatabaseState(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.catalog = &fakeCatalog{engine: domain.EnginePostgreSQL, empty: true}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindDatabase)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if !strings.Contains(StripANSI(m.View().Content), "no objects visible") {
		t.Fatalf("empty state missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestTreeSelectionShowsDetailsUntilEnter(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "down")
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Details" {
		t.Fatalf("tree selection tab = %q, want Details", m.details.currentTab())
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{"Structure", "bigint"} {
		if !strings.Contains(view, want) {
			t.Fatalf("details %q missing:\n%s", want, view)
		}
	}
	if m.data.loaded {
		t.Fatal("tree selection must not load the Data page")
	}
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Data" || !m.data.loaded {
		t.Fatalf("Enter tab=%q loaded=%v", m.details.currentTab(), m.data.loaded)
	}
	if !strings.Contains(StripANSI(m.View().Content), "row-001") {
		t.Fatal("Enter did not load the Data page")
	}
}

func TestTreeMoveKeepsActiveTab(t *testing.T) {
	m := workspace(t)
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Details" || m.details.node.Name != "invoices" || !m.details.structureLoaded {
		t.Fatalf("details move = %q %q loaded=%v", m.details.currentTab(), m.details.node.Name, m.details.structureLoaded)
	}
	m, cmd = press(t, m, "2")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Indexes" || m.details.node.Name != "audit_log" || !m.details.indexesLoaded {
		t.Fatalf("indexes move = %q %q loaded=%v", m.details.currentTab(), m.details.node.Name, m.details.indexesLoaded)
	}
	m, cmd = press(t, m, "4")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "up")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Data" || m.details.node.Name != "invoices" || !m.data.loaded {
		t.Fatalf("data move = %q %q loaded=%v", m.details.currentTab(), m.details.node.Name, m.data.loaded)
	}
}

func TestTreeMoveResetsPagingState(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "ctrl+a")
	m = settle(t, m, cmd)
	if m.data.batch != dataFetchAllSize || len(m.data.rows) < 200 {
		t.Fatalf("setup rows=%d batch=%d", len(m.data.rows), m.data.batch)
	}
	m, _ = press(t, m, "shift+tab")
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	if m.details.currentTab() != "Data" || m.details.node.Name != "invoices" {
		t.Fatalf("tab = %q node = %q", m.details.currentTab(), m.details.node.Name)
	}
	if m.data.batch != dataPageSize || m.data.maxRows != dataMaxWindowRows || m.data.windowStart != 0 {
		t.Fatalf("paging not reset: batch=%d maxRows=%d start=%d", m.data.batch, m.data.maxRows, m.data.windowStart)
	}
	if !m.data.loaded || len(m.data.rows) != 100 {
		t.Fatalf("first page = %d rows, loaded=%v", len(m.data.rows), m.data.loaded)
	}
}

func TestContextTabsAndPlaceholders(t *testing.T) {
	m := workspace(t)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "1 Details · 2 Indexes · 3 Privileges · 4 Data") {
		t.Fatalf("table tabs missing:\n%s", view)
	}
	if !strings.Contains(view, "row-001") {
		t.Fatalf("data grid missing rows:\n%s", view)
	}
	m, cmd := press(t, m, "1")
	m = settle(t, m, cmd)
	view = StripANSI(m.View().Content)
	for _, want := range []string{"public.customers", "Structure", "column", "bigint"} {
		if !strings.Contains(view, want) {
			t.Fatalf("details %q missing:\n%s", want, view)
		}
	}
	m, cmd = press(t, m, "2")
	m = settle(t, m, cmd)
	view = StripANSI(m.View().Content)
	for _, want := range []string{"customers_pkey", "Statistics", "Structure", "btree"} {
		if !strings.Contains(view, want) {
			t.Fatalf("indexes %q missing:\n%s", want, view)
		}
	}
	m, cmd = press(t, m, "3")
	m = settle(t, m, cmd)
	view = StripANSI(m.View().Content)
	for _, want := range []string{"principal", "app_ro", "grants", "SELECT on public.customers"} {
		if !strings.Contains(view, want) {
			t.Fatalf("privileges %q missing:\n%s", want, view)
		}
	}
	m, _ = press(t, m, "4")
	if !strings.Contains(StripANSI(m.View().Content), "row-001") {
		t.Fatal("4 must show the Data tab")
	}
	m = selectKind(t, m, domain.KindView)
	if !strings.Contains(StripANSI(m.View().Content), "detail view not available in v1") {
		t.Fatal("view placeholder missing")
	}
	if strings.Contains(StripANSI(m.View().Content), "1 Data") {
		t.Fatal("view must not show a Data tab")
	}
	m = selectKind(t, m, domain.KindSequence)
	if !strings.Contains(StripANSI(m.View().Content), "detail view not available in v1") {
		t.Fatal("sequence placeholder missing")
	}
}

func TestSecurityPermissionError(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.catalog = &fakeCatalog{
		engine:       domain.EnginePostgreSQL,
		principalErr: domain.NewError(domain.KindPermission, "catalog", "permission denied"),
	}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindUsers)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	if !strings.Contains(StripANSI(m.View().Content), "not permitted") {
		t.Fatalf("permission state missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestLazyChildrenAreCached(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	catalog := &fakeCatalog{engine: domain.EnginePostgreSQL}
	e.connector.catalog = catalog
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindDatabase)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if catalog.childrenCalls != 1 {
		t.Fatalf("children calls = %d", catalog.childrenCalls)
	}
	m, _ = press(t, m, "left")
	m, _ = press(t, m, "right")
	if catalog.childrenCalls != 1 {
		t.Fatalf("cached children reloaded: %d calls", catalog.childrenCalls)
	}
}

func TestTreeHomeEndAndPageMovement(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "home")
	if p, _ := m.tree.selectedNode(); p.Name != "orders" {
		t.Fatalf("home selected %q", p.Name)
	}
	m, _ = press(t, m, "end")
	if p, _ := m.tree.selectedNode(); p.Kind != domain.KindSecurity {
		t.Fatalf("end selected %q", p.Name)
	}
	tree := newTree()
	tree.rows = []treeRow{
		{Key: "a"},
		{Key: "info", Info: "loading…"},
		{Key: "b"},
	}
	tree.selected = 0
	tree.move(1)
	if tree.selected != 2 {
		t.Fatalf("move over info selected %d", tree.selected)
	}
	tree.move(-1)
	if tree.selected != 0 {
		t.Fatalf("reverse move selected %d", tree.selected)
	}
}

func TestTreeRootErrorAndRetry(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	catalog := &fakeCatalog{engine: domain.EnginePostgreSQL, databasesErr: domain.NewError(domain.KindPermission, "catalog", "permission denied")}
	e.connector.catalog = catalog
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	if m.tree.rootErr == "" {
		t.Fatal("root error not recorded")
	}
	if !strings.Contains(StripANSI(m.View().Content), "not permitted") {
		t.Fatalf("root error missing:\n%s", StripANSI(m.View().Content))
	}
	catalog.databasesErr = nil
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.tree.rootErr != "" || len(m.tree.roots) == 0 {
		t.Fatalf("retry did not load roots: err=%q roots=%v", m.tree.rootErr, m.tree.roots)
	}
}

func TestTreeParentErrorIsAttached(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.catalog = &fakeCatalog{engine: domain.EnginePostgreSQL, childrenErrAll: true}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	m = selectKind(t, m, domain.KindDatabase)
	key := m.tree.selectedKey()
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	node, ok := m.tree.node(key)
	if !ok || node.Err == "" {
		t.Fatalf("parent error not attached: %+v", node)
	}
	if !strings.Contains(StripANSI(m.viewTree(21, 28)), "error not permitted") {
		t.Fatalf("tree error row missing:\n%s", StripANSI(m.viewTree(21, 28)))
	}
}

func TestUnknownAccessOmitsMarker(t *testing.T) {
	node := domain.Node{Kind: domain.KindTable, Name: "customers", Access: domain.AccessUnknown}
	line := treeRowText(treeRow{Node: node})
	if strings.Contains(line, "—") || strings.Contains(line, "ro") || strings.Contains(line, "rw") {
		t.Fatalf("unknown access rendered a marker: %q", line)
	}
}

func TestRightOnLeafKeepsTreeFocus(t *testing.T) {
	m := workspace(t)
	if m.focus != FocusTree {
		t.Fatalf("focus = %v", m.focus)
	}
	m, _ = press(t, m, "right")
	if m.focus != FocusTree {
		t.Fatalf("right on leaf moved focus to %v", m.focus)
	}
}

func TestStaleTreeRootAfterDisconnect(t *testing.T) {
	m := workspace(t)
	oldGen := m.treeGen
	m, _ = press(t, m, "ctrl+d")
	if m.screen != ScreenPicker {
		t.Fatal("not disconnected")
	}
	next, _ := m.Update(treeRootMsg{gen: oldGen, nodes: []domain.Node{{Name: "late", Kind: domain.KindDatabase}}})
	m = next.(Model)
	if len(m.tree.roots) != 0 || strings.Contains(m.message, "connected ·") {
		t.Fatalf("late root mutated picker: roots=%v message=%q", m.tree.roots, m.message)
	}
}

func TestDetailsScrollAndPosition(t *testing.T) {
	m := workspace(t)
	m.details.node = domain.Node{Kind: domain.KindUser, Name: "role", Scope: "cluster"}
	for i := 0; i < 30; i++ {
		m.details.node.Attributes = append(m.details.node.Attributes, "ATTR")
	}
	m.details.selectNode(m.details.node)
	m.scrollDetails(10)
	if m.details.offset == 0 {
		t.Fatal("details did not scroll")
	}
	if !strings.Contains(StripANSI(m.View().Content), "of") {
		t.Fatal("position indicator missing")
	}
}

func TestProfileRestoresLastObject(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	m, _ = press(t, m, "ctrl+d")
	loaded, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var saved *domain.ObjectRef
	for _, p := range loaded {
		if p.Label == "prod-ro" {
			saved = p.LastObject
		}
	}
	if saved == nil || saved.Kind != domain.KindTable || saved.Name != "customers" {
		t.Fatalf("last object = %+v", saved)
	}
	m = connectSelected(t, m)
	node, ok := m.tree.selectedNode()
	if !ok || node.Kind != domain.KindTable || node.Name != "customers" {
		t.Fatalf("restored selection = %+v, %v", node, ok)
	}
	if !strings.Contains(StripANSI(m.View().Content), "customers  ro") {
		t.Fatalf("restored tree missing customers:\n%s", StripANSI(m.View().Content))
	}
}

func TestRestoreFindsObjectUnderSchemasGroup(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	profiles, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref := domain.ObjectRef{
		Engine: domain.EnginePostgreSQL, Kind: domain.KindTable,
		Database: "orders", Schema: "reporting", Name: "customers",
	}
	for i := range profiles {
		if profiles[i].Label == "prod-ro" {
			profiles[i].LastObject = &ref
		}
	}
	if err := e.store.Save(context.Background(), profiles); err != nil {
		t.Fatal(err)
	}
	m := selectLabel(t, e.model(t), "prod-ro")
	m = connectSelected(t, m)
	node, ok := m.tree.selectedNode()
	if !ok || node.Name != "customers" || node.Ref.Schema != "reporting" {
		t.Fatalf("restored selection = %+v, %v", node, ok)
	}
}

func TestQuitSavesLastObject(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	if _, cmd := press(t, m, "ctrl+q"); cmd == nil {
		t.Fatal("Ctrl+Q must quit")
	}
	loaded, err := e.store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range loaded {
		if p.Label != "prod-ro" {
			continue
		}
		if p.LastObject == nil || p.LastObject.Name != "customers" {
			t.Fatalf("last object = %+v", p.LastObject)
		}
		return
	}
	t.Fatal("prod-ro not found")
}

func TestReconnectResetsTabAndRestoresLastObject(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := selectLabel(t, e.model(t), "prod-ro")
	m = workspaceFrom(t, m)
	m, _ = press(t, m, "3")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "ctrl+d")
	m = connectSelected(t, m)
	node, ok := m.tree.selectedNode()
	if m.details.tab != 0 || !ok || node.Name != "invoices" || m.screen != ScreenWorkspace {
		t.Fatalf("workspace not restored: tab=%d node=%+v ok=%v screen=%v", m.details.tab, node, ok, m.screen)
	}
}
