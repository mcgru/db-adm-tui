package mysql

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
	mock.ExpectExec(`SET SESSION TRANSACTION READ ONLY`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := c.SetReadOnly(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if !c.SessionReadOnly() {
		t.Fatal("session read-only not tracked")
	}
	mock.ExpectExec(`SET SESSION TRANSACTION READ WRITE`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := c.SetReadOnly(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT CURRENT_USER\(\)`).WillReturnRows(sqlmock.NewRows([]string{"user"}).AddRow("billing_ro@%"))
	mock.ExpectQuery(`SHOW GRANTS FOR CURRENT_USER\(\)`).WillReturnRows(sqlmock.NewRows([]string{"grants"}).
		AddRow("GRANT SELECT ON `billing`.* TO `billing_ro`@`%`"))
	elevated, err := c.Elevated(context.Background())
	if err != nil || elevated {
		t.Fatalf("elevated = %v, %v", elevated, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMysqlElevatedGrants(t *testing.T) {
	for _, grants := range [][]string{
		{"GRANT ALL PRIVILEGES ON *.* TO `root`@`localhost`"},
		{"GRANT SUPER ON *.* TO `admin`@`%`"},
		{"GRANT SYSTEM_USER ON *.* TO `admin`@`%`"},
	} {
		if !mysqlElevated(grants) {
			t.Fatalf("grants %v not elevated", grants)
		}
	}
	if mysqlElevated([]string{"GRANT SELECT ON `billing`.* TO `billing_ro`@`%`"}) {
		t.Fatal("read-only grant reported elevated")
	}
}
