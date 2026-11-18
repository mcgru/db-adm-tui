package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestSampleStructureCountsNestedPresence(t *testing.T) {
	docs := []bson.D{
		{
			{Key: "_id", Value: int32(1)},
			{Key: "profile", Value: bson.D{
				{Key: "email", Value: "a@example.test"},
				{Key: "name", Value: "Alice"},
			}},
			{Key: "tags", Value: bson.A{"one", "two"}},
		},
		{
			{Key: "_id", Value: int32(2)},
			{Key: "profile", Value: bson.D{
				{Key: "email", Value: nil},
			}},
		},
	}
	structure := sampleStructure(docs)
	if structure.Source != domain.StructureSample || structure.Sampled != 2 {
		t.Fatalf("structure = %+v", structure)
	}
	got := map[string]domain.StructureField{}
	for _, field := range structure.Fields {
		got[field.Name] = field
	}
	for _, name := range []string{"_id", "profile", "profile.email", "profile.name", "tags"} {
		if _, ok := got[name]; !ok {
			t.Fatalf("field %q missing: %+v", name, structure.Fields)
		}
	}
	if got["_id"].Present != 2 || got["_id"].Type != "number" {
		t.Fatalf("_id = %+v", got["_id"])
	}
	if got["profile"].Present != 2 || got["profile"].Type != "object" {
		t.Fatalf("profile = %+v", got["profile"])
	}
	if got["profile.name"].Present != 1 || got["profile.name"].Type != "string" {
		t.Fatalf("profile.name = %+v", got["profile.name"])
	}
	if got["tags"].Type != "array" {
		t.Fatalf("tags = %+v", got["tags"])
	}
	if structure.Fields[0].Name != "_id" {
		t.Fatalf("_id is not first: %+v", structure.Fields)
	}
}

func TestMongoIndexMethod(t *testing.T) {
	cases := []struct {
		keys bson.D
		want string
	}{
		{bson.D{{Key: "id", Value: int32(1)}}, "btree"},
		{bson.D{{Key: "body", Value: "text"}}, "text"},
		{bson.D{{Key: "loc", Value: "2dsphere"}}, "2dsphere"},
		{nil, "mongo"},
	}
	for _, tc := range cases {
		if got := mongoIndexMethod(tc.keys); got != tc.want {
			t.Fatalf("mongoIndexMethod(%+v) = %q, want %q", tc.keys, got, tc.want)
		}
	}
}
