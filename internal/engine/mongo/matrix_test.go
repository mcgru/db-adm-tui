package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMatrixRoleRefMarksBuiltinRoles(t *testing.T) {
	ref := matrixRoleRef(matrixRole{Role: "read", DB: "analytics", IsBuiltin: true})
	if !ref.Builtin || ref.Kind != domain.KindRole || ref.Database != "analytics" || ref.Name != "read@analytics" {
		t.Fatalf("ref = %+v", ref)
	}
	custom := matrixRoleRef(matrixRole{Role: "app_role", DB: "analytics"})
	if custom.Builtin {
		t.Fatalf("custom role marked builtin: %+v", custom)
	}
}

func TestMongoPrivilegeMatchesDatabaseScope(t *testing.T) {
	database := domain.ObjectRef{Engine: domain.EngineMongoDB, Kind: domain.KindDatabase, Database: "analytics", Name: "analytics"}
	collection := domain.ObjectRef{Engine: domain.EngineMongoDB, Kind: domain.KindCollection, Database: "analytics", Name: "customers"}
	collectionPrivilege := bson.D{{Key: "db", Value: "analytics"}, {Key: "collection", Value: "customers"}}
	if mongoPrivilegeMatches(collectionPrivilege, database) {
		t.Fatal("collection privilege matched the database cell")
	}
	if !mongoPrivilegeMatches(collectionPrivilege, collection) {
		t.Fatal("collection privilege did not match its collection")
	}
	databasePrivilege := bson.D{{Key: "db", Value: "analytics"}}
	if !mongoPrivilegeMatches(databasePrivilege, database) || !mongoPrivilegeMatches(databasePrivilege, collection) {
		t.Fatal("database privilege should cover the database and its collections")
	}
}

func TestMongoMatrixPrivilegeUsesExactResource(t *testing.T) {
	database := domain.ObjectRef{Engine: domain.EngineMongoDB, Kind: domain.KindDatabase, Database: "analytics", Name: "analytics"}
	collection := domain.ObjectRef{Engine: domain.EngineMongoDB, Kind: domain.KindCollection, Database: "analytics", Name: "customers"}
	databasePrivilege := bson.D{{Key: "db", Value: "analytics"}}
	collectionPrivilege := bson.D{{Key: "db", Value: "analytics"}, {Key: "collection", Value: "customers"}}
	clusterPrivilege := bson.D{{Key: "cluster", Value: true}}
	if !mongoMatrixPrivilegeMatches(databasePrivilege, database) {
		t.Fatal("database privilege should fill the database cell")
	}
	if mongoMatrixPrivilegeMatches(databasePrivilege, collection) {
		t.Fatal("database privilege must not fill a collection cell")
	}
	if !mongoMatrixPrivilegeMatches(collectionPrivilege, collection) {
		t.Fatal("collection privilege should fill its collection cell")
	}
	if mongoMatrixPrivilegeMatches(collectionPrivilege, database) {
		t.Fatal("collection privilege must not fill the database cell")
	}
	if mongoMatrixPrivilegeMatches(clusterPrivilege, database) {
		t.Fatal("cluster privilege has no exact matrix cell")
	}
}

func TestUniqueSortedActions(t *testing.T) {
	got := uniqueSorted([]string{"update", "find", "update"})
	if len(got) != 2 || got[0] != "find" || got[1] != "update" {
		t.Fatalf("actions = %v", got)
	}
}
