package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func focusQuery(t *testing.T, m Model) Model {
	t.Helper()
	for i := 0; i < 3 && m.focus != FocusQuery; i++ {
		m, _ = press(t, m, "tab")
	}
	if m.focus != FocusQuery {
		t.Fatalf("focus = %v", m.focus)
	}
	return m
}

func setQuery(m Model, statement string) Model {
	m.query.SetValue(statement)
	m.query.SetCursor(len([]rune(statement)))
	return m
}

func TestQueryExecuteResultsAndLeave(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{
		{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "row-001", Kind: domain.CellText}}},
				},
			},
			HasMore: true,
		},
		{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "row-002", Kind: domain.CellText}}},
				},
			},
		},
	}}
	e.connector.query = executor
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT id, name FROM customers")
	m, cmd := press(t, m, "enter")
	if !m.resultsActive || !m.queryRunning {
		t.Fatalf("query did not start: results=%v running=%v message=%q", m.resultsActive, m.queryRunning, m.message)
	}
	m = runQueryCmd(t, m, cmd)
	if m.queryRunning {
		t.Fatal("query still running")
	}
	if m.details.currentTab() != "Results" {
		t.Fatalf("tab = %q", m.details.currentTab())
	}
	if executor.request.Statement != "SELECT id, name FROM customers" {
		t.Fatalf("request = %+v", executor.request)
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{"Results", "row-001", "1 row"} {
		if !strings.Contains(view, want) {
			t.Fatalf("results view missing %q:\n%s", want, view)
		}
	}

	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	if m.focus != FocusMain {
		t.Fatalf("focus = %v", m.focus)
	}
	m, cmd = press(t, m, "m")
	m = runCmd(t, m, cmd)
	if len(m.data.rows) != 2 || m.data.hasMore {
		t.Fatalf("after m rows=%d more=%v", len(m.data.rows), m.data.hasMore)
	}
	if !strings.Contains(StripANSI(m.View().Content), "row-002") {
		t.Fatalf("second page missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "m")
	if m.message != "end of data" {
		t.Fatalf("m at end message = %q", m.message)
	}

	m, _ = press(t, m, "esc")
	if m.resultsActive {
		t.Fatal("Esc did not leave Results")
	}
	if m.details.currentTab() != "Data" {
		t.Fatalf("restored tab = %q", m.details.currentTab())
	}
	if len(m.data.rows) != 5 {
		t.Fatalf("restored data rows = %d", len(m.data.rows))
	}
}

func TestQueryErrorKeepsStatement(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.query = &fakeQueryExecutor{err: domain.NewError(domain.KindEngine, "query", "relation \"invoies\" does not exist")}
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT * FROM invoies")
	m, cmd := press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)
	view := StripANSI(m.View().Content)
	for _, want := range []string{"ERROR: relation", "SELECT * FROM invoies", "Esc returns to public.customers."} {
		if !strings.Contains(view, want) {
			t.Fatalf("error view missing %q:\n%s", want, view)
		}
	}
	m, _ = press(t, m, "esc")
	if m.resultsActive || m.details.currentTab() != "Data" {
		t.Fatalf("error Esc did not restore: active=%v tab=%q", m.resultsActive, m.details.currentTab())
	}
	if m.query.Value() != "SELECT * FROM invoies" {
		t.Fatalf("statement was not preserved: %q", m.query.Value())
	}
}

func TestQueryBlockedBeforeEngine(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	executor := &fakeQueryExecutor{}
	e.connector.query = executor
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "DELETE FROM customers")
	m, cmd := press(t, m, "enter")
	if cmd != nil {
		t.Fatal("blocked query returned a command")
	}
	if m.resultsActive {
		t.Fatal("blocked query opened Results")
	}
	if !strings.HasPrefix(m.message, "Blocked: ") {
		t.Fatalf("message = %q", m.message)
	}
	if executor.request.Statement != "" {
		t.Fatalf("blocked query reached the engine: %+v", executor.request)
	}
}

func TestQueryHistoryRecallAndAppend(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	historyStore := &fakeHistoryStore{entries: []domain.HistoryEntry{{Statement: "SELECT 1"}}}
	e.svc.SetHistory(historyStore)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	if len(m.history) != 1 {
		t.Fatalf("history = %+v", m.history)
	}
	m = setQuery(m, "SELECT 2")
	m, cmd := press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)
	if len(historyStore.entries) != 2 || historyStore.entries[1].Statement != "SELECT 2" {
		t.Fatalf("stored history = %+v", historyStore.entries)
	}
	m.messagesClear()
	m = setQuery(m, "")
	m, _ = press(t, m, "up")
	if m.query.Value() != "SELECT 2" {
		t.Fatalf("Up recalled %q", m.query.Value())
	}
	m, _ = press(t, m, "up")
	if m.query.Value() != "SELECT 1" {
		t.Fatalf("second Up recalled %q", m.query.Value())
	}
	m, _ = press(t, m, "down")
	if m.query.Value() != "SELECT 2" {
		t.Fatalf("Down recalled %q", m.query.Value())
	}
}

func TestQueryCompletionAndHelp(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = typeString(t, m, "sel")
	m, _ = press(t, m, "ctrl+space")
	if !m.completion.active {
		t.Fatalf("completion inactive: message=%q", m.message)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "completion") || !strings.Contains(view, "SELECT") {
		t.Fatalf("completion popup missing:\n%s", view)
	}
	m, _ = press(t, m, "enter")
	if m.completion.active {
		t.Fatal("completion stayed open")
	}
	if m.query.Value() != "SELECT" {
		t.Fatalf("accepted completion = %q", m.query.Value())
	}

	m = setQuery(m, "select")
	m, _ = press(t, m, "ctrl+f1")
	if m.modal != modalHelp {
		t.Fatalf("help modal = %v", m.modal)
	}
	view = StripANSI(m.View().Content)
	if !strings.Contains(view, "projects rows") || !strings.Contains(view, "Enter examples") {
		t.Fatalf("help body missing:\n%s", view)
	}
	m, _ = press(t, m, "enter")
	if !m.helpExtended {
		t.Fatal("Enter did not open extended help")
	}
	view = StripANSI(m.View().Content)
	for _, want := range []string{"examples", "Examples", "SELECT", "FROM public.customers"} {
		if !strings.Contains(view, want) {
			t.Fatalf("extended help missing %q:\n%s", want, view)
		}
	}
	m, _ = press(t, m, "esc")
	if m.modal != modalNone {
		t.Fatal("help did not close")
	}
}

func TestMongoFindHelpShowsExamples(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := connectSelected(t, selectLabel(t, e.model(t), "analytics"))
	m = focusQuery(t, m)
	m = setQuery(m, "find")
	m, _ = press(t, m, "ctrl+f1")
	m, _ = press(t, m, "enter")
	view := StripANSI(m.View().Content)
	for _, want := range []string{"Examples", "$and", "Projection"} {
		if !strings.Contains(view, want) {
			t.Fatalf("find examples missing %q:\n%s", want, view)
		}
	}
	for i := 0; i < 12 && !strings.Contains(view, "_id: 0"); i++ {
		m, _ = press(t, m, "down")
		view = StripANSI(m.View().Content)
	}
	if !strings.Contains(view, "_id: 0") {
		t.Fatalf("projection example did not scroll into view:\n%s", view)
	}
}

func TestHelpCtrlF1OpensExamples(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "select")
	m, _ = press(t, m, "ctrl+f1")
	if m.modal != modalHelp || m.helpExtended {
		t.Fatalf("help modal=%v extended=%v", m.modal, m.helpExtended)
	}
	m, _ = press(t, m, "ctrl+f1")
	if !m.helpExtended {
		t.Fatal("second Ctrl+F1 did not open extended help")
	}
	if !strings.Contains(StripANSI(m.View().Content), "Examples") {
		t.Fatalf("examples missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "esc")
	if m.modal != modalNone {
		t.Fatal("help did not close")
	}
}

func TestMQLCompletionInsertsArrayTemplate(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := connectSelected(t, selectLabel(t, e.model(t), "analytics"))
	m = focusQuery(t, m)
	m = setQuery(m, "$o")
	m, _ = press(t, m, "ctrl+space")
	m = chooseCompletion(t, m, "$or")
	if m.query.Value() != "$or: [  ]" {
		t.Fatalf("array template = %q", m.query.Value())
	}
	if m.query.Position() != 7 {
		t.Fatalf("array cursor = %d, want 7", m.query.Position())
	}
}

func TestFunctionCompletionInsertsParentheses(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "cou")
	m, _ = press(t, m, "ctrl+space")
	m = chooseCompletion(t, m, "COUNT")
	if m.query.Value() != "COUNT()" {
		t.Fatalf("function template = %q", m.query.Value())
	}
	if m.query.Position() != 6 {
		t.Fatalf("function cursor = %d, want 6", m.query.Position())
	}
}

func chooseCompletion(t *testing.T, m Model, want string) Model {
	t.Helper()
	for i, item := range m.completion.items {
		if item == want {
			m.completion.selected = i
			m, _ = press(t, m, "enter")
			return m
		}
	}
	t.Fatalf("%q missing from %v", want, m.completion.items)
	return m
}

func TestEscTabOpensCompletion(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "sel")
	m, _ = press(t, m, "esc")
	if !m.escChord {
		t.Fatal("Esc did not arm the completion chord")
	}
	m, _ = press(t, m, "tab")
	if !m.completion.active {
		t.Fatalf("Esc+Tab did not open completion: message=%q", m.message)
	}
	if m.focus != FocusQuery {
		t.Fatalf("Esc+Tab moved focus to %v", m.focus)
	}
	m, _ = press(t, m, "enter")
	if m.query.Value() != "SELECT" {
		t.Fatalf("accepted completion = %q", m.query.Value())
	}
}

func TestDoubleEscTabOpensCompletion(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "sel")
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "tab")
	if !m.completion.active {
		t.Fatalf("Esc Esc Tab did not open completion: message=%q", m.message)
	}
	if m.focus != FocusQuery {
		t.Fatalf("Esc Esc Tab moved focus to %v", m.focus)
	}
}

func TestCompletionUsesWholeWordInMiddleAndAtStart(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	for _, cursor := range []int{0, 2} {
		m = setQuery(m, "SELECT")
		m.query.SetCursor(cursor)
		m, _ = press(t, m, "ctrl+space")
		if !m.completion.active {
			t.Fatalf("cursor %d: completion inactive: message=%q", cursor, m.message)
		}
		found := false
		for _, item := range m.completion.items {
			if item == "SELECT" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("cursor %d: SELECT missing from %v", cursor, m.completion.items)
		}
		for i, item := range m.completion.items {
			if item == "SELECT" {
				m.completion.selected = i
			}
		}
		m, _ = press(t, m, "enter")
		if m.query.Value() != "SELECT" {
			t.Fatalf("cursor %d: accepted completion = %q", cursor, m.query.Value())
		}
	}
}

func TestMongoCompletionAfterColonUsesTheKey(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := connectSelected(t, selectLabel(t, e.model(t), "analytics"))
	m = focusQuery(t, m)
	m = setQuery(m, "$o:")
	m.query.SetCursor(3)
	m, _ = press(t, m, "ctrl+space")
	if !m.completion.active {
		t.Fatalf("completion inactive: message=%q", m.message)
	}
	found := false
	for i, item := range m.completion.items {
		if item == "$or" {
			found = true
			m.completion.selected = i
		}
	}
	if !found {
		t.Fatalf("$or missing from %v", m.completion.items)
	}
	m, _ = press(t, m, "enter")
	if m.query.Value() != "$or:[  ]" {
		t.Fatalf("colon completion = %q", m.query.Value())
	}
	m.query.SetCursor(4)
	m, _ = press(t, m, "ctrl+f1")
	if m.modal != modalHelp || m.helpTerm != "$or" {
		t.Fatalf("help modal=%v term=%q", m.modal, m.helpTerm)
	}
	m, _ = press(t, m, "enter")
	if !strings.Contains(StripANSI(m.View().Content), "Either predicate may match") {
		t.Fatalf("$or examples missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestCompletionPopupAnchorsAtCursor(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT")
	m.query.SetCursor(3)
	m, _ = press(t, m, "ctrl+space")
	if !m.completion.active {
		t.Fatal("completion did not open")
	}
	x, y := m.completionAnchor(20, 6)
	if x != 11 {
		t.Fatalf("popup x = %d, want 11", x)
	}
	if want := m.height - 2 - 6; y != want {
		t.Fatalf("popup y = %d, want %d", y, want)
	}
	if y+6 > m.height-2 {
		t.Fatalf("popup %d..%d overlaps the query line at %d", y, y+6, m.height-2)
	}

	m.query.SetCursor(0)
	m.refreshCompletion()
	x, _ = m.completionAnchor(20, 6)
	if x != 8 {
		t.Fatalf("popup x at word start = %d, want 8", x)
	}
}

func TestStatementCompletionPopupAnchorsAtCursor(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+e")
	if m.modal != modalStatement {
		t.Fatalf("modal = %v", m.modal)
	}
	m.statementInput.SetCursor(0)
	m, _ = press(t, m, "ctrl+space")
	if !m.completion.active {
		t.Fatal("modal completion did not open")
	}
	x0, y := m.completionAnchor(20, 6)
	m.statementInput.SetCursor(3)
	m.refreshCompletion()
	x3, _ := m.completionAnchor(20, 6)
	if x3-x0 != 3 {
		t.Fatalf("statement popup x moved %d, want 3", x3-x0)
	}
	if y < 0 {
		t.Fatalf("statement popup y = %d", y)
	}
}

func TestEscTabOpensCompletionInStatementModal(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "sel")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+e")
	if m.modal != modalStatement {
		t.Fatalf("modal = %v", m.modal)
	}
	m.statementInput.SetCursor(len([]rune("sel")))
	m, _ = press(t, m, "esc")
	if !m.escChord {
		t.Fatal("Esc did not arm the modal chord")
	}
	m, _ = press(t, m, "tab")
	if m.modal != modalStatement {
		t.Fatalf("Esc+Tab closed the modal: %v", m.modal)
	}
	if !m.completion.active {
		t.Fatal("Esc+Tab did not open modal completion")
	}
	m, _ = press(t, m, "enter")
	if m.statementInput.Value() != "SELECT" {
		t.Fatalf("modal completion = %q", m.statementInput.Value())
	}
}

func TestTabCompletesInStatementModal(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = setQuery(m, "sel")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+e")
	if m.modal != modalStatement {
		t.Fatalf("modal = %v", m.modal)
	}
	m.statementInput.SetCursor(len([]rune("sel")))
	m, _ = press(t, m, "tab")
	if !m.completion.active {
		t.Fatal("Tab did not open modal completion")
	}
	m, _ = press(t, m, "tab")
	if m.completion.active {
		t.Fatal("second Tab did not accept completion")
	}
	if m.statementInput.Value() != "SELECT" {
		t.Fatalf("completion value = %q", m.statementInput.Value())
	}
}

func TestEscTabDoesNotStealCompletionEscape(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m = typeString(t, m, "sel")
	m, _ = press(t, m, "ctrl+space")
	if !m.completion.active {
		t.Fatal("completion did not open")
	}
	m, _ = press(t, m, "esc")
	if m.completion.active || m.escChord {
		t.Fatalf("Esc did not close completion: active=%v chord=%v", m.completion.active, m.escChord)
	}
}

func TestStatementPromptUsesMQL(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "analytics"))
	m = focusQuery(t, m)
	m = setQuery(m, "db.customers.find({})")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+e")
	if m.modal != modalStatement {
		t.Fatalf("modal = %v", m.modal)
	}
	if view := StripANSI(m.viewModal()); !strings.Contains(view, "MQL> db.customers.find({})") {
		t.Fatalf("compact MQL prompt missing:\n%s", view)
	}
}

func TestStatementEditModalExecutes(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
		Columns: []domain.Column{{Name: "count"}},
		Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "5", Kind: domain.CellNumber}}}},
	}}}}
	e.connector.query = executor
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT 1")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+e")
	if m.modal != modalStatement {
		t.Fatalf("modal = %v", m.modal)
	}
	if view := StripANSI(m.viewModal()); !strings.Contains(view, "SQL> SELECT 1") {
		t.Fatalf("compact SQL prompt missing:\n%s", view)
	}
	m.statementInput.SetValue("SELECT count(*) FROM customers")
	m.statementInput.SetCursor(len([]rune("SELECT count(*) FROM customers")))
	m, cmd := press(t, m, "enter")
	if m.modal != modalNone {
		t.Fatal("edit modal stayed open")
	}
	m = runQueryCmd(t, m, cmd)
	if executor.request.Statement != "SELECT count(*) FROM customers" {
		t.Fatalf("request = %+v", executor.request)
	}
	if m.query.Value() != "SELECT count(*) FROM customers" {
		t.Fatalf("query bar = %q", m.query.Value())
	}
	if !m.resultsActive {
		t.Fatal("edited statement did not run")
	}
}

func TestResultsTwoColumnsUseTheFullWidth(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	left := strings.Repeat("L", 50)
	right := strings.Repeat("R", 50)
	e.connector.query = &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
		Columns: []domain.Column{{Name: "left"}, {Name: "right"}},
		Rows: []domain.Row{{Cells: []domain.Cell{
			{Text: left, Kind: domain.CellText},
			{Text: right, Kind: domain.CellText},
		}}},
	}}}}
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = next.(Model)
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT left, right FROM data_sample")
	m, cmd := press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, left) || !strings.Contains(view, right) {
		t.Fatalf("wide result values were truncated:\n%s", view)
	}
	if strings.Contains(view, "…") {
		t.Fatalf("wide result still shows an ellipsis:\n%s", view)
	}

	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	view = StripANSI(m.View().Content)
	if !strings.Contains(view, "left") || !strings.Contains(view, "right") {
		t.Fatalf("two-column result hid or truncated a column:\n%s", view)
	}
}

func TestResultsTabRestoresOnObjectOpen(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
		Columns: []domain.Column{{Name: "id"}},
		Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
	}}}}
	e.connector.query = executor
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = focusQuery(t, m)
	m = setQuery(m, "SELECT 1")
	m, cmd := press(t, m, "enter")
	m = runQueryCmd(t, m, cmd)
	m, _ = press(t, m, "tab")
	if m.focus != FocusTree {
		t.Fatalf("focus = %v", m.focus)
	}
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.resultsActive {
		t.Fatal("opening an object from Results did not release it")
	}
	if m.details.currentTab() != "Data" {
		t.Fatalf("tab = %q", m.details.currentTab())
	}
}

// messagesClear drops the transient status text between history assertions.
func (m *Model) messagesClear() {
	m.message = ""
	m.messageLevel = levelNone
}
