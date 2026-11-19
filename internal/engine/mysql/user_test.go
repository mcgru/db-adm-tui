package mysql

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
	mock.ExpectExec(`CREATE USER 'app_new'@'%' IDENTIFIED BY 'secret'`).WillReturnResult(sqlmock.NewResult(0, 0))
	result, err := c.CreateUser(context.Background(), domain.UserSpec{
		Engine: domain.EngineMySQL, Name: "app_new", Password: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Statement, "ecret") || !strings.Contains(result.Statement, "***") {
		t.Fatalf("audit statement = %q", result.Statement)
	}
	mock.ExpectExec(`DROP USER 'app_new'@'%'`).WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.DropUser(context.Background(), domain.RoleRef{Name: "app_new"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLPasswordLiteralEscaping(t *testing.T) {
	if got := quoteMySQLLiteral(`a\b'c`); got != `'a\\b''c'` {
		t.Fatalf("literal = %q", got)
	}
}

func TestMySQLUserAccountHost(t *testing.T) {
	account, err := mysqlUserAccount("app_new@localhost", "")
	if err != nil || account != "'app_new'@'localhost'" {
		t.Fatalf("account = %q, %v", account, err)
	}
}
