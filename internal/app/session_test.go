package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

type fakeDataReader struct {
	page  domain.DataPage
	query domain.DataQuery
	seen  bool
}

func (f *fakeDataReader) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	f.query = query
	f.seen = true
	return f.page, nil
}

type fakeStatsReader struct {
	statistics domain.Statistics
	structure  domain.Structure
	indexes    []domain.Index
	exact      int64
	ref        domain.ObjectRef
	seen       bool
}

func (f *fakeStatsReader) Statistics(ctx context.Context, ref domain.ObjectRef) (domain.Statistics, error) {
	f.ref = ref
	f.seen = true
	return f.statistics, nil
}

func (f *fakeStatsReader) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	f.ref = ref
	f.seen = true
	return f.structure, nil
}

func (f *fakeStatsReader) ExactCount(ctx context.Context, ref domain.ObjectRef) (int64, error) {
	f.ref = ref
	f.seen = true
	return f.exact, nil
}

func (f *fakeStatsReader) Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error) {
	f.ref = ref
	f.seen = true
	return f.indexes, nil
}

type fakePrivilegeReader struct {
	report domain.PrivilegeReport
	ref    domain.ObjectRef
	seen   bool
}

func (f *fakePrivilegeReader) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	f.ref = ref
	f.seen = true
	return f.report, nil
}

func TestSessionPrivilegesRequiresAReader(t *testing.T) {
	f := newFixture(t)
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}}, profile, Credential{}, f.connector, f.resolver, nil)
	_, err := session.Privileges(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "billing", Name: "invoices",
	})
	if err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v", err)
	}
}

func TestSessionPrivilegesReturnsTheReaderReport(t *testing.T) {
	f := newFixture(t)
	reader := &fakePrivilegeReader{report: domain.PrivilegeReport{
		Principal: "billing_ro@%",
		Scope:     "table billing.invoices",
		Sections:  []domain.PrivilegeSection{{Title: "grants", Lines: []string{"SELECT on billing.invoices"}}},
	}}
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, PrivilegeReader: reader,
	}, profile, Credential{}, f.connector, f.resolver, nil)
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	report, err := session.Privileges(context.Background(), ref)
	if err != nil || report.Principal != "billing_ro@%" || len(report.Sections) != 1 {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if !reader.seen || reader.ref != ref {
		t.Fatalf("reader ref = %+v", reader.ref)
	}
}

func TestSessionMetadataRequiresAStatsReader(t *testing.T) {
	f := newFixture(t)
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}}, profile, Credential{}, f.connector, f.resolver, nil)
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	if _, err := session.Structure(context.Background(), ref); err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("structure err = %v", err)
	}
	if _, err := session.Indexes(context.Background(), ref); err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("indexes err = %v", err)
	}
	if _, err := session.Statistics(context.Background(), ref); err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("statistics err = %v", err)
	}
	if _, err := session.ExactCount(context.Background(), ref); err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("exact count err = %v", err)
	}
}

func TestSessionMetadataReturnsReaderValues(t *testing.T) {
	f := newFixture(t)
	reader := &fakeStatsReader{
		statistics: domain.Statistics{EstimatedTotal: 10, EstimateSource: domain.EstimateCatalog, SizeBytes: 4096, IndexSizeBytes: -1},
		structure:  domain.Structure{Source: domain.StructureCatalog, Fields: []domain.StructureField{{Name: "id", Type: "bigint"}}},
		indexes:    []domain.Index{{Name: "invoices_pkey", Columns: []string{"id"}, Unique: true}},
		exact:      7,
	}
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, StatsReader: reader,
	}, profile, Credential{}, f.connector, f.resolver, nil)
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	statistics, err := session.Statistics(context.Background(), ref)
	if err != nil || statistics.EstimatedTotal != 10 || statistics.SizeBytes != 4096 {
		t.Fatalf("statistics = %+v, %v", statistics, err)
	}
	structure, err := session.Structure(context.Background(), ref)
	if err != nil || len(structure.Fields) != 1 || structure.Fields[0].Name != "id" {
		t.Fatalf("structure = %+v, %v", structure, err)
	}
	indexes, err := session.Indexes(context.Background(), ref)
	if err != nil || len(indexes) != 1 || indexes[0].Name != "invoices_pkey" {
		t.Fatalf("indexes = %+v, %v", indexes, err)
	}
	exact, err := session.ExactCount(context.Background(), ref)
	if err != nil || exact != 7 {
		t.Fatalf("exact = %d, %v", exact, err)
	}
	if !reader.seen || reader.ref != ref {
		t.Fatalf("reader ref = %+v", reader.ref)
	}
}

func TestSessionPageRequiresADataReader(t *testing.T) {
	f := newFixture(t)
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}}, profile, Credential{}, f.connector, f.resolver, nil)
	_, err := session.Page(context.Background(), domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}, Limit: 100,
	})
	if err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v, kind %v", err, domain.KindOf(err))
	}
}

func TestSessionPageReturnsTheReaderPage(t *testing.T) {
	f := newFixture(t)
	reader := &fakeDataReader{page: domain.DataPage{
		Grid: domain.Grid{
			Columns: []domain.Column{{Name: "id"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
		},
		HasMore:        true,
		EstimatedTotal: 10,
		EstimateSource: domain.EstimateCatalog,
	}}
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, DataReader: reader,
	}, profile, Credential{}, f.connector, f.resolver, nil)
	filter := &domain.DataFilter{Field: "name", Pattern: "^a", CaseInsensitive: true}
	query := domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"},
		Offset: 100, Limit: 100, Filter: filter,
	}
	page, err := session.Page(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if !reader.seen || reader.query != query {
		t.Fatalf("reader query = %+v, want %+v", reader.query, query)
	}
	if reader.query.Filter != filter || *reader.query.Filter != *filter {
		t.Fatalf("filter = %+v, want %+v", reader.query.Filter, filter)
	}
	if len(page.Grid.Rows) != 1 || page.Grid.Rows[0].Cells[0].Text != "1" {
		t.Fatalf("page = %+v", page)
	}
}

func TestSessionPageRejectsDeadSession(t *testing.T) {
	f := newFixture(t)
	reader := &fakeDataReader{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, DataReader: reader,
	}, domain.Profile{Engine: domain.EngineMySQL}, Credential{}, f.connector, f.resolver, nil)
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := session.Page(context.Background(), domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}, Limit: 100,
	})
	if err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v, kind %v", err, domain.KindOf(err))
	}
}

func TestSessionPageSwitchesPostgresDatabase(t *testing.T) {
	f := newFixture(t)
	reader := &fakeDataReader{}
	f.connector.data = reader
	profile := domain.Profile{
		Label: "prod-ro", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{database: "orders"}, DataReader: reader,
	}, profile, Credential{Password: "secret", Provided: true}, f.connector, f.resolver, nil)
	query := domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}, Limit: 100,
	}
	if _, err := session.Page(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if f.connector.profile.Database != "billing" {
		t.Fatalf("switched profile database = %q", f.connector.profile.Database)
	}
	if !reader.seen {
		t.Fatal("data reader was not called after the switch")
	}
	if got := session.CurrentDatabase(); got != "billing" {
		t.Fatalf("current database = %q", got)
	}
}

type fakeQueryCursor struct {
	pages  []domain.QueryPage
	index  int
	closed bool
}

func (c *fakeQueryCursor) Next(ctx context.Context, limit int) (domain.QueryPage, error) {
	if c.index >= len(c.pages) {
		return domain.QueryPage{}, nil
	}
	page := c.pages[c.index]
	c.index++
	return page, nil
}

func (c *fakeQueryCursor) Close(ctx context.Context) error {
	c.closed = true
	return nil
}

type fakeQueryExecutor struct {
	pages   []domain.QueryPage
	err     error
	request domain.QueryRequest
	cursor  *fakeQueryCursor
}

func (e *fakeQueryExecutor) Query(ctx context.Context, request domain.QueryRequest) (ports.QueryCursor, error) {
	e.request = request
	if e.err != nil {
		return nil, e.err
	}
	e.cursor = &fakeQueryCursor{pages: e.pages}
	return e.cursor, nil
}

type fakeHistoryStore struct {
	entries   []domain.HistoryEntry
	loadErr   error
	appendErr error
	loaded    []string
}

func (h *fakeHistoryStore) Load(ctx context.Context, profile string) ([]domain.HistoryEntry, error) {
	h.loaded = append(h.loaded, profile)
	return h.entries, h.loadErr
}

func (h *fakeHistoryStore) Append(ctx context.Context, entry domain.HistoryEntry) error {
	h.entries = append(h.entries, entry)
	return h.appendErr
}

func TestSessionValidateQuery(t *testing.T) {
	f := newFixture(t)
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}},
		domain.Profile{Engine: domain.EnginePostgreSQL}, Credential{}, f.connector, f.resolver, nil)
	if err := session.ValidateQuery("SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if err := session.ValidateQuery("DELETE FROM t"); err == nil || domain.KindOf(err) != domain.KindGuard {
		t.Fatalf("err = %v", err)
	} else if !strings.Contains(err.Error(), "Ctrl+X Ctrl+W") {
		t.Fatalf("guard error has no mode hint: %v", err)
	}
	mongo := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}},
		domain.Profile{Engine: domain.EngineMongoDB}, Credential{}, f.connector, f.resolver, nil)
	if err := mongo.ValidateQuery(`find({})`); err != nil {
		t.Fatal(err)
	}
	if err := mongo.ValidateQuery(`drop()`); err == nil {
		t.Fatal("MQL drop was accepted")
	}
}

func TestSessionQueryPagesAndHistory(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{
		{Grid: domain.Grid{
			Columns: []domain.Column{{Name: "id"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
		}, HasMore: true},
		{Grid: domain.Grid{
			Columns: []domain.Column{{Name: "id"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "2", Kind: domain.CellNumber}}}},
		}},
	}}
	historyStore := &fakeHistoryStore{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor,
	}, domain.Profile{Label: "prod", Engine: domain.EnginePostgreSQL}, Credential{Password: "secret", Provided: true}, f.connector, f.resolver, historyStore)
	request := domain.QueryRequest{Statement: "SELECT id FROM t", Database: "orders"}
	first, err := session.ExecuteQuery(context.Background(), request, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Grid.Rows) != 1 || !first.HasMore || !session.ResultsOpen() {
		t.Fatalf("first = %+v open=%v", first, session.ResultsOpen())
	}
	if executor.request.Statement != request.Statement {
		t.Fatalf("request = %+v", executor.request)
	}
	if _, err := session.ExecuteQuery(context.Background(), request, 100); err == nil {
		t.Fatal("second query while results are open must fail")
	}
	second, err := session.ResultsNext(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Grid.Rows) != 1 || second.HasMore || session.ResultsOpen() {
		t.Fatalf("second = %+v open=%v", second, session.ResultsOpen())
	}
	if len(historyStore.entries) != 1 || historyStore.entries[0].Statement != request.Statement {
		t.Fatalf("history = %+v", historyStore.entries)
	}
	if !executor.cursor.closed {
		t.Fatal("exhausted cursor was not closed")
	}
}

func TestSessionBlockedQueryNeverReachesTheEngine(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor,
	}, domain.Profile{Engine: domain.EnginePostgreSQL}, Credential{}, f.connector, f.resolver, nil)
	_, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "DROP TABLE t"}, 100)
	if err == nil || domain.KindOf(err) != domain.KindGuard {
		t.Fatalf("err = %v", err)
	}
	if executor.cursor != nil {
		t.Fatal("guard-blocked query reached the engine")
	}
}

func TestSessionCloseResults(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{
		Grid:    domain.Grid{Columns: []domain.Column{{Name: "id"}}, Rows: []domain.Row{{Cells: []domain.Cell{{Text: "1"}}}}},
		HasMore: true,
	}}}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor,
	}, domain.Profile{Engine: domain.EnginePostgreSQL}, Credential{}, f.connector, f.resolver, nil)
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "SELECT 1"}, 100); err != nil {
		t.Fatal(err)
	}
	session.CloseResults()
	if session.ResultsOpen() {
		t.Fatal("results still open")
	}
	if executor.cursor == nil || !executor.cursor.closed {
		t.Fatal("cursor not closed")
	}
}

func TestServiceHistoryReturnsStatements(t *testing.T) {
	f := newFixture(t)
	store := &fakeHistoryStore{entries: []domain.HistoryEntry{
		{Statement: "SELECT 1"},
		{Statement: "SELECT 2"},
	}}
	f.svc.SetHistory(store)
	statements := f.svc.History(context.Background(), domain.Profile{Label: "prod"})
	if len(statements) != 2 || statements[0] != "SELECT 1" {
		t.Fatalf("statements = %+v", statements)
	}
}

func TestSessionResultsHoldTheEngineLease(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{
		Grid:    domain.Grid{Columns: []domain.Column{{Name: "id"}}, Rows: []domain.Row{{Cells: []domain.Cell{{Text: "1"}}}}},
		HasMore: true,
	}}}
	reader := &fakeDataReader{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, DataReader: reader, QueryExecutor: executor,
	}, domain.Profile{Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, nil)
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "SELECT 1"}, 100); err != nil {
		t.Fatal(err)
	}
	_, err := session.Page(context.Background(), domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "orders", Name: "customers"}, Limit: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "results open") {
		t.Fatalf("err = %v", err)
	}
	if reader.seen {
		t.Fatal("data reader ran while results were open")
	}
	session.CloseResults()
	if _, err := session.Page(context.Background(), domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "orders", Name: "customers"}, Limit: 100,
	}); err != nil {
		t.Fatalf("page after release: %v", err)
	}
}

type fakeModeController struct {
	readOnly bool
	calls    []bool
	elevated bool
	err      error
}

func (m *fakeModeController) SetReadOnly(ctx context.Context, readOnly bool) error {
	if m.err != nil {
		return m.err
	}
	m.readOnly = readOnly
	m.calls = append(m.calls, readOnly)
	return nil
}

func (m *fakeModeController) SessionReadOnly() bool { return m.readOnly }

func (m *fakeModeController) Elevated(ctx context.Context) (bool, error) {
	return m.elevated, nil
}

func TestSessionReadOnlyByDefaultAndSwitch(t *testing.T) {
	f := newFixture(t)
	mode := &fakeModeController{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, ModeController: mode,
	}, domain.Profile{Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, nil)
	if !session.ReadOnly() {
		t.Fatal("new session is not read-only")
	}
	if err := session.SetReadOnly(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if len(mode.calls) != 0 {
		t.Fatalf("redundant mode call: %v", mode.calls)
	}
	if err := session.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if session.ReadOnly() || len(mode.calls) != 1 || mode.calls[0] != false {
		t.Fatalf("mode=%v calls=%v", session.ReadOnly(), mode.calls)
	}
	if err := session.ValidateQuery("DELETE FROM customers"); err != nil {
		t.Fatalf("read-write guard blocked: %v", err)
	}
	if err := session.SetReadOnly(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err := session.ValidateQuery("DELETE FROM customers"); err == nil || domain.KindOf(err) != domain.KindGuard {
		t.Fatalf("read-only guard err = %v", err)
	}
}

func TestSessionConnectEnablesReadOnlyAndDetectsElevated(t *testing.T) {
	f := newFixture(t)
	mode := &fakeModeController{elevated: true}
	f.connector.mode = mode
	profile := domain.Profile{
		Label: "prod", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}
	result, err := f.svc.Connect(context.Background(), profile, Credential{Password: "secret", Provided: true})
	if err != nil {
		t.Fatal(err)
	}
	defer result.Session.Close()
	if !result.Elevated {
		t.Fatal("elevated account was not reported")
	}
	if len(mode.calls) != 1 || mode.calls[0] != true {
		t.Fatalf("mode calls = %v", mode.calls)
	}
	if !result.Session.ReadOnly() {
		t.Fatal("connected session is not read-only")
	}
}

func TestSessionModeBlockedByResultsLease(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{
		Grid:    domain.Grid{Columns: []domain.Column{{Name: "id"}}, Rows: []domain.Row{{Cells: []domain.Cell{{Text: "1"}}}}},
		HasMore: true,
	}}}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor, ModeController: &fakeModeController{},
	}, domain.Profile{Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, nil)
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "SELECT 1"}, 100); err != nil {
		t.Fatal(err)
	}
	if err := session.SetReadOnly(context.Background(), false); err == nil || !strings.Contains(err.Error(), "results open") {
		t.Fatalf("err = %v", err)
	}
	session.CloseResults()
}

func TestSessionReadWriteAllowsAWritesThroughTheGuard(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
		Columns: []domain.Column{{Name: "id"}},
		Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
	}}}}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor, ModeController: &fakeModeController{},
	}, domain.Profile{Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, nil)
	if err := session.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "DELETE FROM customers WHERE id = 1"}, 100); err != nil {
		t.Fatal(err)
	}
	if executor.request.Statement == "" {
		t.Fatal("read-write statement did not reach the engine")
	}
	if err := session.SetReadOnly(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	executor.request = domain.QueryRequest{}
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "DELETE FROM customers WHERE id = 2"}, 100); err == nil {
		t.Fatal("read-only guard accepted a write")
	}
	if executor.request.Statement != "" {
		t.Fatal("read-only write reached the engine")
	}
}

func TestDatabaseSwitchResetsReadOnly(t *testing.T) {
	f := newFixture(t)
	mode := &fakeModeController{}
	f.connector.mode = mode
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{database: "orders"}, ModeController: mode,
	}, domain.Profile{
		Label: "prod", Engine: domain.EnginePostgreSQL, Host: "db1", Port: 5432,
		Database: "orders", User: "app_ro",
	}, Credential{Password: "secret", Provided: true}, f.connector, f.resolver, nil)
	if err := session.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Children(context.Background(), domain.ObjectRef{
		Kind: domain.KindSchema, Database: "billing", Name: "public",
	}); err != nil {
		t.Fatal(err)
	}
	if !session.ReadOnly() {
		t.Fatal("database switch did not reset read-only")
	}
	if !mode.readOnly {
		t.Fatal("new connection was not set read-only")
	}
}

func TestHistoryFailureWarnsWithoutBlockingQuery(t *testing.T) {
	f := newFixture(t)
	executor := &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
		Columns: []domain.Column{{Name: "id"}},
		Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
	}}}}
	historyStore := &fakeHistoryStore{appendErr: errors.New("disk full hunter2")}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, QueryExecutor: executor,
	}, domain.Profile{Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{Password: "hunter2", Provided: true}, f.connector, f.resolver, historyStore)
	if _, err := session.ExecuteQuery(context.Background(), domain.QueryRequest{Statement: "SELECT 1"}, 100); err != nil {
		t.Fatal(err)
	}
	warning := session.LastWarning()
	if !strings.Contains(warning, "history write failed") {
		t.Fatalf("warning = %q", warning)
	}
	if strings.Contains(warning, "hunter2") {
		t.Fatalf("warning leaked secret: %q", warning)
	}
	if session.LastWarning() != "" {
		t.Fatal("warning was not cleared")
	}
}

type fakeRoleAdmin struct {
	result domain.MutationResult
	err    error
	calls  int
}

func (f *fakeRoleAdmin) CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeRoleAdmin) DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeRoleAdmin) GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeRoleAdmin) RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeRoleAdmin) GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeRoleAdmin) RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	f.calls++
	return f.result, f.err
}

func TestRoleMutationsRequireReadWriteAndAudit(t *testing.T) {
	f := newFixture(t)
	mode := &fakeModeController{}
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `CREATE ROLE "app_role"`}}
	historyStore := &fakeHistoryStore{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, ModeController: mode, RoleAdmin: admin,
	}, domain.Profile{Label: "prod", Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, historyStore)
	if _, err := session.CreateRole(context.Background(), domain.RoleRef{Name: "app_role"}); err == nil || domain.KindOf(err) != domain.KindGuard {
		t.Fatalf("read-only err = %v", err)
	}
	if admin.calls != 0 {
		t.Fatal("read-only mutation reached the engine")
	}
	if err := session.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	result, err := session.CreateRole(context.Background(), domain.RoleRef{Name: "app_role"})
	if err != nil || result.Statement == "" || admin.calls != 1 {
		t.Fatalf("result = %+v, err = %v, calls=%d", result, err, admin.calls)
	}
	if len(historyStore.entries) != 1 || historyStore.entries[0].Mode != "r/w" {
		t.Fatalf("history = %+v", historyStore.entries)
	}
}

func TestRoleMutationFailureIsAudited(t *testing.T) {
	f := newFixture(t)
	admin := &fakeRoleAdmin{
		result: domain.MutationResult{Statement: `DROP ROLE "app_role"`},
		err:    domain.NewError(domain.KindPermission, "role", "not permitted"),
	}
	historyStore := &fakeHistoryStore{}
	session := newSession(ports.SessionBundle{
		Session: &fakeConn{}, Catalog: fakeCatalog{}, ModeController: &fakeModeController{}, RoleAdmin: admin,
	}, domain.Profile{Label: "prod", Engine: domain.EnginePostgreSQL, Database: "orders"}, Credential{}, f.connector, f.resolver, historyStore)
	if err := session.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := session.DropRole(context.Background(), domain.RoleRef{Name: "app_role"}); err == nil {
		t.Fatal("expected error")
	}
	if len(historyStore.entries) != 1 || historyStore.entries[0].Status != "error" {
		t.Fatalf("history = %+v", historyStore.entries)
	}
}
