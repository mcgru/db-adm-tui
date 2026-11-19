package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestPrivilegesGrantsAndRoles(t *testing.T) {
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
	mock.ExpectQuery(`SELECT CURRENT_USER\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"user"}).AddRow("billing_ro@%"))
	mock.ExpectQuery(`SHOW GRANTS FOR 'billing_ro'@'%'`).
		WillReturnRows(sqlmock.NewRows([]string{"grants"}).
			AddRow("GRANT SELECT ON `billing`.`invoices` TO `billing_ro`@`%`"))
	mock.ExpectQuery(`SELECT CURRENT_ROLE\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"CURRENT_ROLE()"}).AddRow("reader@%"))
	mock.ExpectQuery(`SELECT ROLE_NAME, ROLE_HOST, IS_GRANTABLE`).
		WithArgs("billing_ro@%").
		WillReturnRows(sqlmock.NewRows([]string{"ROLE_NAME", "ROLE_HOST", "IS_GRANTABLE"}).
			AddRow("reader", "%", "NO"))

	report, err := c.Privileges(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "billing", Name: "invoices",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Principal != "billing_ro@%" || report.Scope != "table billing.invoices" {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Sections) != 3 {
		t.Fatalf("sections = %+v", report.Sections)
	}
	if report.Sections[0].Lines[0] != "GRANT SELECT ON `billing`.`invoices` TO `billing_ro`@`%`" {
		t.Fatalf("grant = %+v", report.Sections[0])
	}
	if report.Sections[1].Lines[0] != "reader@%" {
		t.Fatalf("active role = %+v", report.Sections[1])
	}
	if report.Sections[2].Lines[0] != "reader@%" {
		t.Fatalf("granted role = %+v", report.Sections[2])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
