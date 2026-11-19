package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestPageUsesLimitPlusOneAndEstimate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := &connection{conn: conn, db: db}

	mock.ExpectQuery(`SELECT a.attname`).WillReturnRows(sqlmock.NewRows([]string{"attname"}).AddRow("id"))
	dataRows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "a").AddRow(2, "b").AddRow(3, "c")
	mock.ExpectQuery(`SELECT \* FROM "public"\."customers" ORDER BY "id" LIMIT \$1 OFFSET \$2`).
		WithArgs(3, 0).WillReturnRows(dataRows)
	mock.ExpectQuery(`SELECT reltuples::bigint`).WillReturnRows(sqlmock.NewRows([]string{"reltuples"}).AddRow(10))

	page, err := c.Page(context.Background(), domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers"},
		Offset: 0,
		Limit:  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Grid.Rows) != 2 || !page.HasMore {
		t.Fatalf("rows=%d hasMore=%v", len(page.Grid.Rows), page.HasMore)
	}
	if page.Offset != 0 || page.OrderKey != "id" || !page.OrderKeyKnown {
		t.Fatalf("page metadata = %+v", page)
	}
	if page.EstimatedTotal != 10 || page.EstimateSource != domain.EstimateCatalog {
		t.Fatalf("estimate = %d %q", page.EstimatedTotal, page.EstimateSource)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPageBindsRegexpFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := &connection{conn: conn, db: db}

	mock.ExpectQuery(`SELECT a.attname`).WillReturnRows(sqlmock.NewRows([]string{"attname"}).AddRow("id"))
	mock.ExpectQuery(`SELECT \* FROM "public"\."customers" WHERE CAST\("name" AS text\) ~\* \$3 ORDER BY "id" LIMIT \$1 OFFSET \$2`).
		WithArgs(2, 0, "^a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice").AddRow(2, "bob"))
	mock.ExpectQuery(`SELECT reltuples::bigint`).WillReturnRows(sqlmock.NewRows([]string{"reltuples"}).AddRow(2))

	page, err := c.Page(context.Background(), domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers"},
		Offset: 0,
		Limit:  1,
		Filter: &domain.DataFilter{Field: "name", Pattern: "^a", CaseInsensitive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Grid.Rows) != 1 || !page.HasMore {
		t.Fatalf("rows=%d hasMore=%v", len(page.Grid.Rows), page.HasMore)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPageRejectsEmptyFilterField(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := &connection{conn: conn, db: db}
	_, err = c.Page(context.Background(), domain.DataQuery{
		Ref:   domain.ObjectRef{Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers"},
		Limit: 1, Filter: &domain.DataFilter{Pattern: "x"},
	})
	if domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
