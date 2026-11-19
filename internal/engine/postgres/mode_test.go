package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSetReadOnlyAndElevated(t *testing.T) {
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
	mock.ExpectExec(`SET default_transaction_read_only = on`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := c.SetReadOnly(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if !c.SessionReadOnly() {
		t.Fatal("session read-only not tracked")
	}
	mock.ExpectExec(`SET default_transaction_read_only = off`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := c.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if c.SessionReadOnly() {
		t.Fatal("session read-write not tracked")
	}
	mock.ExpectQuery(`SELECT rolsuper OR rolcreaterole OR rolcreatedb`).
		WillReturnRows(sqlmock.NewRows([]string{"elevated"}).AddRow(true))
	elevated, err := c.Elevated(context.Background())
	if err != nil || !elevated {
		t.Fatalf("elevated = %v, %v", elevated, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
