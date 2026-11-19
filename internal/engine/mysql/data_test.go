package mysql

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildDataSelectUsesBackticksAndBoundWindow(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "bil`ling", Name: "in voices"}
	got := buildDataSelect(ref, "id, `part`", nil)
	want := "SELECT * FROM `bil``ling`.`in voices` ORDER BY `id`, ```part``` LIMIT ? OFFSET ?"
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
	got = buildDataSelect(domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"}, "", nil)
	if strings.Contains(got, "ORDER BY") {
		t.Fatalf("empty order key must not sort: %q", got)
	}
}

func TestBuildDataSelectFilterUsesRegexpAndCaseFlag(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"}
	got := buildDataSelect(ref, "id", &domain.DataFilter{
		Field: "em`ail", Pattern: "^a", CaseInsensitive: true,
	})
	want := "SELECT * FROM `billing`.`customers` WHERE LOWER(`em``ail`) REGEXP LOWER(?) ORDER BY `id` LIMIT ? OFFSET ?"
	if got != want {
		t.Fatalf("insensitive query = %q, want %q", got, want)
	}
	got = buildDataSelect(ref, "id", &domain.DataFilter{
		Field: "email", Pattern: "^a", CaseInsensitive: false,
	})
	if !strings.Contains(got, "WHERE `email` REGEXP BINARY ?") {
		t.Fatalf("sensitive query = %q", got)
	}
}

func TestMySQLCellKinds(t *testing.T) {
	cases := map[string]domain.CellKind{
		"BIGINT":    domain.CellNumber,
		"DECIMAL":   domain.CellNumber,
		"TINYINT":   domain.CellNumber,
		"BOOLEAN":   domain.CellBool,
		"JSON":      domain.CellJSON,
		"BLOB":      domain.CellBinary,
		"VARBINARY": domain.CellBinary,
		"DATETIME":  domain.CellDatetime,
		"VARCHAR":   domain.CellText,
		"UNKNOWN":   domain.CellText,
	}
	for input, want := range cases {
		if got := mysqlCellKind(input); got != want {
			t.Fatalf("mysqlCellKind(%q) = %q, want %q", input, got, want)
		}
	}
}
