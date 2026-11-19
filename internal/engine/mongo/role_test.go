package mongo

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMongoRoleRef(t *testing.T) {
	db, name, err := mongoRoleRef(domain.RoleRef{Name: "analyst@analytics"})
	if err != nil || db != "analytics" || name != "analyst" {
		t.Fatalf("role = %q/%q, %v", db, name, err)
	}
	db, name, err = mongoRoleRef(domain.RoleRef{Database: "admin", Name: "app_role"})
	if err != nil || db != "admin" || name != "app_role" {
		t.Fatalf("role = %q/%q, %v", db, name, err)
	}
	if _, _, err := mongoRoleRef(domain.RoleRef{Name: "app_role"}); err == nil {
		t.Fatal("missing database accepted")
	}
}

func TestMongoGrantBuildsResourceAndActions(t *testing.T) {
	actions, resource, label, err := mongoGrant(domain.PrivilegeGrant{
		Role:       domain.RoleRef{Database: "analytics", Name: "read_role"},
		Target:     domain.ObjectRef{Kind: domain.KindCollection, Database: "analytics", Name: "customers"},
		Privileges: []string{"find", "update", "find"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[0] != "find" || actions[1] != "update" {
		t.Fatalf("actions = %v", actions)
	}
	if values := mongoResourceValues(resource); values["db"] != "analytics" || values["collection"] != "customers" {
		t.Fatalf("resource = %v", values)
	}
	if !strings.Contains(label, "collection:customers") {
		t.Fatalf("label = %q", label)
	}
}

func TestMongoGrantRejectsUnknownAction(t *testing.T) {
	_, _, _, err := mongoGrant(domain.PrivilegeGrant{
		Role:       domain.RoleRef{Database: "analytics", Name: "read_role"},
		Target:     domain.ObjectRef{Kind: domain.KindCollection, Database: "analytics", Name: "customers"},
		Privileges: []string{"dropDatabase"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("err = %v", err)
	}
}

func TestMongoRoleDocumentShape(t *testing.T) {
	role := bson.D{{Key: "role", Value: "app_role"}, {Key: "db", Value: "analytics"}}
	if role[0].Key != "role" || role[1].Value != "analytics" {
		t.Fatalf("role doc = %v", role)
	}
}
