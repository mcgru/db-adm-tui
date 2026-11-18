package postgres

import (
	"context"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

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

func TestPostgresOrderKeySkipsNonTables(t *testing.T) {
	c := newCatalog(&connection{})
	key, err := c.OrderKey(context.Background(), domain.ObjectRef{Kind: domain.KindDatabase})
	if err != nil || key != "" {
		t.Fatalf("key = %q, %v", key, err)
	}
}
