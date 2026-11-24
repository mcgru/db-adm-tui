package tui

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	err        error
	password   string
	conns      []*fakeConn
	catalog    *fakeCatalog
	data       *fakeDataReader
	stats      *fakeStatsReader
	privileges *fakePrivilegeReader
	query      *fakeQueryExecutor
	mode       *fakeModeController
	role       *fakeRoleAdmin
	matrix     *fakeRoleMatrix
	user       *fakeUserAdmin
	row        *fakeRowAdmin
}

type fakeRowAdmin struct {
	result  domain.MutationResult
	err     error
	calls   int
	inserts []domain.RowInsert
	updates []domain.RowUpdate
	deletes []domain.RowDelete
}

func (f *fakeRowAdmin) InsertRow(ctx context.Context, spec domain.RowInsert) (domain.MutationResult, error) {
	f.calls++
	f.inserts = append(f.inserts, spec)
	return f.result, f.err
}

func (f *fakeRowAdmin) UpdateRows(ctx context.Context, spec domain.RowUpdate) (domain.MutationResult, error) {
	f.calls++
	f.updates = append(f.updates, spec)
	return f.result, f.err
}

func (f *fakeRowAdmin) DeleteRows(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error) {
	f.calls++
	f.deletes = append(f.deletes, spec)
	return f.result, f.err
}

type fakeUserAdmin struct {
	result   domain.MutationResult
	err      error
	calls    int
	lastSpec domain.UserSpec
	lastUser domain.RoleRef
}

func (f *fakeUserAdmin) CreateUser(ctx context.Context, spec domain.UserSpec) (domain.MutationResult, error) {
	f.calls++
	f.lastSpec = spec
	return f.result, f.err
}

func (f *fakeUserAdmin) DropUser(ctx context.Context, user domain.RoleRef) (domain.MutationResult, error) {
	f.calls++
	f.lastUser = user
	return f.result, f.err
}

type fakeRoleMatrix struct {
	membership      domain.RoleMatrix
	privileges      domain.PrivilegeMatrix
	err             error
	membershipCalls int
	privilegeCalls  int
	objects         []domain.ObjectRef
}

func (f *fakeRoleMatrix) RoleMatrix(ctx context.Context) (domain.RoleMatrix, error) {
	f.membershipCalls++
	if f.err != nil {
		return domain.RoleMatrix{}, f.err
	}
	return f.membership, nil
}

func (f *fakeRoleMatrix) PrivilegeMatrix(ctx context.Context, objects []domain.ObjectRef) (domain.PrivilegeMatrix, error) {
	f.privilegeCalls++
	f.objects = objects
	if f.err != nil {
		return domain.PrivilegeMatrix{}, f.err
	}
	data := f.privileges
	if len(data.Objects) == 0 {
		data.Objects = append([]domain.ObjectRef(nil), objects...)
	}
	return data, nil
}

type fakeRoleAdmin struct {
	result     domain.MutationResult
	err        error
	calls      int
	last       string
	lastRole   domain.RoleRef
	lastMember domain.RoleRef
	lastGrant  domain.PrivilegeGrant
}

func (f *fakeRoleAdmin) CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	f.calls++
	f.last = "CreateRole"
	return f.result, f.err
}

func (f *fakeRoleAdmin) DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	f.calls++
	f.last = "DropRole"
	return f.result, f.err
}

func (f *fakeRoleAdmin) GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	f.calls++
	f.last = "GrantRole"
	f.lastRole = membership.Role
	f.lastMember = membership.Member
	return f.result, f.err
}

func (f *fakeRoleAdmin) RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	f.calls++
	f.last = "RevokeRole"
	f.lastRole = membership.Role
	f.lastMember = membership.Member
	return f.result, f.err
}

func (f *fakeRoleAdmin) GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	f.calls++
	f.last = "GrantPrivileges"
	f.lastGrant = grant
	return f.result, f.err
}

func (f *fakeRoleAdmin) RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	f.calls++
	f.last = "RevokePrivileges"
	f.lastGrant = grant
	return f.result, f.err
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

func (f *fakeQueryExecutor) Query(ctx context.Context, request domain.QueryRequest) (ports.QueryCursor, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	f.cursor = &fakeQueryCursor{pages: f.pages}
	return f.cursor, nil
}

type fakeHistoryStore struct {
	entries   []domain.HistoryEntry
	appendErr error
}

func (h *fakeHistoryStore) Load(ctx context.Context, profile string) ([]domain.HistoryEntry, error) {
	return h.entries, nil
}

func (h *fakeHistoryStore) Append(ctx context.Context, entry domain.HistoryEntry) error {
	h.entries = append(h.entries, entry)
	return h.appendErr
}

type fakePrivilegeReader struct {
	report domain.PrivilegeReport
	err    error
	ref    domain.ObjectRef
	seen   bool
}

func (f *fakePrivilegeReader) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	f.ref = ref
	f.seen = true
	if f.err != nil {
		return domain.PrivilegeReport{}, f.err
	}
	return f.report, nil
}

func defaultFakePrivileges(engine domain.Engine) *fakePrivilegeReader {
	if engine == domain.EngineMongoDB {
		return &fakePrivilegeReader{report: domain.PrivilegeReport{
			Principal: "analyst@analytics",
			Scope:     "collection analytics.customers",
			Sections: []domain.PrivilegeSection{
				{Title: "roles", Lines: []string{"read@analytics", "readWrite@analytics"}},
				{Title: "privileges", Lines: []string{"find, aggregate on analytics.customers"}},
			},
		}}
	}
	return &fakePrivilegeReader{report: domain.PrivilegeReport{
		Principal: "app_ro",
		Scope:     "table public.customers",
		Sections: []domain.PrivilegeSection{
			{Title: "grants", Lines: []string{
				"SELECT on public.customers · direct",
				"UPDATE on public.customers · via role reporting",
			}},
			{Title: "memberships", Lines: []string{"role reporting · inherited"}},
			{Title: "attributes", Lines: []string{"LOGIN · NOSUPERUSER · INHERIT"}},
		},
	}}
}

type fakeStatsReader struct {
	statistics    domain.Statistics
	structure     domain.Structure
	indexes       []domain.Index
	exact         int64
	statisticsErr error
	structureErr  error
	indexesErr    error
	exactErr      error
	exactFn       func(context.Context) (int64, error)
}

func (f *fakeStatsReader) Statistics(ctx context.Context, ref domain.ObjectRef) (domain.Statistics, error) {
	if f.statisticsErr != nil {
		return domain.Statistics{}, f.statisticsErr
	}
	return f.statistics, nil
}

func (f *fakeStatsReader) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	if f.structureErr != nil {
		return domain.Structure{}, f.structureErr
	}
	return f.structure, nil
}

func (f *fakeStatsReader) ExactCount(ctx context.Context, ref domain.ObjectRef) (int64, error) {
	if f.exactFn != nil {
		return f.exactFn(ctx)
	}
	if f.exactErr != nil {
		return 0, f.exactErr
	}
	return f.exact, nil
}

func (f *fakeStatsReader) Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error) {
	if f.indexesErr != nil {
		return nil, f.indexesErr
	}
	return f.indexes, nil
}

func defaultFakeMatrix(engine domain.Engine) *fakeRoleMatrix {
	if engine == domain.EngineMongoDB {
		user := domain.RoleRef{Engine: engine, Database: "analytics", Name: "analyst@analytics", Kind: domain.KindUser}
		role := domain.RoleRef{Engine: engine, Database: "analytics", Name: "app_role@analytics", Kind: domain.KindRole}
		builtin := domain.RoleRef{Engine: engine, Database: "analytics", Name: "read@analytics", Kind: domain.KindRole, Builtin: true}
		object := domain.ObjectRef{Engine: engine, Kind: domain.KindDatabase, Database: "analytics", Name: "analytics"}
		return &fakeRoleMatrix{
			membership: domain.RoleMatrix{
				Principals: []domain.RoleRef{user, role},
				Roles:      []domain.RoleRef{builtin, role},
				Members: map[string]map[string]bool{
					user.Key(): {builtin.Key(): true},
				},
			},
			privileges: domain.PrivilegeMatrix{
				Principals: []domain.RoleRef{role},
				Objects:    []domain.ObjectRef{object},
				Cells: map[string]map[string][]string{
					role.Key(): {object.Key(): {"find", "update"}},
				},
			},
		}
	}
	user := domain.RoleRef{Engine: engine, Name: "app_ro", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: engine, Name: "reporting", Kind: domain.KindRole}
	other := domain.RoleRef{Engine: engine, Name: "app_role", Kind: domain.KindRole}
	database := "orders"
	if engine == domain.EngineMySQL {
		database = "billing"
	}
	object := domain.ObjectRef{Engine: engine, Kind: domain.KindDatabase, Database: database, Name: database}
	defaultPriv := "SELECT"
	if engine == domain.EnginePostgreSQL {
		defaultPriv = "CONNECT"
	}
	return &fakeRoleMatrix{
		membership: domain.RoleMatrix{
			Principals: []domain.RoleRef{user, other},
			Roles:      []domain.RoleRef{role, other},
			Members: map[string]map[string]bool{
				user.Key(): {role.Key(): true},
			},
		},
		privileges: domain.PrivilegeMatrix{
			Principals: []domain.RoleRef{role, user},
			Objects:    []domain.ObjectRef{object},
			Cells: map[string]map[string][]string{
				role.Key(): {object.Key(): {defaultPriv}},
				user.Key(): {object.Key(): {defaultPriv, "UPDATE"}},
			},
		},
	}
}

func defaultFakeStats(engine domain.Engine) *fakeStatsReader {
	if engine == domain.EngineMongoDB {
		return &fakeStatsReader{
			statistics: domain.Statistics{
				EstimatedTotal: 5, EstimateSource: domain.EstimateCollection,
				SizeBytes: 16384, IndexSizeBytes: 8192,
			},
			exact: 5,
			structure: domain.Structure{
				Source:  domain.StructureSample,
				Sampled: 2,
				Fields: []domain.StructureField{
					{Name: "_id", Type: "objectId", Present: 2, Sampled: 2},
					{Name: "name", Type: "string", Present: 2, Sampled: 2},
					{Name: "profile", Type: "object", Present: 1, Sampled: 2},
					{Name: "profile.email", Type: "string", Present: 1, Sampled: 2},
				},
			},
			indexes: []domain.Index{{
				Name: "_id_", Columns: []string{"_id"}, Unique: true, Method: "btree", SizeBytes: 8192,
			}},
		}
	}
	return &fakeStatsReader{
		statistics: domain.Statistics{
			EstimatedTotal: 5, EstimateSource: domain.EstimateCatalog,
			SizeBytes: 8192, IndexSizeBytes: -1,
		},
		exact: 5,
		structure: domain.Structure{
			Source: domain.StructureCatalog,
			Fields: []domain.StructureField{
				{Name: "id", Type: "bigint", Nullable: false},
				{Name: "name", Type: "text", Nullable: true},
				{Name: "status", Type: "text", Nullable: true},
			},
		},
		indexes: []domain.Index{{
			Name: "customers_pkey", Columns: []string{"id"}, Unique: true,
			Method: "btree", SizeBytes: 8192, Definition: "CREATE UNIQUE INDEX customers_pkey ON public.customers USING btree (id)",
		}},
	}
}

func (f *fakeConnector) Connect(ctx context.Context, p domain.Profile, password string) (ports.SessionBundle, error) {
	f.password = password
	if f.err != nil {
		return ports.SessionBundle{}, f.err
	}
	conn := &fakeConn{}
	f.conns = append(f.conns, conn)
	catalog := f.catalog
	if catalog == nil {
		catalog = &fakeCatalog{engine: p.Engine}
	}
	data := f.data
	if data == nil {
		data = &fakeDataReader{total: 5}
	}
	stats := f.stats
	if stats == nil {
		stats = defaultFakeStats(p.Engine)
	}
	privileges := f.privileges
	if privileges == nil {
		privileges = defaultFakePrivileges(p.Engine)
	}
	query := f.query
	if query == nil {
		query = &fakeQueryExecutor{pages: []domain.QueryPage{{Grid: domain.Grid{
			Columns: []domain.Column{{Name: "id"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: "1", Kind: domain.CellNumber}}}},
		}}}}
	}
	mode := f.mode
	if mode == nil {
		mode = &fakeModeController{}
	}
	role := f.role
	if role == nil {
		role = &fakeRoleAdmin{result: domain.MutationResult{Statement: "role change"}}
	}
	matrix := f.matrix
	if matrix == nil {
		matrix = defaultFakeMatrix(p.Engine)
	}
	user := f.user
	if user == nil {
		user = &fakeUserAdmin{result: domain.MutationResult{Statement: "user change"}}
	}
	row := f.row
	if row == nil {
		row = &fakeRowAdmin{result: domain.MutationResult{Statement: "row change", Affected: 1}}
	}
	return ports.SessionBundle{
		Session: conn, Catalog: catalog, DataReader: data,
		StatsReader: stats, PrivilegeReader: privileges, QueryExecutor: query,
		ModeController: mode, RoleAdmin: role, RoleMatrix: matrix, UserAdmin: user,
		RowAdmin: row,
	}, nil
}
func (f *fakeConnector) Close() error { return nil }

type fakeDataReader struct {
	total        int
	noOrder      bool
	columns      []domain.Column
	estimate     int64
	unavailable  bool
	emptyColumns bool
	failAt       map[int]error
	block        chan struct{}
	calls        []domain.DataQuery
	pageFn       func(domain.DataQuery) (domain.DataPage, error)
}

func (d *fakeDataReader) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	d.calls = append(d.calls, query)
	if d.block != nil {
		select {
		case <-ctx.Done():
			return domain.DataPage{}, ctx.Err()
		case <-d.block:
		}
	}
	if err, ok := d.failAt[query.Offset]; ok {
		return domain.DataPage{}, err
	}
	if d.pageFn != nil {
		return d.pageFn(query)
	}
	columns := d.columns
	if len(columns) == 0 && !d.emptyColumns {
		columns = []domain.Column{{Name: "id"}, {Name: "name"}, {Name: "status"}}
	}
	offset := query.Offset
	if offset > d.total {
		offset = d.total
	}
	end := offset + query.Limit
	if end > d.total {
		end = d.total
	}
	statuses := []string{"active", "paused", "closed"}
	rows := make([]domain.Row, 0, end-offset)
	for i := offset; i < end; i++ {
		rows = append(rows, domain.Row{Cells: []domain.Cell{
			{Text: strconv.Itoa(i + 1), Kind: domain.CellNumber},
			{Text: fmt.Sprintf("row-%03d", i+1), Kind: domain.CellText},
			{Text: statuses[i%len(statuses)], Kind: domain.CellText},
		}})
	}
	grid := domain.Grid{Columns: columns, Rows: rows}
	total := int64(d.total)
	source := domain.EstimateCatalog
	if d.unavailable {
		total = -1
		source = ""
	}
	if d.estimate > 0 {
		total = d.estimate
	}
	orderKey := "id"
	if d.noOrder {
		orderKey = ""
	}
	return domain.DataPage{
		Grid:           grid,
		Offset:         query.Offset,
		HasMore:        end < d.total,
		EstimatedTotal: total,
		EstimateSource: source,
		OrderKey:       orderKey,
		OrderKeyKnown:  true,
	}, nil
}

type fakeCatalog struct {
	engine         domain.Engine
	childrenErr    map[string]error
	databasesErr   error
	childrenErrAll bool
	childrenCalls  int
	principalErr   error
	empty          bool
	noOrder        bool
}

func (c *fakeCatalog) Databases(ctx context.Context) ([]domain.Node, error) {
	if c.databasesErr != nil {
		return nil, c.databasesErr
	}
	name := "orders"
	switch c.engine {
	case domain.EngineMySQL:
		name = "billing"
	case domain.EngineMongoDB:
		name = "analytics"
	}
	nodes := []domain.Node{databaseNode(name, c.engine)}
	if c.engine == domain.EnginePostgreSQL {
		nodes = append(nodes, databaseNode("billing", c.engine))
	}
	return nodes, nil
}

func databaseNode(name string, engine domain.Engine) domain.Node {
	return domain.Node{
		Ref:        domain.ObjectRef{Engine: engine, Kind: domain.KindDatabase, Database: name, Name: name},
		Name:       name,
		Kind:       domain.KindDatabase,
		Expandable: true,
	}
}

func (c *fakeCatalog) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	c.childrenCalls++
	if c.childrenErrAll {
		return nil, domain.NewError(domain.KindPermission, "catalog", "permission denied")
	}
	if c.childrenErr != nil {
		if err := c.childrenErr[ref.Key()]; err != nil {
			return nil, err
		}
	}
	if c.empty {
		return nil, nil
	}
	switch ref.Kind {
	case domain.KindDatabase:
		if c.engine == domain.EnginePostgreSQL {
			return []domain.Node{
				{
					Ref:        domain.ObjectRef{Engine: c.engine, Kind: domain.KindSchema, Database: ref.Database, Schema: "public", Name: "public"},
					Name:       "public",
					Kind:       domain.KindSchema,
					Expandable: true,
				},
				{
					Ref:        domain.ObjectRef{Engine: c.engine, Kind: domain.KindSchemas, Database: ref.Database, Name: "schemas"},
					Name:       "schemas",
					Kind:       domain.KindSchemas,
					Expandable: true,
				},
			}, nil
		}
		return c.objects(ref.Database, ""), nil
	case domain.KindSchemas:
		return []domain.Node{{
			Ref:        domain.ObjectRef{Engine: c.engine, Kind: domain.KindSchema, Database: ref.Database, Schema: "reporting", Name: "reporting"},
			Name:       "reporting",
			Kind:       domain.KindSchema,
			Expandable: true,
		}}, nil
	case domain.KindSchema:
		return c.objects(ref.Database, ref.Schema), nil
	}
	return nil, domain.NewError(domain.KindConfig, "catalog", "no children")
}

func (c *fakeCatalog) objects(database, schema string) []domain.Node {
	kind := domain.KindTable
	if c.engine == domain.EngineMongoDB {
		kind = domain.KindCollection
	}
	nodes := []domain.Node{
		{Ref: domain.ObjectRef{Engine: c.engine, Kind: kind, Database: database, Schema: schema, Name: "customers"}, Name: "customers", Kind: kind, Access: domain.AccessRead},
		{Ref: domain.ObjectRef{Engine: c.engine, Kind: kind, Database: database, Schema: schema, Name: "invoices"}, Name: "invoices", Kind: kind, Access: domain.AccessWrite},
		{Ref: domain.ObjectRef{Engine: c.engine, Kind: kind, Database: database, Schema: schema, Name: "audit_log"}, Name: "audit_log", Kind: kind, Access: domain.AccessNone},
		{Ref: domain.ObjectRef{Engine: c.engine, Kind: domain.KindView, Database: database, Schema: schema, Name: "customer_view"}, Name: "customer_view", Kind: domain.KindView, Access: domain.AccessRead},
	}
	if c.engine != domain.EngineMongoDB {
		nodes = append(nodes, domain.Node{
			Ref:  domain.ObjectRef{Engine: c.engine, Kind: domain.KindSequence, Database: database, Schema: schema, Name: "orders_seq"},
			Name: "orders_seq", Kind: domain.KindSequence,
		})
	}
	return nodes
}

func (c *fakeCatalog) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	if c.principalErr != nil {
		return nil, c.principalErr
	}
	if ref.Kind == domain.KindUsers {
		name := "app_ro"
		scope := "cluster"
		switch c.engine {
		case domain.EngineMySQL:
			name = "billing_ro@%"
			scope = "%"
		case domain.EngineMongoDB:
			name = "analyst@analytics"
			scope = "analytics"
		}
		return []domain.Principal{{Name: name, Kind: domain.KindUser, Scope: scope, Attributes: []string{"LOGIN"}}}, nil
	}
	name := "reporting"
	scope := "cluster"
	if c.engine == domain.EngineMongoDB {
		name = "reporting@analytics"
		scope = "analytics"
	}
	return []domain.Principal{{Name: name, Kind: domain.KindRole, Scope: scope, Attributes: []string{"NOLOGIN"}}}, nil
}

func (c *fakeCatalog) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	if c.noOrder {
		return "", nil
	}
	if c.engine == domain.EngineMongoDB {
		return "_id", nil
	}
	return "id", nil
}

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
	t.Setenv("MYPASSFILE", "")
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
	case "ctrl+home":
		return tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl}
	case "ctrl+end":
		return tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl}
	case "ctrl+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}
	case "ctrl+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl}
	case "ctrl+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
	case "ctrl+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "insert":
		return tea.KeyPressMsg{Code: tea.KeyInsert}
	case "space":
		return tea.KeyPressMsg{Text: " ", Code: tea.KeySpace}
	case "f1":
		return tea.KeyPressMsg{Code: tea.KeyF1}
	case "ctrl+f1":
		return tea.KeyPressMsg{Code: tea.KeyF1, Mod: tea.ModCtrl}
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
	return settle(t, m, cmd)
}

// runQueryCmd executes the query command from a startQuery batch without
// waiting for the one-second elapsed-time tick.
func runQueryCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		if len(batch) == 0 {
			return m
		}
		return settle(t, m, batch[0])
	}
	return settle(t, m, cmd)
}

func settle(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for i := 0; cmd != nil && i < 20; i++ {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				m = settle(t, m, child)
			}
			return m
		}
		next, nextCmd := m.Update(msg)
		mm, ok := next.(Model)
		if !ok {
			t.Fatalf("Update returned %T", next)
		}
		m = mm
		cmd = nextCmd
	}
	return m
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

func selectKind(t *testing.T, m Model, kind domain.ObjectKind) Model {
	t.Helper()
	for i, row := range m.tree.rows {
		if row.Info == "" && row.Node.Kind == kind {
			m.tree.selected = i
			return settle(t, m, m.syncDetails())
		}
	}
	t.Fatalf("tree kind %q not found", kind)
	return m
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
	return workspaceFrom(t, m)
}

func workspaceFrom(t *testing.T, m Model) Model {
	t.Helper()
	m = connectSelected(t, m)
	if m.screen != ScreenWorkspace {
		t.Fatalf("expected workspace, message=%q modal=%v", m.message, m.modal)
	}
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "down")
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "down")
	m = settle(t, m, cmd)
	m, cmd = press(t, m, "enter")
	m = settle(t, m, cmd)
	m, _ = press(t, m, "shift+tab")
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

func fillNewForm(t *testing.T, m Model, p domain.Profile) Model {
	t.Helper()
	m.newEngine = p.Engine
	m.newHost.SetValue(p.Host)
	port := p.Port
	if port == 0 {
		port = p.Engine.DefaultPort()
	}
	m.newPort.SetValue(strconv.Itoa(port))
	m.newDatabase.SetValue(p.Database)
	m.newUser.SetValue(p.User)
	m.newAuth.SetValue(p.AuthDatabase)
	m.newLabel.SetValue(p.Label)
	m.newAllowNoTLS = p.AllowNoTLS
	m.newFocus = fieldLabel
	m.focusNewField()
	return m
}
