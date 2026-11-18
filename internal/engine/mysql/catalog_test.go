package mysql

import (
	"context"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMySQLObjectKind(t *testing.T) {
	cases := map[string]domain.ObjectKind{
		"BASE TABLE":  domain.KindTable,
		"VIEW":        domain.KindView,
		"SYSTEM VIEW": domain.KindView,
		"SEQUENCE":    domain.KindSequence,
	}
	for input, want := range cases {
		got, ok := mysqlObjectKind(input)
		if !ok || got != want {
			t.Fatalf("mysqlObjectKind(%q) = %q, %v; want %q", input, got, ok, want)
		}
	}
	if _, ok := mysqlObjectKind("TEMPORARY"); ok {
		t.Fatal("unsupported table type must be skipped")
	}
}

func TestMySQLAccessHelpers(t *testing.T) {
	read := map[string]bool{"SELECT": true}
	write := map[string]bool{"INSERT": true}
	if anyWrite(read) {
		t.Fatal("SELECT must not be a write")
	}
	if !anyWrite(write) {
		t.Fatal("INSERT must be a write")
	}
	if got := quoteGrantee("o'brien", "%"); got != "'o''brien'@'%'" {
		t.Fatalf("grantee = %q", got)
	}
	attrs := mysqlAttributes("caching_sha2_password", "Y")
	if len(attrs) != 2 || attrs[0] != "CACHING_SHA2_PASSWORD" || attrs[1] != "LOCKED" {
		t.Fatalf("attrs = %v", attrs)
	}
	if node := databaseNode("billing"); node.Ref.Engine != domain.EngineMySQL || node.Name != "billing" || !node.Expandable {
		t.Fatalf("node = %+v", node)
	}
}

func TestMySQLOrderKeySkipsNonTables(t *testing.T) {
	c := &connection{}
	key, err := c.OrderKey(context.Background(), domain.ObjectRef{Kind: domain.KindDatabase})
	if err != nil || key != "" {
		t.Fatalf("key = %q, %v", key, err)
	}
}
