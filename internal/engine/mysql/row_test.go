package mysql

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildRowInsert(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	statement, args, err := buildRowInsert(domain.RowInsert{
		Ref: ref,
		Values: []domain.RowValue{
			{Column: "id", Value: "7"},
			{Column: "name", Value: "Ada"},
			{Column: "status", Null: true},
			{Column: "created_at", Default: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if statement != "INSERT INTO `billing`.`invoices` (`id`, `name`, `status`) VALUES (?, ?, ?)" {
		t.Fatalf("statement = %q", statement)
	}
	if len(args) != 3 || args[0] != "7" || args[1] != "Ada" || args[2] != nil {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildRowUpdateAndDeleteMany(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"}
	rows := [][]domain.RowValue{
		{{Column: "id", Value: "7"}},
		{{Column: "id", Value: "8"}},
	}
	statement, args, err := buildRowUpdate(domain.RowUpdate{
		Ref: ref, Column: "status", Value: domain.RowValue{Value: "active"}, Rows: rows,
	})
	if err != nil {
		t.Fatal(err)
	}
	if statement != "UPDATE `billing`.`invoices` SET `status` = ? WHERE (`id` = ?) OR (`id` = ?)" {
		t.Fatalf("update = %q", statement)
	}
	if len(args) != 3 || args[0] != "active" || args[1] != "7" || args[2] != "8" {
		t.Fatalf("args = %#v", args)
	}

	statement, args, err = buildRowUpdate(domain.RowUpdate{
		Ref: ref, Column: "status", Value: domain.RowValue{Default: true}, Rows: rows[:1],
	})
	if err != nil {
		t.Fatal(err)
	}
	if statement != "UPDATE `billing`.`invoices` SET `status` = DEFAULT WHERE (`id` = ?)" {
		t.Fatalf("default update = %q", statement)
	}
	if len(args) != 1 || args[0] != "7" {
		t.Fatalf("default args = %#v", args)
	}

	statement, args, err = buildRowDelete(domain.RowDelete{Ref: ref, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	if statement != "DELETE FROM `billing`.`invoices` WHERE (`id` = ?) OR (`id` = ?)" {
		t.Fatalf("delete = %q", statement)
	}
	if len(args) != 2 || args[0] != "7" || args[1] != "8" {
		t.Fatalf("delete args = %#v", args)
	}
}

func TestBuildRowInsertDefaults(t *testing.T) {
	statement, args, err := buildRowInsert(domain.RowInsert{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "invoices"},
		Values: []domain.RowValue{{Column: "id", Default: true}},
	})
	if err != nil || !strings.Contains(statement, "() VALUES ()") || len(args) != 0 {
		t.Fatalf("statement=%q args=%#v err=%v", statement, args, err)
	}
}
