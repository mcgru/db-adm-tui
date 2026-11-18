package tui

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func press(t *testing.T, m Model, key string) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.handleKey(key)
	mm, ok := next.(Model)
	if !ok {
		t.Fatalf("handleKey(%q) returned %T", key, next)
	}
	return mm, cmd
}

func workspace(t *testing.T) Model {
	t.Helper()
	m, _ := press(t, New(), "enter")
	if m.screen != ScreenWorkspace {
		t.Fatal("expected workspace")
	}
	return m
}

func TestGolden(t *testing.T) {
	m := New()
	assertGolden(t, "picker.golden", m.View().Content)

	m = workspace(t)
	assertGolden(t, "workspace.golden", m.View().Content)

	m, _ = press(t, m, "2")
	assertGolden(t, "workspace-details.golden", m.View().Content)

	m, _ = press(t, m, "3")
	assertGolden(t, "workspace-privileges.golden", m.View().Content)

	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-query-focus.golden", m.View().Content)

	m = workspace(t)
	m, _ = press(t, m, "tab")
	assertGolden(t, "workspace-main-focus.golden", m.View().Content)

	next, _ := New().Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	assertGolden(t, "too-small.golden", next.(Model).View().Content)

	wide := New()
	next, _ = wide.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	wide = next.(Model)
	wide = func() Model { mm, _ := press(t, wide, "enter"); return mm }()
	assertGolden(t, "workspace-wide.golden", wide.View().Content)

	narrow := New()
	next, _ = narrow.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	narrow = next.(Model)
	assertGolden(t, "too-small-narrow.golden", narrow.View().Content)
}

func TestTooSmallRecovers(t *testing.T) {
	m := New()
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
	m := New()
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
	m := New()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 20})
	m = next.(Model)

	m, cmd := press(t, m, "enter")
	if cmd != nil || m.screen != ScreenPicker {
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

func TestPickerNavigationAndProfileSwitch(t *testing.T) {
	m := New()
	m, _ = press(t, m, "up")
	if m.pickerSel != 0 {
		t.Fatalf("up from first row = %d", m.pickerSel)
	}
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	if m.pickerSel != 2 {
		t.Fatalf("down clamp = %d, want 2", m.pickerSel)
	}
	m, _ = press(t, m, "enter")
	if m.activeProfile != 2 {
		t.Fatalf("active profile = %d, want 2", m.activeProfile)
	}
	top := StripANSI(m.viewTopStatus(80))
	if !strings.Contains(top, "billing") || !strings.Contains(top, "MySQL") {
		t.Fatalf("top status = %q", top)
	}
	if !strings.Contains(StripANSI(m.viewQuery(80)), "SQL") {
		t.Fatalf("MySQL query prompt = %q", StripANSI(m.viewQuery(80)))
	}
}

func TestMongoProfileUsesMQL(t *testing.T) {
	m := New()
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "enter")
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "analytics") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	if !strings.Contains(StripANSI(m.viewQuery(80)), "MQL") {
		t.Fatalf("query prompt = %q", StripANSI(m.viewQuery(80)))
	}
}

func TestCyrillicMessages(t *testing.T) {
	m := New()
	m, _ = press(t, m, "т")
	if !strings.Contains(m.message, "new connection") {
		t.Fatalf("т message = %q", m.message)
	}
	m, _ = press(t, m, "ш")
	if !strings.Contains(m.message, "import") {
		t.Fatalf("ш message = %q", m.message)
	}
}

func TestMoreAndHelpMessages(t *testing.T) {
	m := New()
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
	m := New()
	next, _ := m.Update(tea.KeyPressMsg{Text: "т", Code: 'т'})
	m = next.(Model)
	if !strings.Contains(m.message, "new connection") {
		t.Fatalf("cyrillic key press was not routed: %q", m.message)
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
	m := New()
	m, _ = press(t, m, "n")
	if m.message == "" {
		t.Fatal("n should set a message")
	}
	m, _ = press(t, m, "x")
	if m.message != "" {
		t.Fatalf("message did not clear: %q", m.message)
	}
}

func TestReconnectResetsWorkspaceState(t *testing.T) {
	m := New()
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "enter")
	m, _ = press(t, m, "3")
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "down")

	m, _ = press(t, m, "ctrl+d")
	if m.screen != ScreenPicker {
		t.Fatal("expected picker")
	}
	if m.pickerSel != 1 {
		t.Fatalf("picker selection = %d, want disconnected profile", m.pickerSel)
	}

	m, _ = press(t, m, "enter")
	if m.tab != 0 || m.treeSel != 2 || m.focus != FocusTree {
		t.Fatalf("workspace not reset: tab=%d tree=%d focus=%v", m.tab, m.treeSel, m.focus)
	}
}

func TestRawSelectionAndFocusStyles(t *testing.T) {
	m := New()
	picker := m.View().Content
	selectedProfile := m.styles.Select.Render(Pad(profileFields(m.profiles[0]), 80))
	if !strings.Contains(picker, selectedProfile) {
		t.Fatal("selected picker row lacks the Select style")
	}

	m = workspace(t)
	view := m.View().Content
	if !strings.Contains(view, m.styles.Focused.Render("┌")) {
		t.Fatal("focused tree frame lacks the Focused style")
	}
	if !strings.Contains(view, m.styles.Focused.Render(" objects 3 ")) {
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
	m := New()
	next, _ := m.Update(tea.KeyPressMsg{Text: "n", Code: 'n'})
	m = next.(Model)
	if !strings.Contains(m.message, "new connection") {
		t.Fatalf("key press was not routed: %q", m.message)
	}
}

func TestAltScreenAndBackground(t *testing.T) {
	v := New().View()
	if !v.AltScreen {
		t.Fatal("shell must use the alternate screen")
	}
	if v.BackgroundColor == nil {
		t.Fatal("shell must set the Nightshift base background")
	}
}
