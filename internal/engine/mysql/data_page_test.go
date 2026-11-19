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

	mock.ExpectQuery(`SELECT COLUMN_NAME`).WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME"}).AddRow("id"))
	mock.ExpectQuery("SELECT \\* FROM `billing`\\.`customers` WHERE LOWER\\(`name`\\) REGEXP LOWER\\(\\?\\) ORDER BY `id` LIMIT \\? OFFSET \\?").
		WithArgs("^a", 2, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice").AddRow(2, "bob"))
	mock.ExpectQuery(`SELECT TABLE_ROWS`).WillReturnRows(sqlmock.NewRows([]string{"TABLE_ROWS"}).AddRow(2))

	page, err := c.Page(context.Background(), domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"},
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

func TestPageBindsSensitiveRegexpFilter(t *testing.T) {
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
	mock.ExpectQuery("SELECT \\* FROM `billing`\\.`customers` WHERE `name` REGEXP BINARY \\? ORDER BY `id` LIMIT \\? OFFSET \\?").
		WithArgs("^A", 2, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Alice"))
	mock.ExpectQuery(`SELECT TABLE_ROWS`).WillReturnRows(sqlmock.NewRows([]string{"TABLE_ROWS"}).AddRow(1))

	page, err := c.Page(context.Background(), domain.DataQuery{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"},
		Offset: 0,
		Limit:  1,
		Filter: &domain.DataFilter{Field: "name", Pattern: "^A", CaseInsensitive: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Grid.Rows) != 1 || page.HasMore {
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
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"},
		Limit:  1,
		Filter: &domain.DataFilter{Pattern: "x"},
	})
	if domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
