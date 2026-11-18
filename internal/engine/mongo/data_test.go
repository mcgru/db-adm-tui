package mongo

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildMongoGridUnionsFields(t *testing.T) {
	docs := []bson.D{
		{{Key: "name", Value: "Alice"}, {Key: "_id", Value: int32(1)}, {Key: "meta", Value: bson.D{{Key: "active", Value: true}}}},
		{{Key: "_id", Value: int32(2)}, {Key: "email", Value: "bob@example.test"}},
		{{Key: "_id", Value: int32(3)}, {Key: "name", Value: nil}},
	}
	grid, hasMore := buildMongoGrid(docs, 10)
	if hasMore {
		t.Fatal("unexpected HasMore")
	}
	var names []string
	for _, column := range grid.Columns {
		names = append(names, column.Name)
	}
	want := []string{"_id", "name", "meta", "email"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("columns = %v, want %v", names, want)
	}
	if grid.Rows[0].Cells[0].Text != "1" || grid.Rows[0].Cells[0].Kind != domain.CellNumber {
		t.Fatalf("_id cell = %+v", grid.Rows[0].Cells[0])
	}
	if !strings.Contains(grid.Rows[0].Cells[2].Text, `"active":true`) || grid.Rows[0].Cells[2].Kind != domain.CellJSON {
		t.Fatalf("nested cell = %+v", grid.Rows[0].Cells[2])
	}
	if grid.Rows[1].Cells[1].Null || grid.Rows[1].Cells[1].Text != "" {
		t.Fatalf("missing field = %+v", grid.Rows[1].Cells[1])
	}
	if !grid.Rows[2].Cells[1].Null {
		t.Fatalf("explicit null = %+v", grid.Rows[2].Cells[1])
	}
}

func TestBuildMongoGridLimitAndMore(t *testing.T) {
	docs := []bson.D{
		{{Key: "_id", Value: int32(1)}},
		{{Key: "_id", Value: int32(2)}},
		{{Key: "_id", Value: int32(3)}},
	}
	grid, hasMore := buildMongoGrid(docs, 2)
	if !hasMore || len(grid.Rows) != 2 {
		t.Fatalf("hasMore=%v rows=%d", hasMore, len(grid.Rows))
	}
}

func TestMongoCellTypes(t *testing.T) {
	when := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		value any
		kind  domain.CellKind
		text  string
		null  bool
	}{
		{nil, domain.CellNull, "", true},
		{"x", domain.CellText, "x", false},
		{"", domain.CellText, "", false},
		{true, domain.CellBool, "true", false},
		{int64(7), domain.CellNumber, "7", false},
		{bson.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}, domain.CellText, "0102030405060708090a0b0c", false},
		{bson.NewDateTimeFromTime(when), domain.CellDatetime, "2026-11-18T12:00:00Z", false},
		{bson.Binary{Data: []byte{1, 2}}, domain.CellBinary, "<binary 2 bytes>", false},
		{bson.A{int32(1)}, domain.CellJSON, `[1]`, false},
	}
	for _, tc := range cases {
		cell := mongoCell(tc.value)
		if cell.Kind != tc.kind || cell.Text != tc.text || cell.Null != tc.null {
			t.Fatalf("mongoCell(%#v) = %+v, want kind=%s text=%q null=%v", tc.value, cell, tc.kind, tc.text, tc.null)
		}
	}
}
