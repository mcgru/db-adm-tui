package mongo

import (
	"context"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestClassifyMongoRole(t *testing.T) {
	cases := []struct {
		role string
		db   string
		want string
	}{
		{"root", "admin", "rw"},
		{"readWriteAnyDatabase", "admin", "rw"},
		{"readAnyDatabase", "admin", "ro"},
		{"readWrite", "analytics", "rw"},
		{"read", "analytics", "ro"},
		{"read", "other", ""},
		{"customRole", "analytics", ""},
	}
	for _, tc := range cases {
		state := classifyMongoRole(mongoAuthRole{role: tc.role, db: tc.db}, "analytics")
		got := ""
		if state.known {
			switch {
			case state.write:
				got = "rw"
			case state.read:
				got = "ro"
			}
		}
		if got != tc.want {
			t.Fatalf("classify(%s@%s) = %q, want %q", tc.role, tc.db, got, tc.want)
		}
	}
}

func TestMongoDatabaseNode(t *testing.T) {
	node := databaseNode("analytics")
	if node.Ref.Engine != domain.EngineMongoDB || node.Kind != domain.KindDatabase || !node.Expandable {
		t.Fatalf("node = %+v", node)
	}
}

func TestMongoOrderKey(t *testing.T) {
	c := &connection{}
	key, err := c.OrderKey(context.Background(), domain.ObjectRef{Kind: domain.KindCollection, Name: "customers"})
	if err != nil || key != "_id" {
		t.Fatalf("key = %q, %v", key, err)
	}
	key, err = c.OrderKey(context.Background(), domain.ObjectRef{Kind: domain.KindDatabase})
	if err != nil || key != "" {
		t.Fatalf("non-object key = %q, %v", key, err)
	}
}
