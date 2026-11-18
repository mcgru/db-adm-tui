package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// dataWorkspaceWith connects the named profile, expands to the first table or
// collection, loads its first page, and focuses the Data tab.
func dataWorkspaceWith(t *testing.T, label string, setup func(*testEnv)) (Model, *testEnv) {
	t.Helper()
	e := newTestEnv(t)
	seedStandard(t, e)
	if setup != nil {
		setup(e)
	}
	m := selectLabel(t, e.model(t), label)
	m = connectSelected(t, m)
	if m.screen != ScreenWorkspace {
		t.Fatalf("not connected: %q", m.message)
	}
	m = selectKind(t, m, domain.KindDatabase)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	if m.engine() == domain.EnginePostgreSQL {
		m = selectKind(t, m, domain.KindSchema)
		m, cmd = press(t, m, "right")
		m = settle(t, m, cmd)
	}
	kind := domain.KindTable
	if m.engine() == domain.EngineMongoDB {
		kind = domain.KindCollection
	}
	m = selectKind(t, m, kind)
	m, cmd = press(t, m, "enter")
	return settle(t, m, cmd), e
}

func dataWorkspaceWithReader(t *testing.T, reader *fakeDataReader) (Model, *testEnv) {
	t.Helper()
	return dataWorkspaceWith(t, "prod-ro", func(e *testEnv) { e.connector.data = reader })
}

func TestDataInitialPageAndPaging(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	if !m.data.loaded || m.data.loading {
		t.Fatalf("data state loaded=%v loading=%v", m.data.loaded, m.data.loading)
	}
	if m.details.currentTab() != "Data" {
		t.Fatalf("data tab = %d %q", m.details.tab, m.details.currentTab())
	}
	if m.focus != FocusMain {
		t.Fatalf("focus = %v", m.focus)
	}
	if len(m.data.rows) != 100 || m.data.windowStart != 0 || !m.data.hasMore {
		t.Fatalf("initial window rows=%d start=%d more=%v", len(m.data.rows), m.data.windowStart, m.data.hasMore)
	}
	if reader.calls[0].Limit != 100 || reader.calls[0].Offset != 0 {
		t.Fatalf("initial query = %+v", reader.calls[0])
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{"row-001", "loaded 100 of ~250", "id", "name", "status"} {
		if !strings.Contains(view, want) {
			t.Fatalf("data view missing %q:\n%s", want, view)
		}
	}

	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 200 || !m.data.hasMore {
		t.Fatalf("after m rows=%d more=%v", len(m.data.rows), m.data.hasMore)
	}
	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 250 || m.data.hasMore {
		t.Fatalf("after second m rows=%d more=%v", len(m.data.rows), m.data.hasMore)
	}
	if !strings.Contains(StripANSI(m.View().Content), "loaded 250 of ~250") {
		t.Fatalf("status missing final count:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "m")
	if m.message != "end of data" {
		t.Fatalf("m at end = %q", m.message)
	}
}

func TestDataEndGesture(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m.data.cursorRow = 99
	m.ensureDataViewport()

	m, _ = press(t, m, "down")
	if m.message != "press Down twice more" {
		t.Fatalf("first down = %q", m.message)
	}
	m, _ = press(t, m, "down")
	if m.message != "press Down again" {
		t.Fatalf("second down = %q", m.message)
	}
	m, _ = press(t, m, "left")
	if m.data.endDown != 0 {
		t.Fatal("column move did not reset the down gesture")
	}
	m, _ = press(t, m, "down")
	if m.message != "press Down twice more" {
		t.Fatalf("reset down = %q", m.message)
	}
	m, _ = press(t, m, "down")
	m, cmd := press(t, m, "down")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 200 {
		t.Fatalf("end gesture fetched rows=%d", len(m.data.rows))
	}

	m.data.cursorRow = 199
	m.ensureDataViewport()
	m, _ = press(t, m, "pgdown")
	if m.message != "press PgDown again" {
		t.Fatalf("first pgdown = %q", m.message)
	}
	m, cmd = press(t, m, "pgdown")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 250 || m.data.hasMore {
		t.Fatalf("pgdown fetch rows=%d more=%v", len(m.data.rows), m.data.hasMore)
	}
	m.data.cursorRow = 249
	m.ensureDataViewport()
	m, _ = press(t, m, "pgdown")
	if m.message != "end of data" {
		t.Fatalf("pgdown at end = %q", m.message)
	}
}

func TestDataCtrlAAccumulatesWindow(t *testing.T) {
	reader := &fakeDataReader{total: 15000}
	m, _ := dataWorkspaceWithReader(t, reader)
	for i := 0; i < 10; i++ {
		next, cmd := press(t, m, "ctrl+a")
		m = settle(t, next, cmd)
		if m.data.windowStart != 0 {
			t.Fatalf("Ctrl+A %d slid the window to %d", i+1, m.data.windowStart)
		}
	}
	wantRows := 100 + 10*dataFetchAllSize
	if len(m.data.rows) != wantRows {
		t.Fatalf("accumulated rows = %d, want %d", len(m.data.rows), wantRows)
	}
	summary := m.dataSummary()
	for _, want := range []string{"loaded 10,100 of ~15,000", "more available"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary = %q, want %q", summary, want)
		}
	}

	m, cmd := press(t, m, "ctrl+a")
	m = settle(t, m, cmd)
	wantRows += dataFetchAllSize
	if len(m.data.rows) != wantRows || m.data.windowStart != 0 {
		t.Fatalf("grown window start=%d rows=%d, want 0/%d", m.data.windowStart, len(m.data.rows), wantRows)
	}

	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	if m.data.windowStart != dataFetchAllSize || len(m.data.rows) != wantRows {
		t.Fatalf("m after growth start=%d rows=%d", m.data.windowStart, len(m.data.rows))
	}

	m, cmd = press(t, m, "ctrl+home")
	m = settle(t, m, cmd)
	if m.data.windowStart != 0 || m.data.cursorRow != 0 {
		t.Fatalf("ctrl+home start=%d cursor=%d, want 0/0", m.data.windowStart, m.data.cursorRow)
	}
	last := reader.calls[len(reader.calls)-1]
	if last.Offset != 0 || last.Limit != dataFetchAllSize {
		t.Fatalf("prefix fetch = %+v, want offset 0 limit %d", last, dataFetchAllSize)
	}
}

func TestDataCtrlHomeAndEndNavigateLoadedRange(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m, _ = press(t, m, "ctrl+end")
	if m.data.cursorRow != 99 {
		t.Fatalf("ctrl+end cursor = %d, want 99", m.data.cursorRow)
	}
	m, _ = press(t, m, "ctrl+home")
	if m.data.cursorRow != 0 {
		t.Fatalf("ctrl+home cursor = %d, want 0", m.data.cursorRow)
	}
}

func TestDataRowCapKeepsAbsoluteCursorOnPaging(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m.data.maxRows = 3
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(m.data.rows))
	}
	if m.data.windowStart != 197 || m.data.cursorRow != 197 {
		t.Fatalf("start=%d cursor=%d, want 197/197", m.data.windowStart, m.data.cursorRow)
	}
	if !strings.Contains(m.dataSummary(), "rows 198–200 of ~250") {
		t.Fatalf("summary = %q", m.dataSummary())
	}
	gridView := StripANSI(strings.Join(m.renderGrid(m.dataInnerWidth(), 4), "\n"))
	if !strings.Contains(gridView, "198") || !strings.Contains(gridView, "row-198") {
		t.Fatalf("absolute row gutter lost after slide:\n%s", gridView)
	}
}

func TestDataByteCap(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m.data.maxBytes = 200
	m, cmd := press(t, m, "ctrl+a")
	m = settle(t, m, cmd)
	if m.data.windowStart == 0 {
		t.Fatalf("byte cap did not evict rows: bytes=%d rows=%d", m.data.bytes, len(m.data.rows))
	}
	if m.data.bytes > m.data.maxBytes {
		t.Fatalf("bytes = %d, cap = %d", m.data.bytes, m.data.maxBytes)
	}
	if len(m.data.rows) >= 200 {
		t.Fatalf("rows = %d, want fewer than 200", len(m.data.rows))
	}
}

func TestDataFetchErrorKeepsRows(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	reader.failAt = map[int]error{100: domain.NewError(domain.KindEngine, "data", "boom")}
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 100 {
		t.Fatalf("failed page changed rows: %d", len(m.data.rows))
	}
	if !strings.Contains(m.data.err, "boom") {
		t.Fatalf("error = %q", m.data.err)
	}
	if !strings.Contains(StripANSI(m.View().Content), "error: boom") {
		t.Fatalf("inline error missing:\n%s", StripANSI(m.View().Content))
	}
	reader.failAt = nil
	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 200 || m.data.err != "" {
		t.Fatalf("retry rows=%d err=%q", len(m.data.rows), m.data.err)
	}
}

func TestDataInitialErrorThenRetry(t *testing.T) {
	reader := &fakeDataReader{total: 5, failAt: map[int]error{0: domain.NewError(domain.KindPermission, "data", "not permitted")}}
	m, _ := dataWorkspaceWithReader(t, reader)
	if m.data.loaded || m.data.err == "" {
		t.Fatalf("initial error loaded=%v err=%q", m.data.loaded, m.data.err)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "error: not permitted") {
		t.Fatalf("initial error missing:\n%s", view)
	}
	reader.failAt = nil
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	if !m.data.loaded || len(m.data.rows) != 5 {
		t.Fatalf("retry loaded=%v rows=%d", m.data.loaded, len(m.data.rows))
	}
}

func TestDataCancelFetchKeepsRows(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	reader.block = make(chan struct{})
	m, cmd := press(t, m, "m")
	if cmd == nil || !m.data.loading {
		t.Fatalf("fetch did not start: cmd=%v loading=%v", cmd != nil, m.data.loading)
	}
	m, _ = press(t, m, "esc")
	if m.data.loading {
		t.Fatal("Esc did not stop the fetch")
	}
	if m.message != "cancelled" {
		t.Fatalf("message = %q", m.message)
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	if len(m.data.rows) != 100 {
		t.Fatalf("cancel changed rows: %d", len(m.data.rows))
	}
	if m.data.err != "" {
		t.Fatalf("cancel recorded an error: %q", m.data.err)
	}
	reader.block = nil
	m, cmd = press(t, m, "m")
	m = settle(t, m, cmd)
	if len(m.data.rows) != 200 {
		t.Fatalf("recovery rows = %d", len(m.data.rows))
	}
}

func TestDataNoStableOrderWarns(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.catalog = &fakeCatalog{engine: domain.EnginePostgreSQL, noOrder: true}
		e.connector.data = &fakeDataReader{total: 5, noOrder: true}
	})
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "no stable order; rows may repeat or skip") {
		t.Fatalf("order warning missing:\n%s", view)
	}
}

func TestDataMySQLInitialPage(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "billing", func(e *testEnv) {
		e.connector.data = &fakeDataReader{total: 2}
	})
	if !m.data.loaded || len(m.data.rows) != 2 || m.details.currentTab() != "Data" {
		t.Fatalf("mysql data loaded=%v rows=%d tab=%q", m.data.loaded, len(m.data.rows), m.details.currentTab())
	}
	if !strings.Contains(StripANSI(m.View().Content), "row-001") {
		t.Fatalf("mysql data grid missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestDataTotalUnavailableStatus(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5, unavailable: true})
	if got := m.dataSummary(); got != "loaded 5 · total unavailable" {
		t.Fatalf("summary = %q", got)
	}
	if !strings.Contains(StripANSI(m.View().Content), "total unavailable") {
		t.Fatalf("status missing unavailable copy:\n%s", StripANSI(m.View().Content))
	}
}

func TestDataEmptyObjectKeepsHeaders(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 0})
	if !m.data.loaded || len(m.data.rows) != 0 || len(m.data.columns) != 3 {
		t.Fatalf("empty state loaded=%v rows=%d cols=%d", m.data.loaded, len(m.data.rows), len(m.data.columns))
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "No rows. public.customers is empty.") {
		t.Fatalf("empty copy missing:\n%s", view)
	}
	if !strings.Contains(view, "id") || !strings.Contains(view, "status") {
		t.Fatalf("known headers missing:\n%s", view)
	}
	if !strings.Contains(view, "0 rows") {
		t.Fatalf("0 rows status missing:\n%s", view)
	}
}

func TestDataMongoEmptyObjectHasNoColumns(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) {
		e.connector.data = &fakeDataReader{total: 0, emptyColumns: true}
	})
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "No rows. analytics.customers is empty.") {
		t.Fatalf("mongo empty copy missing:\n%s", view)
	}
}

func TestDataMongoUnionAcrossPages(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		if query.Offset == 0 {
			return domain.DataPage{
				Grid: domain.Grid{
					Columns: []domain.Column{{Name: "_id"}, {Name: "name"}},
					Rows: []domain.Row{
						{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "Alice", Kind: domain.CellText}}},
						{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, {Text: "Bob", Kind: domain.CellText}}},
					},
				},
				HasMore:        true,
				EstimatedTotal: 4,
				EstimateSource: domain.EstimateCollection,
			}, nil
		}
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "_id"}, {Name: "name"}, {Name: "age"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{{Text: "3", Kind: domain.CellNumber}, {Text: "Carol", Kind: domain.CellText}, {Text: "30", Kind: domain.CellNumber}}},
					{Cells: []domain.Cell{{Text: "4", Kind: domain.CellNumber}, {Text: "Dan", Kind: domain.CellText}, {Text: "40", Kind: domain.CellNumber}}},
				},
			},
			HasMore:        false,
			EstimatedTotal: 4,
			EstimateSource: domain.EstimateCollection,
		}, nil
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) { e.connector.data = reader })
	if len(m.data.columns) != 2 || len(m.data.rows) != 2 {
		t.Fatalf("initial columns=%d rows=%d", len(m.data.columns), len(m.data.rows))
	}
	m, cmd := press(t, m, "m")
	m = settle(t, m, cmd)
	var names []string
	for _, column := range m.data.columns {
		names = append(names, column.Name)
	}
	if strings.Join(names, ",") != "_id,name,age" {
		t.Fatalf("union columns = %v", names)
	}
	if len(m.data.rows) != 4 {
		t.Fatalf("rows = %d", len(m.data.rows))
	}
	if m.data.rows[0].Cells[2].Text != "" || m.data.rows[0].Cells[2].Null {
		t.Fatalf("missing older field = %+v", m.data.rows[0].Cells[2])
	}
	if m.data.rows[2].Cells[1].Text != "Carol" || m.data.rows[2].Cells[2].Text != "30" {
		t.Fatalf("remapped row = %+v", m.data.rows[2].Cells)
	}
}

func TestDataBacktickExpandsOnlySelectedMongoJSONColumn(t *testing.T) {
	reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
		return domain.DataPage{
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "_id"}, {Name: "profile"}, {Name: "settings"}},
				Rows: []domain.Row{
					{Cells: []domain.Cell{
						{Text: "1", Kind: domain.CellNumber},
						{Text: `{"email":"alice@example.test","name":"Alice"}`, Kind: domain.CellJSON},
						{Text: `{"theme":"dark","alerts":true}`, Kind: domain.CellJSON},
					}},
					{Cells: []domain.Cell{
						{Text: "2", Kind: domain.CellNumber},
						{Text: `{"name":"Bob","email":null}`, Kind: domain.CellJSON},
						{Text: `{"theme":"light"}`, Kind: domain.CellJSON},
					}},
					{Cells: []domain.Cell{
						{Text: "3", Kind: domain.CellNumber},
						{Text: `"not-an-object"`, Kind: domain.CellJSON},
						domain.NullCell(),
					}},
				},
			},
			OrderKey:      "_id",
			OrderKeyKnown: true,
		}, nil
	}}
	m, _ := dataWorkspaceWith(t, "analytics", func(e *testEnv) { e.connector.data = reader })
	if len(m.data.view) != 3 {
		t.Fatalf("initial view = %d columns", len(m.data.view))
	}

	m.data.cursorCol = 1
	m, _ = press(t, m, "`")
	names := viewNames(m.data.view)
	if got := strings.Join(names, ","); got != "_id,profile.email,profile.name,settings" {
		t.Fatalf("profile expansion = %v", got)
	}
	if m.data.cursorCol != 1 {
		t.Fatalf("cursor column = %d, want profile.email", m.data.cursorCol)
	}
	view := StripANSI(m.View().Content)
	for _, want := range []string{"profile.email", "profile.name", "alice@example.test", "Alice", "Bob"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "settings.theme") {
		t.Fatalf("unselected JSON column expanded:\n%s", view)
	}
	if cell := cellForColumn(m.data.rows[1], m.data.view[1]); !cell.Null {
		t.Fatalf("null subfield = %+v", cell)
	}
	if cell := cellForColumn(m.data.rows[2], m.data.view[1]); cell.Null || cell.Text != "" {
		t.Fatalf("non-object subfield = %+v", cell)
	}

	m.data.cursorCol = 3
	m, _ = press(t, m, "`")
	names = viewNames(m.data.view)
	if got := strings.Join(names, ","); got != "_id,profile.email,profile.name,settings.theme,settings.alerts" {
		t.Fatalf("settings expansion = %v", got)
	}
	if m.data.cursorCol != 3 {
		t.Fatalf("settings cursor = %d, want settings.theme", m.data.cursorCol)
	}

	m, _ = press(t, m, "`")
	names = viewNames(m.data.view)
	if got := strings.Join(names, ","); got != "_id,profile.email,profile.name,settings" {
		t.Fatalf("settings collapse = %v", got)
	}
	if m.data.cursorCol != 3 {
		t.Fatalf("settings cursor after collapse = %d, want 3", m.data.cursorCol)
	}

	m.data.cursorCol = 1
	m, _ = press(t, m, "`")
	names = viewNames(m.data.view)
	if got := strings.Join(names, ","); got != "_id,profile,settings" {
		t.Fatalf("profile collapse = %v", got)
	}
	if m.data.cursorCol != 1 {
		t.Fatalf("profile cursor after collapse = %d, want 1", m.data.cursorCol)
	}
}

func viewNames(view []dataColumn) []string {
	names := make([]string, len(view))
	for i, column := range view {
		names[i] = column.name
	}
	return names
}

func TestDataBacktickExpandsSQLJSONColumns(t *testing.T) {
	for _, tc := range []struct {
		label string
		kind  domain.CellKind
	}{
		{"prod-ro", domain.CellJSON},
		{"billing", domain.CellText},
	} {
		t.Run(tc.label, func(t *testing.T) {
			reader := &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
				return domain.DataPage{
					Grid: domain.Grid{
						Columns: []domain.Column{{Name: "id"}, {Name: "payload"}},
						Rows: []domain.Row{{
							Cells: []domain.Cell{
								{Text: "1", Kind: domain.CellNumber},
								{Text: `{"a":1,"b":"x"}`, Kind: tc.kind},
							},
						}},
					},
					OrderKey:      "id",
					OrderKeyKnown: true,
				}, nil
			}}
			m, _ := dataWorkspaceWith(t, tc.label, func(e *testEnv) { e.connector.data = reader })
			m.data.cursorCol = 1
			m, _ = press(t, m, "`")
			if got := strings.Join(viewNames(m.data.view), ","); got != "id,payload.a,payload.b" {
				t.Fatalf("expanded columns = %q", got)
			}
			view := StripANSI(m.View().Content)
			if !strings.Contains(view, "payload.a") || !strings.Contains(view, "payload.b") {
				t.Fatalf("expanded SQL JSON missing:\n%s", view)
			}
		})
	}
}

func TestDataBacktickWithoutJSONFields(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "`")
	if len(m.data.expandedJSON) != 0 {
		t.Fatal("plain text must not expand")
	}
	if m.message != "selected column has no JSON object fields" {
		t.Fatalf("message = %q", m.message)
	}
}

func TestDataGridNavigation(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5})
	m, _ = press(t, m, "down")
	if m.data.cursorRow != 1 {
		t.Fatalf("down cursor = %d", m.data.cursorRow)
	}
	m, _ = press(t, m, "right")
	if m.data.cursorCol != 1 {
		t.Fatalf("right column = %d", m.data.cursorCol)
	}
	m, _ = press(t, m, "end")
	if m.data.cursorCol != 2 {
		t.Fatalf("end column = %d", m.data.cursorCol)
	}
	m, _ = press(t, m, "home")
	if m.data.cursorCol != 0 {
		t.Fatalf("home column = %d", m.data.cursorCol)
	}
	m, _ = press(t, m, "pgdown")
	if m.data.cursorRow != 4 {
		t.Fatalf("pgdown cursor = %d", m.data.cursorRow)
	}
	m, _ = press(t, m, "pgup")
	if m.data.cursorRow != 0 {
		t.Fatalf("pgup cursor = %d", m.data.cursorRow)
	}
	m, _ = press(t, m, "left")
	if m.data.cursorCol != 0 {
		t.Fatalf("left column = %d", m.data.cursorCol)
	}
}

func TestDataHorizontalScrollKeepsCursorVisible(t *testing.T) {
	columns := make([]domain.Column, 0, 20)
	for i := 1; i <= 20; i++ {
		columns = append(columns, domain.Column{Name: "column-" + formatCount(int64(i))})
	}
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 5, columns: columns})
	m, _ = press(t, m, "end")
	if m.data.cursorCol != 19 {
		t.Fatalf("cursor column = %d", m.data.cursorCol)
	}
	if m.data.colOffset == 0 {
		t.Fatal("horizontal offset did not follow the cursor")
	}
	if !strings.Contains(StripANSI(m.View().Content), "column-20") {
		t.Fatalf("cursor column not visible:\n%s", StripANSI(m.View().Content))
	}
	m, _ = press(t, m, "home")
	if m.data.cursorCol != 0 || m.data.colOffset != 0 {
		t.Fatalf("home did not return: col=%d offset=%d", m.data.cursorCol, m.data.colOffset)
	}
}

func TestDataNullAndEmptyCells(t *testing.T) {
	m, _ := dataWorkspaceWith(t, "prod-ro", func(e *testEnv) {
		e.connector.data = &fakeDataReader{pageFn: func(query domain.DataQuery) (domain.DataPage, error) {
			return domain.DataPage{
				Grid: domain.Grid{
					Columns: []domain.Column{{Name: "id"}, {Name: "note"}},
					Rows: []domain.Row{
						{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}, {Text: "", Kind: domain.CellText}}},
						{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}, domain.NullCell()}},
					},
				},
				EstimatedTotal: 2,
				EstimateSource: domain.EstimateCatalog,
			}, nil
		}}
	})
	if m.data.rows[0].Cells[1].Null || m.data.rows[0].Cells[1].Text != "" {
		t.Fatalf("empty cell = %+v", m.data.rows[0].Cells[1])
	}
	if !m.data.rows[1].Cells[1].Null {
		t.Fatalf("null cell = %+v", m.data.rows[1].Cells[1])
	}
	if !strings.Contains(StripANSI(m.View().Content), "null") {
		t.Fatalf("null word missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestDataSearchPlaceholdersDoNotQuery(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	before := len(reader.calls)
	m, _ = press(t, m, "/")
	if m.message != "search lands in the next story" {
		t.Fatalf("/ message = %q", m.message)
	}
	if len(reader.calls) != before {
		t.Fatal("/ queried the database")
	}
	m, _ = press(t, m, "ctrl+f")
	if m.message != "search lands in the next story" {
		t.Fatalf("ctrl+f message = %q", m.message)
	}
	if len(reader.calls) != before {
		t.Fatal("Ctrl+F queried the database")
	}
	if !strings.Contains(StripANSI(m.View().Content), "search lands in the next story") {
		t.Fatal("placeholder not rendered")
	}
}

func TestDataClearsOnNonDataObject(t *testing.T) {
	reader := &fakeDataReader{total: 5}
	m, _ := dataWorkspaceWithReader(t, reader)
	before := len(reader.calls)
	m = selectKind(t, m, domain.KindView)
	if m.data.has || m.data.loaded || len(m.data.rows) != 0 || len(m.data.columns) != 0 {
		t.Fatalf("data state survived view selection: %+v", m.data)
	}
	if len(reader.calls) != before {
		t.Fatal("selecting a view queried data")
	}
}

func TestDataModalKeysDoNotTriggerGesture(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m.data.cursorRow = 99
	m.data.endDown = 2
	m.modal = modalSave
	m, _ = press(t, m, "down")
	if m.data.endDown != 0 {
		t.Fatal("modal key did not reset the end gesture")
	}
	m.modal = modalNone
}

func TestDataFetchIgnoresStaleGeneration(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 250})
	m.data.cursorRow = 99
	m, _ = press(t, m, "down")
	m.dataGen++
	next, _ := m.Update(dataPageMsg{gen: m.dataGen - 1, page: domain.DataPage{
		Grid: domain.Grid{Columns: []domain.Column{{Name: "late"}}, Rows: []domain.Row{{Cells: []domain.Cell{{Text: "x", Kind: domain.CellText}}}}},
	}})
	m = next.(Model)
	if len(m.data.columns) != 3 || len(m.data.rows) != 100 {
		t.Fatalf("stale page mutated the window: cols=%d rows=%d", len(m.data.columns), len(m.data.rows))
	}
}

func TestFormatCount(t *testing.T) {
	cases := map[int64]string{
		0:       "0",
		999:     "999",
		1000:    "1,000",
		12480:   "12,480",
		1000000: "1,000,000",
		-1:      "-1",
	}
	for input, want := range cases {
		if got := formatCount(input); got != want {
			t.Fatalf("formatCount(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestGoldenDataStates(t *testing.T) {
	empty, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 0})
	assertGolden(t, "workspace-data-empty.golden", empty.View().Content)

	failed, _ := dataWorkspaceWithReader(t, &fakeDataReader{
		total:  5,
		failAt: map[int]error{0: domain.NewError(domain.KindEngine, "data", "relation unavailable")},
	})
	assertGolden(t, "workspace-data-error.golden", failed.View().Content)
}

func TestDataTreeCursorDoesNotFetchUntilOpened(t *testing.T) {
	reader := &fakeDataReader{total: 100}
	m, _ := dataWorkspaceWithReader(t, reader)
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab") // tree
	before := len(reader.calls)
	m, _ = press(t, m, "down") // invoices
	if len(reader.calls) != before {
		t.Fatalf("cursor move queried data: %d -> %d", before, len(reader.calls))
	}
	m, cmd := press(t, m, "enter")
	m = settle(t, m, cmd)
	if len(reader.calls) <= before {
		t.Fatal("Enter on a table did not load data")
	}
}

func TestDataCancelDropsLateSuccess(t *testing.T) {
	reader := &fakeDataReader{total: 250}
	m, _ := dataWorkspaceWithReader(t, reader)
	gen := m.dataGen
	m.cancelDataFetch()
	next, _ := m.Update(dataPageMsg{gen: gen, direction: dataNext, page: domain.DataPage{
		Grid: domain.Grid{Rows: []domain.Row{{Cells: []domain.Cell{{Text: "late"}}}}},
	}})
	m = next.(Model)
	if len(m.data.rows) != 100 || m.data.rows[0].Cells[0].Text == "late" {
		t.Fatalf("late success changed rows: %d %+v", len(m.data.rows), m.data.rows[0])
	}
}

func TestDataRetryHintOnInitialError(t *testing.T) {
	reader := &fakeDataReader{total: 5, failAt: map[int]error{0: domain.NewError(domain.KindEngine, "data", "boom")}}
	m, _ := dataWorkspaceWithReader(t, reader)
	if !strings.Contains(StripANSI(m.View().Content), "press m to retry") {
		t.Fatalf("retry hint missing:\n%s", StripANSI(m.View().Content))
	}
}

func TestDataOversizedWindowState(t *testing.T) {
	m := workspace(t)
	m.data.loaded = true
	m.data.tooLarge = true
	m.data.rows = nil
	m.data.columns = nil
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "Row exceeds the 64 MB window") {
		t.Fatalf("oversized state missing:\n%s", view)
	}
}

func TestDataPreviousShortPageUsesOffset(t *testing.T) {
	m, _ := dataWorkspaceWithReader(t, &fakeDataReader{total: 300})
	m.data.windowStart = 200
	rows := make([]domain.Row, 100)
	for i := range rows {
		rows[i] = domain.Row{Cells: []domain.Cell{{Text: "row", Kind: domain.CellText}}}
	}
	m.data.rows = rows
	m.data.columns = []domain.Column{{Name: "name"}}
	m.data.recount()
	m.dataGen++
	next, _ := m.Update(dataPageMsg{
		gen:       m.dataGen,
		direction: dataPrevious,
		page: domain.DataPage{
			Offset: 150,
			Grid: domain.Grid{
				Columns: []domain.Column{{Name: "name"}},
				Rows:    make([]domain.Row, 50),
			},
			HasMore: true,
		},
	})
	m = next.(Model)
	if m.data.windowStart != 150 {
		t.Fatalf("window start = %d, want 150", m.data.windowStart)
	}
	if m.data.cursorRow != 199 {
		t.Fatalf("cursor = %d, want 199", m.data.cursorRow)
	}
}
