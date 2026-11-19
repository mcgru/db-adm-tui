package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestRoleAdminStatements(t *testing.T) {
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
	mock.ExpectExec(`CREATE ROLE 'app_role'@'%'`).WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.CreateRole(context.Background(), domain.RoleRef{Name: "app_role"}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`GRANT 'app_role'@'%' TO 'app_user'@'localhost'`).WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.GrantRole(context.Background(), domain.RoleMembership{
		Role: domain.RoleRef{Name: "app_role"}, Member: domain.RoleRef{Name: "app_user@localhost"},
	}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("GRANT SELECT, UPDATE ON `billing`\\.`invoices` TO 'app_role'@'%'").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.GrantPrivileges(context.Background(), domain.PrivilegeGrant{
		Role:       domain.RoleRef{Name: "app_role"},
		Target:     domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"},
		Privileges: []string{"select", "UPDATE"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMariaDBCreateRoleUsesHostlessName(t *testing.T) {
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
	mock.ExpectExec("CREATE ROLE `maria_role`").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.CreateRole(context.Background(), domain.RoleRef{Name: "maria_role", Kind: domain.KindRole}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("GRANT `maria_role` TO 'app_user'@'%'").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := c.GrantRole(context.Background(), domain.RoleMembership{
		Role:   domain.RoleRef{Name: "maria_role", Kind: domain.KindRole},
		Member: domain.RoleRef{Name: "app_user@%", Kind: domain.KindUser},
	}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRoleAdminRejectsUnknownPrivilege(t *testing.T) {
	c := &connection{}
	_, err := c.GrantPrivileges(context.Background(), domain.PrivilegeGrant{
		Role:       domain.RoleRef{Name: "app_role"},
		Target:     domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"},
		Privileges: []string{"DROP DATABASE"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported privilege") {
		t.Fatalf("err = %v", err)
	}
}
