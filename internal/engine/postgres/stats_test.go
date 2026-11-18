package postgres

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
	mock.ExpectQuery(`SELECT a.attname`).
		WithArgs("public", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"attname", "format_type", "nullable", "default", "attnum"}).
			AddRow("id", "bigint", false, "nextval('customers_id_seq')", 1).
			AddRow("name", "text", true, "", 2))

	structure, err := c.Structure(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers",
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

func TestIndexesScanColumnsAndSize(t *testing.T) {
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
	mock.ExpectQuery(`SELECT i.relname`).
		WithArgs("public", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"relname", "unique", "method", "definition", "size", "columns"}).
			AddRow("customers_pkey", true, "btree", "CREATE UNIQUE INDEX ...", int64(8192), "id"))

	indexes, err := c.Indexes(context.Background(), domain.ObjectRef{
		Kind: domain.KindTable, Database: "orders", Schema: "public", Name: "customers",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 1 {
		t.Fatalf("indexes = %+v", indexes)
	}
	index := indexes[0]
	if index.Name != "customers_pkey" || !index.Unique || index.Method != "btree" ||
		index.SizeBytes != 8192 || len(index.Columns) != 1 || index.Columns[0] != "id" {
		t.Fatalf("index = %+v", index)
	}
}
