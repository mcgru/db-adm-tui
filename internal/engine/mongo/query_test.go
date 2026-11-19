package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/mql"
)

func TestMongoDocumentConversion(t *testing.T) {
	value := mql.Object{
		{Key: "name", Value: "a"},
		{Key: "n", Value: int64(3)},
		{Key: "nested", Value: mql.Object{{Key: "ok", Value: true}}},
		{Key: "list", Value: mql.Array{int64(1), "two"}},
	}
	document, err := mongoDocument(value)
	if err != nil {
		t.Fatal(err)
	}
	if len(document) != 4 || document[0].Key != "name" || document[0].Value != "a" {
		t.Fatalf("document = %+v", document)
	}
	nested, ok := document[2].Value.(bson.D)
	if !ok || len(nested) != 1 || nested[0].Key != "ok" {
		t.Fatalf("nested = %#v", document[2].Value)
	}
	list, ok := document[3].Value.(bson.A)
	if !ok || len(list) != 2 {
		t.Fatalf("list = %#v", document[3].Value)
	}
}

func TestMongoValueObjectIDAndDate(t *testing.T) {
	id, err := mongoValue(mql.ObjectID("507f1f77bcf86cd799439011"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := id.(bson.ObjectID); !ok {
		t.Fatalf("id = %#v", id)
	}
	if _, err := mongoValue(mql.ObjectID("bad")); err == nil {
		t.Fatal("invalid ObjectId accepted")
	}
}

func TestMongoDocumentsGridUnionsFields(t *testing.T) {
	grid := mongoDocumentsGrid([]bson.D{
		{{Key: "_id", Value: int32(1)}, {Key: "a", Value: "x"}},
		{{Key: "_id", Value: int32(2)}, {Key: "b", Value: "y"}},
	})
	if len(grid.Columns) != 3 || grid.Columns[0].Name != "_id" || grid.Columns[2].Name != "b" {
		t.Fatalf("columns = %+v", grid.Columns)
	}
	if len(grid.Rows) != 2 || grid.Rows[0].Cells[2].Display() != "" {
		t.Fatalf("rows = %+v", grid.Rows)
	}
}

func TestSliceCursorPages(t *testing.T) {
	cursor := &sliceCursor{grid: domainGrid(3)}
	first, err := cursor.Next(nil, 2)
	if err != nil || len(first.Grid.Rows) != 2 || !first.HasMore {
		t.Fatalf("first = %+v, %v", first, err)
	}
	second, err := cursor.Next(nil, 2)
	if err != nil || len(second.Grid.Rows) != 1 || second.HasMore {
		t.Fatalf("second = %+v, %v", second, err)
	}
}

func domainGrid(rows int) domain.Grid {
	grid := domain.Grid{Columns: []domain.Column{{Name: "n"}}}
	for i := 0; i < rows; i++ {
		grid.Rows = append(grid.Rows, domain.Row{Cells: []domain.Cell{{Text: "x", Kind: domain.CellText}}})
	}
	return grid
}
