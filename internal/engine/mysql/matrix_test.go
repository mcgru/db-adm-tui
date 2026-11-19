package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestRoleMatrixLoadsRoleAccountsAndMemberships(t *testing.T) {
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
	mock.ExpectQuery(`SELECT VERSION`).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("8.0.36"))
	mock.ExpectQuery(`SELECT DISTINCT FROM_USER`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app_role", "%").
			AddRow("app_role2", "%"))
	mock.ExpectQuery(`SELECT User, Host`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app_ro", "%").
			AddRow("app_role", "%").
			AddRow("app_role2", "%"))
	mock.ExpectQuery(`SELECT FROM_USER, FROM_HOST, TO_USER, TO_HOST`).WillReturnRows(
		sqlmock.NewRows([]string{"from_user", "from_host", "to_user", "to_host"}).
			AddRow("app_role", "%", "app_ro", "%"))
	matrix, err := c.RoleMatrix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Roles) != 2 {
		t.Fatalf("roles = %+v", matrix.Roles)
	}
	user := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_ro@%", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_role@%", Kind: domain.KindRole}
	if !matrix.Members[user.Key()][role.Key()] {
		t.Fatal("membership not loaded")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMariaDBRoleMatrixMapsRoleNames(t *testing.T) {
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
	mock.ExpectQuery(`SELECT VERSION`).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("10.11.2-MariaDB"))
	mock.ExpectQuery(`SELECT User, Host, is_role`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host", "is_role"}).
			AddRow("app_ro", "%", "N").
			AddRow("app_role", "", "Y"))
	mock.ExpectQuery(`SELECT User, Host`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app_ro", "%").
			AddRow("app_role", ""))
	mock.ExpectQuery(`SELECT Role, '', User, Host`).WillReturnRows(
		sqlmock.NewRows([]string{"role", "unused", "user", "host"}).
			AddRow("app_role", "", "app_ro", "%"))
	matrix, err := c.RoleMatrix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	user := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_ro@%", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_role", Kind: domain.KindRole}
	if !matrix.Members[user.Key()][role.Key()] {
		t.Fatalf("MariaDB membership missing: %+v", matrix.Members)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPrivilegeMatrixLoadsSchemaAndTableCells(t *testing.T) {
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
	mock.ExpectQuery(`SELECT VERSION`).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("8.0.36"))
	mock.ExpectQuery(`SELECT DISTINCT FROM_USER`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app_role", "%"))
	mock.ExpectQuery(`SELECT User, Host`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app_ro", "%").
			AddRow("app_role", "%"))
	mock.ExpectQuery(`information_schema.TABLE_PRIVILEGES`).WillReturnRows(
		sqlmock.NewRows([]string{"grantee", "schema", "table", "privilege"}).
			AddRow("'app_role'@'%'", "billing", "invoices", "SELECT").
			AddRow("'app_ro'@'%'", "billing", "", "SELECT"))
	database := domain.ObjectRef{Engine: domain.EngineMySQL, Kind: domain.KindDatabase, Database: "billing", Name: "billing"}
	table := domain.ObjectRef{Engine: domain.EngineMySQL, Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	matrix, err := c.PrivilegeMatrix(context.Background(), []domain.ObjectRef{database, table})
	if err != nil {
		t.Fatal(err)
	}
	user := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_ro@%", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: domain.EngineMySQL, Name: "app_role@%", Kind: domain.KindRole}
	if got := matrix.Cells[role.Key()][table.Key()]; len(got) != 1 || got[0] != "SELECT" {
		t.Fatalf("table cell = %v", got)
	}
	if got := matrix.Cells[user.Key()][database.Key()]; len(got) != 1 || got[0] != "SELECT" {
		t.Fatalf("database cell = %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
