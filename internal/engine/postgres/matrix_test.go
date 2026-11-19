package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestRoleMatrixLoadsAllLoginsAndMemberships(t *testing.T) {
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
	mock.ExpectQuery(`SELECT rolname, rolcanlogin`).WillReturnRows(
		sqlmock.NewRows([]string{"rolname", "rolcanlogin"}).
			AddRow("app_ro", true).
			AddRow("reporting", false))
	mock.ExpectQuery(`SELECT member.rolname, granted.rolname`).WillReturnRows(
		sqlmock.NewRows([]string{"member", "role"}).
			AddRow("app_ro", "reporting"))
	matrix, err := c.RoleMatrix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Principals) != 2 || len(matrix.Roles) != 2 {
		t.Fatalf("principals=%d roles=%d", len(matrix.Principals), len(matrix.Roles))
	}
	user := domain.RoleRef{Engine: domain.EnginePostgreSQL, Name: "app_ro", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: domain.EnginePostgreSQL, Name: "reporting", Kind: domain.KindRole}
	if !matrix.Members[user.Key()][role.Key()] {
		t.Fatal("membership not loaded")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPrivilegeMatrixLoadsDatabaseSchemaAndTableCells(t *testing.T) {
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
	mock.ExpectQuery(`SELECT rolname, rolcanlogin`).WillReturnRows(
		sqlmock.NewRows([]string{"rolname", "rolcanlogin"}).
			AddRow("app_ro", true).
			AddRow("reporting", false))
	mock.ExpectQuery(`FROM pg_database`).WillReturnRows(
		sqlmock.NewRows([]string{"grantee", "database", "privilege"}).
			AddRow("reporting", "orders", "CONNECT"))
	mock.ExpectQuery(`FROM pg_namespace`).WillReturnRows(
		sqlmock.NewRows([]string{"grantee", "schema", "privilege"}).
			AddRow("reporting", "public", "USAGE"))
	mock.ExpectQuery(`FROM pg_class`).WillReturnRows(
		sqlmock.NewRows([]string{"grantee", "schema", "table", "privilege"}).
			AddRow("reporting", "public", "customers", "SELECT").
			AddRow("app_ro", "public", "customers", "UPDATE"))
	database := domain.ObjectRef{Engine: domain.EnginePostgreSQL, Kind: domain.KindDatabase, Database: "orders", Name: "orders"}
	schema := domain.ObjectRef{Engine: domain.EnginePostgreSQL, Kind: domain.KindSchema, Database: "orders", Schema: "public", Name: "public"}
	table := domain.ObjectRef{Engine: domain.EnginePostgreSQL, Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers"}
	matrix, err := c.PrivilegeMatrix(context.Background(), []domain.ObjectRef{database, schema, table})
	if err != nil {
		t.Fatal(err)
	}
	user := domain.RoleRef{Engine: domain.EnginePostgreSQL, Name: "app_ro", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: domain.EnginePostgreSQL, Name: "reporting", Kind: domain.KindRole}
	if got := matrix.Cells[role.Key()][database.Key()]; len(got) != 1 || got[0] != "CONNECT" {
		t.Fatalf("database cell = %v", got)
	}
	if got := matrix.Cells[role.Key()][schema.Key()]; len(got) != 1 || got[0] != "USAGE" {
		t.Fatalf("schema cell = %v", got)
	}
	if got := matrix.Cells[role.Key()][table.Key()]; len(got) != 1 || got[0] != "SELECT" {
		t.Fatalf("table cell = %v", got)
	}
	if got := matrix.Cells[user.Key()][table.Key()]; len(got) != 1 || got[0] != "UPDATE" {
		t.Fatalf("user cell = %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
