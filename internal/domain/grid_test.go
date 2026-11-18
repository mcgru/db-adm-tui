package domain

import (
	"strings"
	"testing"
	"time"
)

func TestGridDistinguishesNullFromEmpty(t *testing.T) {
	empty := TextCell("")
	null := NullCell()
	if empty.Null || empty.Display() != "" {
		t.Fatalf("empty cell = %+v, display %q", empty, empty.Display())
	}
	if !null.Null || null.Display() != "null" {
		t.Fatalf("null cell = %+v, display %q", null, null.Display())
	}
	if null.Kind != CellNull || empty.Kind != CellText {
		t.Fatalf("kinds = %q, %q", null.Kind, empty.Kind)
	}
}

func TestCellFromValueNullEmptyAndKinds(t *testing.T) {
	if cell := CellFromValue(nil, CellText); !cell.Null {
		t.Fatalf("nil value = %+v", cell)
	}
	if cell := CellFromValue("", CellText); cell.Null || cell.Text != "" {
		t.Fatalf("empty string = %+v", cell)
	}
	if cell := CellFromValue(true, CellText); cell.Text != "true" || cell.Kind != CellBool {
		t.Fatalf("bool = %+v", cell)
	}
	if cell := CellFromValue(int64(42), CellNumber); cell.Text != "42" || cell.Kind != CellNumber {
		t.Fatalf("number = %+v", cell)
	}
	when := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	if cell := CellFromValue(when, CellDatetime); cell.Kind != CellDatetime || !strings.HasPrefix(cell.Text, "2026-11-18T12:00:00") {
		t.Fatalf("datetime = %+v", cell)
	}
	if cell := CellFromValue([]byte(`{"b": 1, "a": 2}`), CellJSON); cell.Kind != CellJSON || cell.Text != `{"b":1,"a":2}` {
		t.Fatalf("json = %+v", cell)
	}
	if cell := CellFromValue([]byte{1, 2, 3}, CellBinary); cell.Kind != CellBinary || cell.Text != "<binary 3 bytes>" {
		t.Fatalf("binary = %+v", cell)
	}
	if cell := CellFromValue([]byte("abc"), CellText); cell.Text != "abc" || cell.Kind != CellText {
		t.Fatalf("text bytes = %+v", cell)
	}
}

func TestCompactJSONText(t *testing.T) {
	got := CompactJSONText(" {\n \"a\" : [1, 2]\n} ")
	if got != `{"a":[1,2]}` {
		t.Fatalf("compact = %q", got)
	}
	if got := CompactJSONText("not json"); got != "not json" {
		t.Fatalf("invalid json changed: %q", got)
	}
	if got := CompactJSONText(""); got != "" {
		t.Fatalf("empty json changed: %q", got)
	}
	got, err := CompactJSON(map[string]any{"b": 1, "a": 2})
	if err != nil || got != `{"a":2,"b":1}` {
		t.Fatalf("marshal = %q, %v", got, err)
	}
}

func TestGridApproxBytesAndColumnIndex(t *testing.T) {
	grid := Grid{
		Columns: []Column{{Name: "id"}, {Name: "note"}},
		Rows: []Row{
			{Cells: []Cell{TextCell("1"), TextCell("hello")}},
			{Cells: []Cell{TextCell("2"), NullCell()}},
		},
	}
	if grid.ApproxBytes() <= 0 {
		t.Fatal("approx bytes must be positive")
	}
	bigger := grid
	bigger.Rows = append(bigger.Rows, Row{Cells: []Cell{TextCell("3"), TextCell(strings.Repeat("x", 128))}})
	if bigger.ApproxBytes() <= grid.ApproxBytes() {
		t.Fatal("approx bytes did not grow with content")
	}
	if grid.ColumnIndex("note") != 1 || grid.ColumnIndex("missing") != -1 {
		t.Fatalf("column index = %d/%d", grid.ColumnIndex("note"), grid.ColumnIndex("missing"))
	}
}

func TestDataPageEstimateAvailability(t *testing.T) {
	page := DataPage{EstimatedTotal: 12480, EstimateSource: EstimateCatalog}
	if !page.EstimateAvailable() {
		t.Fatal("catalog estimate must be available")
	}
	page = DataPage{EstimatedTotal: -1}
	if page.EstimateAvailable() {
		t.Fatal("missing source must be unavailable")
	}
}
