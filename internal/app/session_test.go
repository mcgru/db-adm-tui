package app

import (
	"context"
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

func TestSessionMetadataRequiresAStatsReader(t *testing.T) {
	f := newFixture(t)
	profile := domain.Profile{Label: "billing", Engine: domain.EngineMySQL, Database: "billing"}
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}}, profile, Credential{}, f.connector, f.resolver)
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
	}, profile, Credential{}, f.connector, f.resolver)
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
	session := newSession(ports.SessionBundle{Session: &fakeConn{}, Catalog: fakeCatalog{}}, profile, Credential{}, f.connector, f.resolver)
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
	}, profile, Credential{}, f.connector, f.resolver)
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
	}, domain.Profile{Engine: domain.EngineMySQL}, Credential{}, f.connector, f.resolver)
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
	}, profile, Credential{Password: "secret", Provided: true}, f.connector, f.resolver)
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
