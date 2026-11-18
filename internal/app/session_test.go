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
	query := domain.DataQuery{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}, Offset: 100, Limit: 100,
	}
	page, err := session.Page(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if !reader.seen || reader.query != query {
		t.Fatalf("reader query = %+v, want %+v", reader.query, query)
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
