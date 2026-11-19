package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestCaseModeIndicatorAndToggle(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	before := len(reader.calls)
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: insensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "i")
	if m.caseInsensitive {
		t.Fatal("Ctrl+X i did not switch to sensitive")
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: sensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	if len(reader.calls) != before {
		t.Fatal("case toggle queried without a filter")
	}
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "i")
	if !m.caseInsensitive {
		t.Fatal("Ctrl+X i did not switch back to insensitive")
	}
}

func TestCaseModeResetsPerSession(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "i")
	if m.caseInsensitive {
		t.Fatal("case mode did not toggle")
	}
	m, _ = press(t, m, "ctrl+d")
	m = connectSelected(t, m)
	if !m.caseInsensitive {
		t.Fatal("case mode survived the session")
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: insensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
}

func TestBufferFilterNarrowsAndRestores(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	before := len(reader.calls)
	m, _ = press(t, m, "/")
	if m.data.prompt != promptBuffer {
		t.Fatalf("prompt = %v, want buffer", m.data.prompt)
	}
	m = typeString(t, m, "active")
	if len(reader.calls) != before {
		t.Fatal("/ queried the database")
	}
	if len(m.data.matches) != 2 {
		t.Fatalf("matches = %v, want rows 1 and 4", m.data.matches)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "row-001") || !strings.Contains(view, "row-004") {
		t.Fatalf("filtered view missing matches:\n%s", view)
	}
	if strings.Contains(view, "row-002") || strings.Contains(view, "row-003") || strings.Contains(view, "row-005") {
		t.Fatalf("filtered view kept non-matches:\n%s", view)
	}
	if !strings.Contains(view, "match 1 of 2") {
		t.Fatalf("match counter missing:\n%s", view)
	}
	if !strings.Contains(m.View().Content, m.styles.Match.Render("active")) {
		t.Fatal("match highlight missing")
	}

	m, _ = press(t, m, "enter")
	if m.data.prompt != promptNone || m.data.bufferPattern != "active" {
		t.Fatalf("Enter prompt=%v pattern=%q", m.data.prompt, m.data.bufferPattern)
	}
	if !strings.Contains(StripANSI(m.View().Content), "match 1 of 2") {
		t.Fatal("filter bar disappeared after Enter")
	}

	m, _ = press(t, m, "esc")
	if m.data.bufferPattern != "" || len(m.data.matches) != 0 {
		t.Fatalf("Esc pattern=%q matches=%v", m.data.bufferPattern, m.data.matches)
	}
	view = StripANSI(m.View().Content)
	for _, want := range []string{"row-001", "row-002", "row-003", "row-004", "row-005"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Esc did not restore %q:\n%s", want, view)
		}
	}
	if len(reader.calls) != before {
		t.Fatal("clearing the buffer filter queried the database")
	}
}

func TestBufferFilterMatchesLiterally(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "row-00.")
	if len(m.data.matches) != 0 {
		t.Fatalf("regexp metacharacter matched rows: %v", m.data.matches)
	}
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "/")
	m = typeString(t, m, "row-00")
	if len(m.data.matches) != 5 {
		t.Fatalf("literal matches = %v, want all rows", m.data.matches)
	}
}

func TestBufferFilterNoMatchStaysEditable(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "zzz")
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, `No matches for "zzz". Esc clears the filter.`) {
		t.Fatalf("no-match copy missing:\n%s", view)
	}
	if m.dataSummary() != "0 of 5 loaded" {
		t.Fatalf("summary = %q", m.dataSummary())
	}
	m = typeString(t, m, "a")
	if m.data.bufferPattern != "zzza" {
		t.Fatalf("pattern = %q, want editable", m.data.bufferPattern)
	}
	m, _ = press(t, m, "backspace")
	if m.data.bufferPattern != "zzz" {
		t.Fatalf("backspace pattern = %q", m.data.bufferPattern)
	}
	m, _ = press(t, m, "esc")
	if m.data.bufferPattern != "" {
		t.Fatalf("Esc pattern = %q", m.data.bufferPattern)
	}
}

func TestBufferFilterMatchNavigationWraps(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	m, _ = press(t, m, "enter")
	if m.data.cursorRow != 0 {
		t.Fatalf("cursor = %d, want first match", m.data.cursorRow)
	}

	m, _ = press(t, m, "n")
	if m.data.cursorRow != 3 {
		t.Fatalf("n cursor = %d, want 3", m.data.cursorRow)
	}
	if !strings.Contains(StripANSI(m.View().Content), "match 2 of 2") {
		t.Fatalf("counter after n missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 0 {
		t.Fatalf("wrap cursor = %d, want 0", m.data.cursorRow)
	}
	m, _ = press(t, m, "N")
	if m.data.cursorRow != 3 {
		t.Fatalf("N cursor = %d, want previous (3)", m.data.cursorRow)
	}
	if m.screen != ScreenWorkspace || m.modal != modalNone {
		t.Fatalf("n left the data view: screen=%v modal=%v", m.screen, m.modal)
	}
}

func TestBufferFilterMatchNavigationVisitsCells(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "a"}, {Name: "b"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{
						{Text: "x", Kind: domain.CellText},
						{Text: "x", Kind: domain.CellText},
					}},
					{Cells: []domain.Cell{
						{Text: "y", Kind: domain.CellText},
						{Text: "x", Kind: domain.CellText},
					}},
				},
			},
			OrderKey:      "a",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "x")
	m, _ = press(t, m, "enter")
	if len(m.data.matchCells) != 3 {
		t.Fatalf("matches = %+v", m.data.matchCells)
	}
	if m.data.cursorRow != 0 || m.data.cursorCol != 0 {
		t.Fatalf("first match = %d/%d", m.data.cursorRow, m.data.cursorCol)
	}
	if !strings.Contains(StripANSI(m.View().Content), "match 1 of 3") {
		t.Fatalf("counter missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 0 || m.data.cursorCol != 1 {
		t.Fatalf("second match = %d/%d", m.data.cursorRow, m.data.cursorCol)
	}
	if !strings.Contains(StripANSI(m.View().Content), "match 2 of 3") {
		t.Fatalf("counter after n missing:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 1 || m.data.cursorCol != 1 {
		t.Fatalf("third match = %d/%d", m.data.cursorRow, m.data.cursorCol)
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 0 || m.data.cursorCol != 0 {
		t.Fatalf("wrapped match = %d/%d", m.data.cursorRow, m.data.cursorCol)
	}
	m, _ = press(t, m, "N")
	if m.data.cursorRow != 1 || m.data.cursorCol != 1 {
		t.Fatalf("previous match = %d/%d", m.data.cursorRow, m.data.cursorCol)
	}
}

func TestBufferFilterPromptOwnsPagingGestures(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	m.data.cursorRow = 99
	m.ensureDataViewport()
	before := len(reader.calls)
	m, _ = press(t, m, "/")
	m, _ = press(t, m, "m")
	m, _ = press(t, m, "pgdown")
	m, _ = press(t, m, "down")
	if len(reader.calls) != before {
		t.Fatalf("prompt gestures fetched pages: %d -> %d", before, len(reader.calls))
	}
	if m.data.bufferPattern != "m" {
		t.Fatalf("m did not edit the pattern: %q", m.data.bufferPattern)
	}
	m, _ = press(t, m, "enter")
	m, _ = press(t, m, "esc")
	if m.data.bufferPattern != "" {
		t.Fatalf("Esc pattern = %q", m.data.bufferPattern)
	}
}

func TestFiltersResetOnTabAndObjectChange(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	m, _ = press(t, m, "enter")
	if m.data.bufferPattern == "" {
		t.Fatal("filter did not apply")
	}
	m, _ = press(t, m, "1")
	if m.data.bufferPattern != "" || len(m.data.matches) != 0 {
		t.Fatalf("tab change kept filter: %q %v", m.data.bufferPattern, m.data.matches)
	}
	m, _ = press(t, m, "4")
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	m, _ = press(t, m, "enter")
	m, _ = press(t, m, "shift+tab")
	m, cmd := press(t, m, "down")
	m = settle(t, m, cmd)
	if m.data.bufferPattern != "" || len(m.data.matches) != 0 {
		t.Fatalf("object change kept filter: %q %v", m.data.bufferPattern, m.data.matches)
	}
}

func TestServerFilterAppliesPagesAndRestores(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	if m.data.cursorCol != 1 {
		t.Fatalf("cursor column = %d, want name", m.data.cursorCol)
	}
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	m.data.windowStart = 50
	m.data.rows = m.data.rows[50:]
	m.data.recount()
	m, _ = press(t, m, "ctrl+f")
	if m.data.prompt != promptServer || m.data.serverField != "name" {
		t.Fatalf("prompt=%v field=%q", m.data.prompt, m.data.serverField)
	}
	if !strings.Contains(StripANSI(m.View().Content), "regexp on name") {
		t.Fatalf("field prompt missing:\n%s", StripANSI(m.View().Content))
	}
	m = typeString(t, m, "^row-")
	m, cmd = press(t, m, "enter")
	if m.data.prompt != promptNone {
		t.Fatalf("apply left the prompt open: %v", m.data.prompt)
	}
	if strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner rendered before the filtered page landed")
	}
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	if m.data.serverFilter.Field != "name" || m.data.serverFilter.Pattern != "^row-" || !m.data.serverFilter.CaseInsensitive {
		t.Fatalf("filter = %+v", m.data.serverFilter)
	}
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Offset != 0 || last.Limit != 100 {
		t.Fatalf("apply query = %+v", last)
	}
	if got := m.dataSummary(); got != "loaded 100 matching rows · more available" {
		t.Fatalf("filtered summary = %q", got)
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{
		"Filtered by regexp on name;",
		"100 matching rows",
		"loaded of ~250 total rows (table estimate)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("banner %q missing:\n%s", want, view)
		}
	}

	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	last = reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Offset != 100 || last.Limit != 100 {
		t.Fatalf("page query = %+v", last)
	}
	if got := m.dataSummary(); got != "loaded 200 matching rows · more available" {
		t.Fatalf("paged summary = %q", got)
	}
	m, cmd = press(t, m, "ctrl+a")
	m = settle(t, m, cmd)
	last = reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Offset != 200 {
		t.Fatalf("Ctrl+A query = %+v", last)
	}
	if len(m.data.rows) != 250 {
		t.Fatalf("filtered rows = %d", len(m.data.rows))
	}

	before := len(reader.calls)
	m, _ = press(t, m, "esc")
	if m.data.serverFilter != nil {
		t.Fatal("Esc kept the server filter")
	}
	if len(m.data.rows) != 150 || m.data.windowStart != 50 || !m.data.hasMore {
		t.Fatalf("cache restore rows=%d start=%d more=%v", len(m.data.rows), m.data.windowStart, m.data.hasMore)
	}
	if len(reader.calls) != before {
		t.Fatal("cache restore queried the database")
	}
	if strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner survived Esc")
	}
}

func TestServerFilterInvalidRegexpKeepsViewAndPrompt(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		if query.Filter != nil {
			return domain.DataPage{}, domain.NewError(domain.KindEngine, "data", "invalid regular expression")
		}
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "Alice", Kind: domain.CellText}}},
					{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "Bob", Kind: domain.CellText}}},
				},
			},
			EstimatedTotal: 2,
			EstimateSource: domain.EstimateCatalog,
			OrderKey:       "id",
			OrderKeyKnown:  true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "[")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.err == "" || !strings.Contains(m.data.err, "invalid regular expression") {
		t.Fatalf("error = %q", m.data.err)
	}
	if len(m.data.rows) != 2 {
		t.Fatalf("failed filter changed rows: %d", len(m.data.rows))
	}
	if m.data.prompt != promptServer || m.data.serverPattern != "[" {
		t.Fatalf("prompt=%v pattern=%q", m.data.prompt, m.data.serverPattern)
	}
	if m.data.serverFilter != nil {
		t.Fatal("failed filter stayed active")
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "error: invalid regular expression") {
		t.Fatalf("inline error missing:\n%s", view)
	}
	m = typeString(t, m, "a]")
	if m.data.serverPattern != "[a]" {
		t.Fatalf("pattern = %q, want editable", m.data.serverPattern)
	}
}

func TestServerFilterCancelKeepsRows(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, _ = press(t, m, "enter")
	if !m.data.loading || m.data.serverFilter == nil {
		t.Fatalf("filter fetch did not start: loading=%v filter=%v", m.data.loading, m.data.serverFilter)
	}
	m, _ = press(t, m, "esc")
	if m.data.loading {
		t.Fatal("Esc did not cancel the filter fetch")
	}
	if m.message != "cancelled" {
		t.Fatalf("message = %q", m.message)
	}
	if m.data.serverFilter != nil {
		t.Fatalf("cancelled apply kept the filter: %+v", m.data.serverFilter)
	}
	if len(m.data.rows) != 5 {
		t.Fatalf("cancel changed rows: %d", len(m.data.rows))
	}
}

func TestServerFilterShowsOnlyMatchingRows(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		rows := []domain.Row{
			{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "alice", Kind: domain.CellText}}},
			{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "bob", Kind: domain.CellText}}},
			{Cells: []domain.Cell{{Text: "3", Kind: domain.CellNumber}, {Text: "alicia", Kind: domain.CellText}}},
		}
		if query.Filter != nil {
			pattern := query.Filter.Pattern
			if query.Filter.CaseInsensitive {
				pattern = strings.ToLower(pattern)
			}
			matched := rows[:0]
			for _, row := range rows {
				text := row.Cells[1].Text
				if query.Filter.CaseInsensitive {
					text = strings.ToLower(text)
				}
				if strings.Contains(text, pattern) {
					matched = append(matched, row)
				}
			}
			rows = matched
		}
		return domain.DataPage{
			Grid:           domain.Grid{Columns: []domain.Column{{Name: "id"}, {Name: "name"}}, Rows: rows},
			EstimatedTotal: 3,
			EstimateSource: domain.EstimateCatalog,
			OrderKey:       "id",
			OrderKeyKnown:  true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "ali")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 2 {
		t.Fatalf("filtered rows = %d, want 2", len(m.data.rows))
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "alice") || !strings.Contains(view, "alicia") || strings.Contains(view, "bob") {
		t.Fatalf("filtered view = \n%s", view)
	}
	if !strings.Contains(view, "2 matching rows") {
		t.Fatalf("banner count missing:\n%s", view)
	}
}

func TestFilterSurfacesAreMutuallyExclusive(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	m, _ = press(t, m, "enter")
	if m.data.bufferPattern == "" {
		t.Fatal("buffer filter did not apply")
	}
	m, _ = press(t, m, "ctrl+f")
	if m.data.bufferPattern != "" || len(m.data.matches) != 0 {
		t.Fatalf("Ctrl+F kept the buffer filter: %q %v", m.data.bufferPattern, m.data.matches)
	}
	if m.data.prompt != promptServer {
		t.Fatalf("prompt = %v, want server", m.data.prompt)
	}
	m, _ = press(t, m, "esc")

	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("server filter did not apply: %q", m.data.err)
	}
	m, _ = press(t, m, "/")
	if m.data.serverFilter != nil {
		t.Fatal("/ kept the server filter")
	}
	if m.data.prompt != promptBuffer {
		t.Fatalf("prompt = %v, want buffer", m.data.prompt)
	}
	if len(m.data.rows) != 100 || m.data.windowStart != 0 {
		t.Fatalf("cache not restored: rows=%d start=%d", len(m.data.rows), m.data.windowStart)
	}
}

func TestServerFilterResetsOnObjectChange(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	m, _ = press(t, m, "shift+tab")
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	if m.data.serverFilter != nil || m.data.serverCached != nil {
		t.Fatalf("object change kept the server filter: %+v", m.data.serverFilter)
	}
	if strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner survived the object change")
	}
}

func TestServerFilterNoCacheEscReloads(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	m.data.serverCached = nil
	m, cmd = press(t, m, "esc")
	if cmd == nil {
		t.Fatal("no-cache Esc did not reload")
	}
	m = settle(t, m, cmd)
	if m.data.serverFilter != nil || m.data.filterClearing {
		t.Fatalf("filter=%+v clearing=%v", m.data.serverFilter, m.data.filterClearing)
	}
	if len(m.data.rows) != 5 || m.data.err != "" {
		t.Fatalf("reloaded rows=%d err=%q", len(m.data.rows), m.data.err)
	}
	if strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner survived the reload")
	}
}

func TestServerFilterReloadFailureKeepsFilteredView(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	filteredRows := len(m.data.rows)
	m.data.serverCached = nil // force the reload branch of Esc
	reader.failAt = map[int]error{0: domain.NewError(domain.KindEngine, "data", "reload failed")}
	m, cmd = press(t, m, "esc")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatal("failed reload dropped the filter")
	}
	if len(m.data.rows) != filteredRows {
		t.Fatalf("failed reload changed rows: %d -> %d", filteredRows, len(m.data.rows))
	}
	if !strings.Contains(m.data.err, "reload failed") {
		t.Fatalf("error = %q", m.data.err)
	}
	if !strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner disappeared on a failed reload")
	}
}

func TestServerFilterCaseToggleRefetchesFromZero(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "^row-")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 200 {
		t.Fatalf("paged rows = %d", len(m.data.rows))
	}
	m.data.windowStart = 50
	m.data.rows = m.data.rows[50:]
	m.data.recount()

	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "i")
	m = settle(t, m, cmd)
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Filter.CaseInsensitive || last.Offset != 0 {
		t.Fatalf("sensitive refetch = %+v", last.Filter)
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: sensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "i")
	m = settle(t, m, cmd)
	last = reader.calls[len(reader.calls)-1]
	if last.Filter == nil || !last.Filter.CaseInsensitive {
		t.Fatalf("insensitive refetch = %+v", last.Filter)
	}
}

func TestBufferFilterCaseToggleRecomputes(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "Alice", Kind: domain.CellText}}},
					{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "alice", Kind: domain.CellText}}},
					{Cells: []domain.Cell{{Text: "3", Kind: domain.CellNumber}, {Text: "Bob", Kind: domain.CellText}}},
				},
			},
			OrderKey:      "id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "Alice")
	if len(m.data.matches) != 2 {
		t.Fatalf("insensitive matches = %v", m.data.matches)
	}
	m, _ = press(t, m, "enter")
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "i")
	if len(m.data.matches) != 1 || m.data.matches[0] != 0 {
		t.Fatalf("sensitive matches = %v", m.data.matches)
	}
	if !strings.Contains(StripANSI(m.View().Content), "match 1 of 1") {
		t.Fatalf("counter after case toggle:\n%s", StripANSI(m.View().Content))
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: sensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
}

func TestServerFilterSQLJSONUsesBaseColumn(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "payload"}},
				Rows: []domain.Row{{
					Cells: []domain.Cell{
						{Text: "1", Kind: domain.CellNumber},
						{Text: `{"a":1,"b":"x"}`, Kind: domain.CellJSON},
					},
				}},
			},
			OrderKey:      "id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m.data.cursorCol = 1
	m, _ = press(t, m, "`")
	if got := strings.Join(viewNames(m.data.view), ","); got != "id,payload.a,payload.b" {
		t.Fatalf("expanded columns = %q", got)
	}
	m, _ = press(t, m, "ctrl+f")
	if m.data.serverField != "payload" {
		t.Fatalf("SQL filter field = %q, want base column", m.data.serverField)
	}
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Filter.Field != "payload" {
		t.Fatalf("SQL filter query = %+v", last.Filter)
	}
}

func TestServerFilterMongoJSONUsesDottedPath(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "_id"}, {Name: "profile"}},
				Rows: []domain.Row{{
					Cells: []domain.Cell{
						{Text: "1", Kind: domain.CellNumber},
						{Text: `{"email":"alice@example.test","name":"Alice"}`, Kind: domain.CellJSON},
					},
				}},
			},
			OrderKey:      "_id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) { e.connector.data = reader })
	m.data.cursorCol = 1
	m, _ = press(t, m, "`")
	if got := strings.Join(viewNames(m.data.view), ","); got != "_id,profile.email,profile.name" {
		t.Fatalf("expanded columns = %q", got)
	}
	m, _ = press(t, m, "ctrl+f")
	if m.data.serverField != "profile.email" {
		t.Fatalf("Mongo filter field = %q, want dotted path", m.data.serverField)
	}
	m = typeString(t, m, "ali")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Filter.Field != "profile.email" {
		t.Fatalf("Mongo filter query = %+v", last.Filter)
	}
}

func TestServerFilterRejectsInvalidMongoField(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "_id"}, {Name: "$bad"}},
				Rows: []domain.Row{{
					Cells: []domain.Cell{
						{Text: "1", Kind: domain.CellNumber},
						{Text: "x", Kind: domain.CellText},
					},
				}},
			},
			OrderKey:      "_id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) { e.connector.data = reader })
	m.data.cursorCol = 1
	before := len(reader.calls)
	m, _ = press(t, m, "ctrl+f")
	if m.data.prompt != promptNone {
		t.Fatalf("invalid field opened the prompt: %v", m.data.prompt)
	}
	if !strings.Contains(m.message, "cannot be filtered") {
		t.Fatalf("message = %q", m.message)
	}
	if !strings.Contains(StripANSI(m.View().Content), "error: the focused field cannot be filtered") {
		t.Fatalf("inline error missing:\n%s", StripANSI(m.View().Content))
	}
	if len(reader.calls) != before {
		t.Fatal("invalid field queried the database")
	}
}

func TestServerFilterNewKeyDoesNotReconnect(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "n")
	if m.modal != modalNone || m.screen != ScreenWorkspace {
		t.Fatalf("n opened a new connection: modal=%v screen=%v", m.modal, m.screen)
	}
}

func TestFilterBarKeeps80x24Shape(t *testing.T) {
	assertShape := func(t *testing.T, m Model) {
		t.Helper()
		view := m.View().Content
		lines := strings.Split(view, "\n")
		if len(lines) != 24 {
			t.Fatalf("view height = %d, want 24", len(lines))
		}
		for _, line := range lines {
			if lipgloss.Width(line) > 80 {
				t.Fatalf("line exceeds 80 cells: %q", StripANSI(line))
			}
		}
	}

	t.Run("bar", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		assertShape(t, m)
	})

	t.Run("no match", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "zzz")
		assertShape(t, m)
	})

	t.Run("server banner", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
		m, _ = press(t, m, "right")
		m, _ = press(t, m, "ctrl+f")
		m = typeString(t, m, "^row-")
		m, cmd := press(t, m, "enter")
		m = settle(t, m, cmd)
		assertShape(t, m)
	})
}

func TestCaseChordWithOpenFilterPrompt(t *testing.T) {
	t.Run("buffer", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		if !strings.Contains(StripANSI(m.View().Content), "match 1 of 2") {
			t.Fatalf("initial counter missing:\n%s", StripANSI(m.View().Content))
		}
		m, _ = press(t, m, "ctrl+x")
		m, _ = press(t, m, "i")
		if m.data.prompt != promptBuffer || m.data.bufferPattern != "active" {
			t.Fatalf("prompt=%v pattern=%q", m.data.prompt, m.data.bufferPattern)
		}
		if m.caseInsensitive {
			t.Fatal("Ctrl+X i did not toggle case")
		}
		if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: sensitive") {
			t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
		}
	})
	t.Run("server", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "right")
		m, _ = press(t, m, "ctrl+f")
		m = typeString(t, m, "^row-")
		m, _ = press(t, m, "ctrl+x")
		m, _ = press(t, m, "i")
		if m.data.prompt != promptServer || m.data.serverPattern != "^row-" {
			t.Fatalf("prompt=%v pattern=%q", m.data.prompt, m.data.serverPattern)
		}
		if m.caseInsensitive {
			t.Fatal("Ctrl+X i did not toggle case")
		}
	})
}

func TestBufferPromptEscClearsDuringFetch(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	reader.block = make(chan struct{})
	m, _ = press(t, m, "m")
	if !m.data.loading {
		t.Fatal("page fetch did not start")
	}
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	if m.data.prompt != promptBuffer || m.data.bufferPattern != "active" {
		t.Fatalf("prompt=%v pattern=%q", m.data.prompt, m.data.bufferPattern)
	}
	m, _ = press(t, m, "esc")
	if m.data.loading {
		t.Fatal("Esc did not cancel the page fetch")
	}
	if m.data.prompt != promptNone || m.data.bufferPattern != "" {
		t.Fatalf("Esc left prompt=%v pattern=%q", m.data.prompt, m.data.bufferPattern)
	}
}

func TestServerPromptClosesWhileApplying(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, _ = press(t, m, "enter")
	if !m.data.loading || m.data.prompt != promptNone {
		t.Fatalf("loading=%v prompt=%v", m.data.loading, m.data.prompt)
	}
	m, _ = press(t, m, "y")
	if m.data.serverPattern != "x" {
		t.Fatalf("keystroke edited the applied pattern: %q", m.data.serverPattern)
	}
}

func TestServerPromptEnterDuringFetchIsNotSilent(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "m")
	if !m.data.loading {
		t.Fatal("page fetch did not start")
	}
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, _ = press(t, m, "enter")
	if !strings.Contains(m.message, "wait for the current fetch") {
		t.Fatalf("message = %q", m.message)
	}
	if m.data.serverFilter != nil {
		t.Fatal("Enter started a filter while a fetch was loading")
	}
}

func TestServerFilterZeroRowsShowsFilteredEmpty(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		if query.Filter != nil {
			return domain.DataPage{
				Grid:           domain.Grid{Columns: []domain.Column{{Name: "id"}, {Name: "name"}}},
				EstimatedTotal: 2,
				EstimateSource: domain.EstimateCatalog,
				OrderKey:       "id",
				OrderKeyKnown:  true,
			}, nil
		}
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "Alice", Kind: domain.CellText}}},
					{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "Bob", Kind: domain.CellText}}},
				},
			},
			EstimatedTotal: 2,
			EstimateSource: domain.EstimateCatalog,
			OrderKey:       "id",
			OrderKeyKnown:  true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "^zzz")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "No rows match the regexp on name.") {
		t.Fatalf("filtered empty copy missing:\n%s", view)
	}
	if strings.Contains(view, "is empty.") {
		t.Fatalf("object-empty copy leaked into the filtered state:\n%s", view)
	}
	if got := m.dataSummary(); got != "0 matching rows loaded" {
		t.Fatalf("summary = %q", got)
	}
	if !strings.Contains(view, "0 matching rows loaded") {
		t.Fatalf("banner count missing:\n%s", view)
	}
}

func TestServerFilterZeroRowsWithoutColumns(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		if query.Filter != nil {
			return domain.DataPage{OrderKey: "id", OrderKeyKnown: true}, nil
		}
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}},
				Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
			},
			OrderKey:      "id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "zzz")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "No rows match the regexp on id.") {
		t.Fatalf("filtered empty copy missing:\n%s", view)
	}
	if strings.Contains(view, "is empty.") {
		t.Fatalf("object-empty copy leaked:\n%s", view)
	}
}

func TestCaseToggleFailureRestoresFilter(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		if query.Filter != nil && !query.Filter.CaseInsensitive {
			return domain.DataPage{}, domain.NewError(domain.KindEngine, "data", "case refetch failed")
		}
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "id"}, {Name: "name"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "Alice", Kind: domain.CellText}}},
					{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "alice", Kind: domain.CellText}}},
				},
			},
			OrderKey:      "id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "ali")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil || len(m.data.rows) != 2 {
		t.Fatalf("setup filter=%+v rows=%d", m.data.serverFilter, len(m.data.rows))
	}
	m, _ = press(t, m, "ctrl+x")
	m, cmd = press(t, m, "i")
	m = settle(t, m, cmd)
	if !m.caseInsensitive {
		t.Fatal("failed toggle left case mode sensitive")
	}
	if m.data.serverFilter == nil || !m.data.serverFilter.CaseInsensitive {
		t.Fatalf("failed toggle left filter = %+v", m.data.serverFilter)
	}
	if len(m.data.rows) != 2 {
		t.Fatalf("failed toggle changed rows: %d", len(m.data.rows))
	}
	if !strings.Contains(m.data.err, "case refetch failed") {
		t.Fatalf("error = %q", m.data.err)
	}
	if !strings.Contains(StripANSI(m.viewTopStatus(80)), "case: insensitive") {
		t.Fatalf("top status = %q", StripANSI(m.viewTopStatus(80)))
	}
}

func TestServerFilterEmptyPatternClears(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	m, _ = press(t, m, "ctrl+f")
	m.data.serverPattern = ""
	m, _ = press(t, m, "enter")
	if m.data.serverFilter != nil || m.data.prompt != promptNone {
		t.Fatalf("empty pattern filter=%+v prompt=%v", m.data.serverFilter, m.data.prompt)
	}
	if len(m.data.rows) != 5 {
		t.Fatalf("empty pattern did not restore rows: %d", len(m.data.rows))
	}
	if strings.Contains(StripANSI(m.View().Content), "Filtered by regexp") {
		t.Fatal("banner survived the cleared filter")
	}
}

func TestFilterPatternPreservesSpacesAndModifiers(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "/")
	m, _ = press(t, m, "space")
	m = typeString(t, m, "ali")
	m, _ = press(t, m, "space")
	if m.data.bufferPattern != " ali " {
		t.Fatalf("pattern = %q", m.data.bufferPattern)
	}
	if !strings.Contains(StripANSI(m.renderFilterBar(60)), " ali ") {
		t.Fatalf("bar trimmed the pattern: %q", StripANSI(m.renderFilterBar(60)))
	}
	next, _ := m.Update(tea.KeyPressMsg{Text: "A", Code: 'A', Mod: tea.ModShift})
	m = next.(Model)
	if !strings.HasSuffix(m.data.bufferPattern, "A") {
		t.Fatalf("Shift key not inserted: %q", m.data.bufferPattern)
	}
	next, _ = m.Update(tea.KeyPressMsg{Text: "ą", Code: 'ą', Mod: tea.ModCtrl | tea.ModAlt})
	m = next.(Model)
	if !strings.HasSuffix(m.data.bufferPattern, "ą") {
		t.Fatalf("AltGr key not inserted: %q", m.data.bufferPattern)
	}
	m.data.bufferPattern = "zzz "
	if !strings.Contains(StripANSI(m.renderNoMatchLine(80)), `zzz `) {
		t.Fatalf("no-match copy trimmed the pattern: %q", StripANSI(m.renderNoMatchLine(80)))
	}
}

func TestFilterPasteSanitizesBothPrompts(t *testing.T) {
	payload := "  a\x1b[31ml\tb\x01  "
	want := "  alb  "
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "/")
	next, _ := m.Update(tea.PasteMsg{Content: payload})
	m = next.(Model)
	if m.data.bufferPattern != want {
		t.Fatalf("buffer paste = %q, want %q", m.data.bufferPattern, want)
	}
	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "ctrl+f")
	next, _ = m.Update(tea.PasteMsg{Content: payload})
	m = next.(Model)
	if m.data.serverPattern != want {
		t.Fatalf("server paste = %q, want %q", m.data.serverPattern, want)
	}
}

func TestServerFilterMatchNavigation(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 1 {
		t.Fatalf("n cursor = %d, want 1", m.data.cursorRow)
	}
	m, _ = press(t, m, "N")
	if m.data.cursorRow != 0 {
		t.Fatalf("N cursor = %d, want 0", m.data.cursorRow)
	}
	m, _ = press(t, m, "N")
	if m.data.cursorRow != 4 {
		t.Fatalf("N wrap cursor = %d, want 4", m.data.cursorRow)
	}
	m, _ = press(t, m, "n")
	if m.data.cursorRow != 0 {
		t.Fatalf("n wrap cursor = %d, want 0", m.data.cursorRow)
	}
}

func TestBeginDataFetchCopiesFilter(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "m")
	if cmd == nil {
		t.Fatal("page fetch did not start")
	}
	m.data.serverFilter.CaseInsensitive = false
	cmd()
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || !last.Filter.CaseInsensitive {
		t.Fatalf("command copied a mutated filter: %+v", last.Filter)
	}
}

func TestBufferFilterWithSlidWindow(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m.data.windowStart = 2
	m.data.rows = m.data.rows[2:]
	m.data.recount()
	m, _ = press(t, m, "/")
	m = typeString(t, m, "row-00")
	if len(m.data.matches) != 3 {
		t.Fatalf("matches = %v", m.data.matches)
	}
	if m.data.dataRowAt(0) != 2 {
		t.Fatalf("first absolute row = %d, want 2", m.data.dataRowAt(0))
	}
	if m.data.cursorRow != 2 || m.data.cursorCol != 1 {
		t.Fatalf("cursor = %d/%d, want 2/1", m.data.cursorRow, m.data.cursorCol)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "row-003") || strings.Contains(view, "row-001") {
		t.Fatalf("slid filtered view:\n%s", view)
	}
}

func TestFilterPromptPassThroughKeys(t *testing.T) {
	t.Run("tab", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		m, _ = press(t, m, "tab")
		if m.focus != FocusQuery || m.data.prompt != promptNone || m.data.bufferPattern != "active" {
			t.Fatalf("focus=%v prompt=%v pattern=%q", m.focus, m.data.prompt, m.data.bufferPattern)
		}
	})
	t.Run("shift-tab", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		m, _ = press(t, m, "shift+tab")
		if m.focus != FocusTree || m.data.prompt != promptNone {
			t.Fatalf("focus=%v prompt=%v", m.focus, m.data.prompt)
		}
	})
	t.Run("disconnect", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		m, _ = press(t, m, "ctrl+d")
		if m.screen != ScreenPicker {
			t.Fatalf("screen = %v, want picker", m.screen)
		}
	})
	t.Run("f1", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "/")
		m = typeString(t, m, "active")
		m, _ = press(t, m, "f1")
		if m.screen != ScreenWorkspace || !strings.Contains(m.message, "help") {
			t.Fatalf("message = %q", m.message)
		}
	})
	t.Run("server tab", func(t *testing.T) {
		m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
		m, _ = press(t, m, "ctrl+f")
		m = typeString(t, m, "x")
		m, _ = press(t, m, "tab")
		if m.focus != FocusQuery || m.data.prompt != promptNone || m.data.serverFilter != nil {
			t.Fatalf("focus=%v prompt=%v filter=%+v", m.focus, m.data.prompt, m.data.serverFilter)
		}
	})
}

func TestBufferPromptAfterNoCacheServerFilter(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m, _ = press(t, m, "ctrl+f")
	m = typeString(t, m, "x")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if m.data.serverFilter == nil {
		t.Fatalf("filter not applied: %q", m.data.err)
	}
	m.data.serverCached = nil
	m, cmd = press(t, m, "/")
	if m.data.prompt != promptBuffer {
		t.Fatalf("prompt = %v, want buffer", m.data.prompt)
	}
	if m.data.serverFilter != nil {
		t.Fatal("server filter survived /")
	}
	if cmd == nil {
		t.Fatal("no reload command returned")
	}
	m = settle(t, m, cmd)
	if m.data.prompt != promptBuffer {
		t.Fatalf("reload closed the buffer prompt: %v", m.data.prompt)
	}
	if len(m.data.rows) != 5 {
		t.Fatalf("reload rows = %d", len(m.data.rows))
	}
}

func TestServerFilterResetsOnTabsAndCycle(t *testing.T) {
	apply := func(t *testing.T, m Model) Model {
		t.Helper()
		m, _ = press(t, m, "ctrl+f")
		m = typeString(t, m, "x")
		var cmd tea.Cmd
		m, cmd = press(t, m, "enter")
		m = settle(t, m, cmd)
		if m.data.serverFilter == nil {
			t.Fatalf("filter not applied: %q", m.data.err)
		}
		return m
	}
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "right")
	m = apply(t, m)
	m, _ = press(t, m, "1")
	if m.data.serverFilter != nil {
		t.Fatal("selectTab(1) kept the server filter")
	}
	m, _ = press(t, m, "4")
	m = apply(t, m)
	m, _ = press(t, m, "]")
	if m.data.serverFilter != nil {
		t.Fatal("cycleTab(]) kept the server filter")
	}
	m, _ = press(t, m, "4")
	m = apply(t, m)
	m, _ = press(t, m, "[")
	if m.data.serverFilter != nil {
		t.Fatal("cycleTab([) kept the server filter")
	}
}

func TestBufferFilterArrowPageRangeNavigation(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "active")
	m, _ = press(t, m, "enter")
	before := len(reader.calls)
	m, _ = press(t, m, "down")
	if m.data.cursorRow != 3 {
		t.Fatalf("down cursor = %d, want 3", m.data.cursorRow)
	}
	m, _ = press(t, m, "down")
	if m.data.cursorRow != 3 || m.message != "end of data" {
		t.Fatalf("down clamp cursor=%d message=%q", m.data.cursorRow, m.message)
	}
	m, _ = press(t, m, "up")
	if m.data.cursorRow != 0 {
		t.Fatalf("up cursor = %d, want 0", m.data.cursorRow)
	}
	m, _ = press(t, m, "pgdown")
	if m.data.cursorRow != 3 {
		t.Fatalf("pgdown cursor = %d, want 3", m.data.cursorRow)
	}
	m, _ = press(t, m, "pgup")
	if m.data.cursorRow != 0 {
		t.Fatalf("pgup cursor = %d, want 0", m.data.cursorRow)
	}
	m, _ = press(t, m, "ctrl+end")
	if m.data.cursorRow != 3 {
		t.Fatalf("ctrl+end cursor = %d, want 3", m.data.cursorRow)
	}
	m, _ = press(t, m, "ctrl+home")
	if m.data.cursorRow != 0 {
		t.Fatalf("ctrl+home cursor = %d, want 0", m.data.cursorRow)
	}
	if len(reader.calls) != before {
		t.Fatal("filtered navigation queried the database")
	}
}

func TestDataRowProjectionTable(t *testing.T) {
	rows := make([]domain.Row, 5)
	d := dataState{rows: rows, bufferPattern: "x", matches: []int{0, 2, 4}, windowStart: 10}
	if got := d.dataRowCount(); got != 3 {
		t.Fatalf("filtered count = %d", got)
	}
	for position, want := range []int{10, 12, 14} {
		if got := d.dataRowAt(position); got != want {
			t.Fatalf("dataRowAt(%d) = %d, want %d", position, got, want)
		}
	}
	if got := d.dataRowPosition(12); got != 1 {
		t.Fatalf("dataRowPosition(12) = %d", got)
	}
	if got := d.dataRowPosition(11); got != -1 {
		t.Fatalf("dataRowPosition(11) = %d, want -1", got)
	}
	if got := d.dataRowPositionAtOrAfter(11); got != 1 {
		t.Fatalf("dataRowPositionAtOrAfter(11) = %d, want 1", got)
	}
	if got := d.dataRowPositionAtOrAfter(15); got != 3 {
		t.Fatalf("dataRowPositionAtOrAfter(15) = %d, want 3", got)
	}

	all := dataState{rows: rows}
	if got := all.dataRowCount(); got != 5 {
		t.Fatalf("unfiltered count = %d", got)
	}
	if got := all.dataRowAt(4); got != 4 {
		t.Fatalf("unfiltered dataRowAt(4) = %d", got)
	}
	if got := all.dataRowPositionAtOrAfter(6); got != 5 {
		t.Fatalf("unfiltered at-or-after end = %d, want 5", got)
	}
}

func TestMatchRangesFoldAndRepeats(t *testing.T) {
	ranges := matchRanges("Alice alice ALICE", "ali", true)
	want := [][2]int{{0, 3}, {6, 9}, {12, 15}}
	if len(ranges) != len(want) {
		t.Fatalf("ranges = %v, want %v", ranges, want)
	}
	for i := range want {
		if ranges[i] != want[i] {
			t.Fatalf("ranges = %v, want %v", ranges, want)
		}
	}
	if got := matchRanges("Σσς", "σ", true); len(got) != 3 {
		t.Fatalf("EqualFold sigma ranges = %v", got)
	}
	if got := matchRanges("ABCabc", "abc", false); len(got) != 1 || got[0] != [2]int{3, 6} {
		t.Fatalf("sensitive ranges = %v", got)
	}
	base := lipgloss.NewStyle()
	mark := lipgloss.NewStyle().Underline(true)
	rendered := highlightMatches("ali x ali", "ali", true, base, mark)
	if got := strings.Count(rendered, mark.Render("ali")); got != 2 {
		t.Fatalf("repeated highlight count = %d\n%q", got, rendered)
	}
}

func TestServerFilterNonTextField(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	if m.data.cursorCol != 0 {
		t.Fatalf("cursor column = %d, want id", m.data.cursorCol)
	}
	m, _ = press(t, m, "ctrl+f")
	if m.data.serverField != "id" {
		t.Fatalf("field = %q, want id", m.data.serverField)
	}
	m = typeString(t, m, "1")
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	last := reader.calls[len(reader.calls)-1]
	if last.Filter == nil || last.Filter.Field != "id" {
		t.Fatalf("filter = %+v", last.Filter)
	}
}

func TestBufferFilterEmptyTable(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 0})
	m, _ = press(t, m, "/")
	m = typeString(t, m, "x")
	if m.data.prompt != promptBuffer {
		t.Fatalf("prompt = %v", m.data.prompt)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, `No matches for "x". Esc clears the filter.`) {
		t.Fatalf("no-match copy missing:\n%s", view)
	}
	if got := m.dataSummary(); got != "0 of 0 loaded" {
		t.Fatalf("summary = %q", got)
	}
}

func TestFilterNoMatchHiddenWhenTooLarge(t *testing.T) {
	d := dataState{bufferPattern: "x", tooLarge: true}
	if d.filterNoMatchVisible() {
		t.Fatal("tooLarge state must not show the no-match line")
	}
	d.tooLarge = false
	if !d.filterNoMatchVisible() {
		t.Fatal("normal zero-match state must show the no-match line")
	}
}

func TestGoldenDataFilterStates(t *testing.T) {
	filtered, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	filtered, _ = press(t, filtered, "/")
	filtered = typeString(t, filtered, "active")
	filtered, _ = press(t, filtered, "enter")
	assertGolden(t, "workspace-data-filter.golden", filtered.View().Content)

	noMatch, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	noMatch, _ = press(t, noMatch, "/")
	noMatch = typeString(t, noMatch, "zzz")
	assertGolden(t, "workspace-data-no-match.golden", noMatch.View().Content)

	server, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	server, _ = press(t, server, "right")
	server, _ = press(t, server, "ctrl+f")
	server = typeString(t, server, "^row-")
	server, cmd := press(t, server, "enter")
	server = settle(t, server, cmd)
	assertGolden(t, "workspace-data-server-filter.golden", server.View().Content)
}
