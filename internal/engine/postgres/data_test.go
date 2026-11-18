package postgres

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildDataSelectQuotesIdentifiersAndOrders(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Schema: `we"ird`, Name: "tab le"}
	got := buildDataSelect(ref, `id,na"me`)
	want := `SELECT * FROM "we""ird"."tab le" ORDER BY "id", "na""me" LIMIT $1 OFFSET $2`
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
	got = buildDataSelect(domain.ObjectRef{Kind: domain.KindTable, Name: "customers"}, "")
	if !strings.Contains(got, `FROM "public"."customers"`) {
		t.Fatalf("default schema missing: %q", got)
	}
	if strings.Contains(got, "ORDER BY") {
		t.Fatalf("empty order key must not sort: %q", got)
	}
}

func TestSplitOrderKey(t *testing.T) {
	cases := map[string][]string{
		"":        nil,
		"id":      {"id"},
		"a, b ,c": {"a", "b", "c"},
		"a,, b,":  {"a", "b"},
		`a,"b,c"`: {"a", `"b`, `c"`},
	}
	for input, want := range cases {
		got := splitOrderKey(input)
		if len(got) != len(want) {
			t.Fatalf("split(%q) = %v, want %v", input, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("split(%q) = %v, want %v", input, got, want)
			}
		}
	}
}

func TestPostgresCellKinds(t *testing.T) {
	cases := map[string]domain.CellKind{
		"INT8":        domain.CellNumber,
		"NUMERIC":     domain.CellNumber,
		"BOOL":        domain.CellBool,
		"JSONB":       domain.CellJSON,
		"BYTEA":       domain.CellBinary,
		"TIMESTAMPTZ": domain.CellDatetime,
		"TEXT":        domain.CellText,
		"UNKNOWN":     domain.CellText,
	}
	for input, want := range cases {
		if got := postgresCellKind(input); got != want {
			t.Fatalf("postgresCellKind(%q) = %q, want %q", input, got, want)
		}
	}
}
