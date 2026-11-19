package mongo

import (
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMongoUserRef(t *testing.T) {
	db, name, err := mongoUserRef(domain.RoleRef{Name: "analyst@analytics", Kind: domain.KindUser})
	if err != nil || db != "analytics" || name != "analyst" {
		t.Fatalf("ref = %q/%q, %v", db, name, err)
	}
	db, name, err = mongoUserRef(domain.RoleRef{Database: "appdb", Name: "app_user"})
	if err != nil || db != "appdb" || name != "app_user" {
		t.Fatalf("ref = %q/%q, %v", db, name, err)
	}
	if _, _, err := mongoUserRef(domain.RoleRef{Name: "app_user"}); err == nil {
		t.Fatal("missing database accepted")
	}
}
