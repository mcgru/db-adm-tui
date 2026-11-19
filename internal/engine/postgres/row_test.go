package postgres

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildRowInsert(t *testing.T) {
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
}

func TestBuildRowUpdateAndDeleteMany(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Schema: "app", Name: "customers"}
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
	if statement != `UPDATE "app"."customers" SET "status" = $1 WHERE ("id" = $2) OR ("id" = $3)` {
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
	if statement != `UPDATE "app"."customers" SET "status" = DEFAULT WHERE ("id" = $1)` {
		t.Fatalf("default update = %q", statement)
	}
	if len(args) != 1 || args[0] != "7" {
		t.Fatalf("default args = %#v", args)
	}

	statement, args, err = buildRowDelete(domain.RowDelete{Ref: ref, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	if statement != `DELETE FROM "app"."customers" WHERE ("id" = $1) OR ("id" = $2)` {
		t.Fatalf("delete = %q", statement)
	}
	if len(args) != 2 || args[0] != "7" || args[1] != "8" {
		t.Fatalf("delete args = %#v", args)
	}
}

func TestBuildRowCompositeKey(t *testing.T) {
	statement, args, err := buildRowDelete(domain.RowDelete{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Name: "customers"},
		Rows: [][]domain.RowValue{
			{{Column: "tenant", Value: "a"}, {Column: "id", Value: "1"}},
			{{Column: "tenant", Value: "b"}, {Column: "id", Value: "2"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if statement != `DELETE FROM "public"."customers" WHERE ("tenant" = $1 AND "id" = $2) OR ("tenant" = $3 AND "id" = $4)` {
		t.Fatalf("statement = %q", statement)
	}
	if len(args) != 4 {
		t.Fatalf("args = %#v", args)
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

func TestBuildRowRejectsMismatchedKeys(t *testing.T) {
	_, _, err := buildRowDelete(domain.RowDelete{
		Ref: domain.ObjectRef{Kind: domain.KindTable, Name: "customers"},
		Rows: [][]domain.RowValue{
			{{Column: "id", Value: "1"}},
			{{Column: "other", Value: "2"}},
		},
	})
	if err == nil {
		t.Fatal("mismatched key columns accepted")
	}
}
