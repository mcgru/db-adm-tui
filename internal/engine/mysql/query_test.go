package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestQueryCursorPagesForward(t *testing.T) {
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
	mock.ExpectQuery(`SELECT id, name FROM customers`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "a").AddRow(2, "b").AddRow(3, "c"))

	cursor, err := c.Query(context.Background(), domain.QueryRequest{Statement: "SELECT id, name FROM customers"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := cursor.Next(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Grid.Rows) != 2 || !first.HasMore {
		t.Fatalf("first page = %+v", first)
	}
	if len(first.Grid.Columns) != 2 || first.Grid.Columns[0].Name != "id" {
		t.Fatalf("columns = %+v", first.Grid.Columns)
	}
	second, err := cursor.Next(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Grid.Rows) != 1 || second.HasMore {
		t.Fatalf("second page = %+v", second)
	}
	if err := cursor.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQueryRejectsEmptyStatement(t *testing.T) {
	c := &connection{}
	if _, err := c.Query(context.Background(), domain.QueryRequest{}); err == nil || domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("err = %v", err)
	}
}
