package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestSchemasKeepPublicFirstAndGroupTheRest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := newCatalog(&connection{conn: conn, db: db})
	mock.ExpectQuery(`SELECT nspname`).
		WillReturnRows(sqlmock.NewRows([]string{"nspname"}).AddRow("audit").AddRow("public").AddRow("reporting"))

	nodes, err := c.schemas(context.Background(), "orders")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("schema nodes = %+v", nodes)
	}
	if nodes[0].Name != "public" || nodes[0].Kind != domain.KindSchema {
		t.Fatalf("public is not first: %+v", nodes[0])
	}
	if nodes[1].Name != "schemas" || nodes[1].Kind != domain.KindSchemas || !nodes[1].Expandable {
		t.Fatalf("schema group missing: %+v", nodes[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaGroupExcludesPublic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c := newCatalog(&connection{conn: conn, db: db})
	mock.ExpectQuery(`SELECT nspname`).
		WillReturnRows(sqlmock.NewRows([]string{"nspname"}).AddRow("audit").AddRow("public").AddRow("reporting"))

	nodes, err := c.otherSchemas(context.Background(), "orders")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Name != "audit" || nodes[1].Name != "reporting" {
		t.Fatalf("schema group children = %+v", nodes)
	}
	for _, node := range nodes {
		if node.Name == "public" {
			t.Fatalf("public leaked into the schema group: %+v", nodes)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresObjectKind(t *testing.T) {
	cases := map[string]domain.ObjectKind{
		"r": domain.KindTable,
		"p": domain.KindTable,
		"f": domain.KindTable,
		"v": domain.KindView,
		"m": domain.KindView,
		"S": domain.KindSequence,
	}
	for relkind, want := range cases {
		if got := postgresObjectKind(relkind); got != want {
			t.Fatalf("postgresObjectKind(%q) = %q, want %q", relkind, got, want)
		}
	}
}

func TestPostgresOrderKeyCastsFormatParameters(t *testing.T) {
	if !strings.Contains(orderKeyQuery, "$1::text") || !strings.Contains(orderKeyQuery, "$2::text") {
		t.Fatal("order-key query must cast format() parameters to text")
	}
}

func TestPostgresOrderKeySkipsNonTables(t *testing.T) {
	c := newCatalog(&connection{})
	key, err := c.OrderKey(context.Background(), domain.ObjectRef{Kind: domain.KindDatabase})
	if err != nil || key != "" {
		t.Fatalf("key = %q, %v", key, err)
	}
}
