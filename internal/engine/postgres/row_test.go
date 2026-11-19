package postgres

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildRowInsertAndDelete(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Schema: "app", Name: "customers"}
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
	if statement != `INSERT INTO "app"."customers" ("id", "name", "status") VALUES ($1, $2, $3)` {
		t.Fatalf("statement = %q", statement)
	}
	if len(args) != 3 || args[0] != "7" || args[1] != "Ada" || args[2] != nil {
		t.Fatalf("args = %#v", args)
	}
	statement, args, err = buildRowDelete(domain.RowDelete{Ref: ref, Keys: []domain.RowValue{
		{Column: "id", Value: "7"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if statement != `DELETE FROM "app"."customers" WHERE "id" = $1` || len(args) != 1 || args[0] != "7" {
		t.Fatalf("delete = %q, %#v", statement, args)
	}
}

func TestBuildRowInsertDefaults(t *testing.T) {
	statement, args, err := buildRowInsert(domain.RowInsert{
		Ref:    domain.ObjectRef{Kind: domain.KindTable, Name: "customers"},
		Values: []domain.RowValue{{Column: "id", Default: true}},
	})
	if err != nil || !strings.Contains(statement, "DEFAULT VALUES") || len(args) != 0 {
		t.Fatalf("statement=%q args=%#v err=%v", statement, args, err)
	}
}
