package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMongoPrivilegeHelpers(t *testing.T) {
	resource := bson.D{
		{Key: "db", Value: "analytics"},
		{Key: "collection", Value: "customers"},
	}
	if got := mongoResourceString(resource); got != "analytics.customers" {
		t.Fatalf("resource = %q", got)
	}
	if !mongoPrivilegeMatches(resource, domain.ObjectRef{
		Kind: domain.KindCollection, Database: "analytics", Name: "customers",
	}) {
		t.Fatal("exact collection must match")
	}
	if mongoPrivilegeMatches(resource, domain.ObjectRef{
		Kind: domain.KindCollection, Database: "analytics", Name: "invoices",
	}) {
		t.Fatal("another collection must not match")
	}
	if !mongoPrivilegeMatches(bson.D{{Key: "anyResource", Value: true}}, domain.ObjectRef{
		Kind: domain.KindCollection, Database: "analytics", Name: "invoices",
	}) {
		t.Fatal("anyResource must match")
	}
	user, db := splitMongoPrincipal("analyst@analytics", "")
	if user != "analyst" || db != "analytics" {
		t.Fatalf("principal = %q %q", user, db)
	}
	lines := mongoRoleLines([]mongoRole{{Role: "read", DB: "analytics"}})
	if len(lines) != 1 || lines[0] != "read@analytics" {
		t.Fatalf("role lines = %v", lines)
	}
}
