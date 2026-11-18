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
