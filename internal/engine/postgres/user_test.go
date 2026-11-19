package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestUserAdminStatements(t *testing.T) {
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
	mock.ExpectExec(`CREATE ROLE "app_new" LOGIN PASSWORD 's''ecret'`).WillReturnResult(sqlmock.NewResult(0, 0))
	result, err := c.CreateUser(context.Background(), domain.UserSpec{
		Engine: domain.EnginePostgreSQL, Name: "app_new", Password: "s'ecret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Statement, "s'ecret") || !strings.Contains(result.Statement, "***") {
		t.Fatalf("audit statement = %q", result.Statement)
	}
	mock.ExpectExec(`DROP ROLE "app_new"`).WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.DropUser(context.Background(), domain.RoleRef{Name: "app_new"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserAdminRejectsLineBreaks(t *testing.T) {
	c := &connection{}
	_, err := c.CreateUser(context.Background(), domain.UserSpec{
		Engine: domain.EnginePostgreSQL, Name: "app_new", Password: "bad\npassword",
	})
	if err == nil || !strings.Contains(err.Error(), "line breaks") {
		t.Fatalf("err = %v", err)
	}
}
