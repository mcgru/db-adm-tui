package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestStructureScansColumns(t *testing.T) {
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
	mock.ExpectQuery(`SELECT COLUMN_NAME`).
		WithArgs("billing", "invoices").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE", "IS_NULLABLE", "COLUMN_DEFAULT", "ORDINAL_POSITION"}).
			AddRow("id", "bigint", "NO", nil, 1).
			AddRow("name", "varchar(255)", "YES", "", 2))

	structure, err := c.Structure(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "billing", Name: "invoices",
	})
	if err != nil {
		t.Fatal(err)
	}
	if structure.Source != domain.StructureCatalog || len(structure.Fields) != 2 {
		t.Fatalf("structure = %+v", structure)
	}
	if structure.Fields[0].Name != "id" || structure.Fields[0].Nullable {
		t.Fatalf("id field = %+v", structure.Fields[0])
	}
	if structure.Fields[1].Name != "name" || !structure.Fields[1].Nullable {
		t.Fatalf("name field = %+v", structure.Fields[1])
	}
}

func TestIndexesGroupColumnsAndCardinality(t *testing.T) {
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
	mock.ExpectQuery(`SELECT INDEX_NAME`).
		WithArgs("billing", "invoices").
		WillReturnRows(sqlmock.NewRows([]string{"INDEX_NAME", "NON_UNIQUE", "INDEX_TYPE", "COLUMN_NAME", "CARDINALITY"}).
			AddRow("PRIMARY", 0, "BTREE", "id", 100).
			AddRow("idx_name", 1, "BTREE", "name", 50).
			AddRow("idx_name", 1, "BTREE", "status", 40))

	indexes, err := c.Indexes(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "billing", Name: "invoices",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 2 {
		t.Fatalf("indexes = %+v", indexes)
	}
	if !indexes[0].Unique || indexes[0].Name != "PRIMARY" || indexes[0].Stats != "cardinality 100" {
		t.Fatalf("primary = %+v", indexes[0])
	}
	if indexes[1].Unique || len(indexes[1].Columns) != 2 ||
		indexes[1].Columns[0] != "name" || indexes[1].Columns[1] != "status" {
		t.Fatalf("composite = %+v", indexes[1])
	}
}
