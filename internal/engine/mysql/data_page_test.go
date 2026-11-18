package mysql

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

	mock.ExpectQuery(`SELECT COLUMN_NAME`).WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME"}).AddRow("id"))
	dataRows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "a").AddRow(2, "b").AddRow(3, "c")
	mock.ExpectQuery("SELECT \\* FROM `billing`\\.`customers` ORDER BY `id` LIMIT \\? OFFSET \\?").
		WithArgs(3, 0).WillReturnRows(dataRows)
	mock.ExpectQuery(`SELECT TABLE_ROWS`).WillReturnRows(sqlmock.NewRows([]string{"TABLE_ROWS"}).AddRow(10))

	page, err := c.Page(context.Background(), domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"},
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
