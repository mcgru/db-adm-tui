package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestPrivilegesTableSections(t *testing.T) {
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
	mock.ExpectQuery(`SELECT current_user`).
		WillReturnRows(sqlmock.NewRows([]string{"current_user"}).AddRow("app_ro"))
	mock.ExpectQuery(`SELECT rolsuper`).
		WithArgs("app_ro").
		WillReturnRows(sqlmock.NewRows([]string{"super", "createrole", "createdb", "login", "repl", "bypass", "inherit"}).
			AddRow(false, false, false, true, false, false, true))
	mock.ExpectQuery(`SELECT r.rolname`).
		WithArgs("app_ro").
		WillReturnRows(sqlmock.NewRows([]string{"role", "admin", "inherited"}).
			AddRow("reporting", false, true))
	mock.ExpectQuery(`SELECT grantee, privilege_type`).
		WithArgs("public", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"grantee", "privilege", "grantable"}).
			AddRow("app_ro", "SELECT", "NO").
			AddRow("reporting", "UPDATE", "NO").
			AddRow("other", "DELETE", "NO"))
	mock.ExpectQuery(`SELECT grantee, column_name, privilege_type`).
		WithArgs("public", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"grantee", "column", "privilege", "grantable"}).
			AddRow("app_ro", "email", "SELECT", "NO"))
	mock.ExpectQuery(`SELECT t.p, has_table_privilege`).
		WithArgs("app_ro", "public", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"privilege", "allowed"}).
			AddRow("SELECT", true).
			AddRow("UPDATE", false))

	report, err := c.Privileges(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Principal != "app_ro" || report.Scope != "table public.customers" {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Sections) != 5 {
		t.Fatalf("sections = %+v", report.Sections)
	}
	grants := report.Sections[2]
	if grants.Title != "grants" || len(grants.Lines) != 2 {
		t.Fatalf("grants = %+v", grants)
	}
	if grants.Lines[0] != "SELECT on public.customers · direct" {
		t.Fatalf("direct grant = %q", grants.Lines[0])
	}
	if grants.Lines[1] != "UPDATE on public.customers · via role reporting" {
		t.Fatalf("role grant = %q", grants.Lines[1])
	}
	if columns := report.Sections[3]; columns.Title != "column grants" || len(columns.Lines) != 1 {
		t.Fatalf("column grants = %+v", columns)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
