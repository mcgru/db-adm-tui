package mysql

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildDataSelectUsesBackticksAndBoundWindow(t *testing.T) {
	ref := domain.ObjectRef{Kind: domain.KindTable, Database: "bil`ling", Name: "in voices"}
	got := buildDataSelect(ref, "id, `part`")
	want := "SELECT * FROM `bil``ling`.`in voices` ORDER BY `id`, ```part``` LIMIT ? OFFSET ?"
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
	got = buildDataSelect(domain.ObjectRef{Kind: domain.KindTable, Database: "billing", Name: "customers"}, "")
	if strings.Contains(got, "ORDER BY") {
		t.Fatalf("empty order key must not sort: %q", got)
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
