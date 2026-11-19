package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CellKind identifies how a grid cell was produced. It is a rendering hint, not
// a database type name.
type CellKind string

const (
	CellText     CellKind = "text"
	CellNumber   CellKind = "number"
	CellBool     CellKind = "bool"
	CellNull     CellKind = "null"
	CellJSON     CellKind = "json"
	CellBinary   CellKind = "binary"
	CellDatetime CellKind = "datetime"
)

// Cell is one rendered value. Null and an empty string are distinct: a null
// cell has Null set, while an empty string has Null unset and empty Text.
type Cell struct {
	Text string   `json:"text"`
	Null bool     `json:"null,omitempty"`
	Kind CellKind `json:"kind"`
}

// NullCell returns an explicit null cell.
func NullCell() Cell { return Cell{Null: true, Kind: CellNull} }

// TextCell returns a text cell. The empty string stays distinct from null.
func TextCell(text string) Cell { return Cell{Text: text, Kind: CellText} }

// Display returns the value the data grid draws. Null is rendered by its word.
func (c Cell) Display() string {
	if c.Null {
		return "null"
	}
	return c.Text
}

// ApproxBytes estimates the memory held by one cell for window accounting.
func (c Cell) ApproxBytes() int {
	return len(c.Text) + len(c.Kind) + 24
}

// Column is one grid column in engine order. Mongo collections use _id first,
// then first-seen field order.
type Column struct {
	Name string `json:"name"`
}

// ApproxBytes estimates the memory held by one column.
func (c Column) ApproxBytes() int { return len(c.Name) + 24 }

// Row is one ordered set of cells.
type Row struct {
	Cells []Cell `json:"cells"`
}

// ApproxBytes estimates the memory held by one row.
func (r Row) ApproxBytes() int {
	total := 24
	for _, cell := range r.Cells {
		total += cell.ApproxBytes()
	}
	return total
}

// Grid is the cross-engine data shape. The TUI renders only Grids.
type Grid struct {
	Columns []Column `json:"columns"`
	Rows    []Row    `json:"rows"`
}

// UniqueColumnNames makes result column names distinct by appending a
// positional suffix. Engines may return duplicate names; the grid merge keys
// columns by name, so duplicates must not collapse.
func UniqueColumnNames(names []string) []string {
	seen := make(map[string]int, len(names))
	result := make([]string, len(names))
	for i, name := range names {
		count := seen[name]
		seen[name] = count + 1
		if count == 0 {
			result[i] = name
			continue
		}
		result[i] = name + " (" + strconv.Itoa(count+1) + ")"
	}
	return result
}

// ApproxBytes estimates the memory held by a grid for the loaded-window cap.
func (g Grid) ApproxBytes() int {
	total := 0
	for _, column := range g.Columns {
		total += column.ApproxBytes()
	}
	for _, row := range g.Rows {
		total += row.ApproxBytes()
	}
	return total
}

// ColumnIndex returns the first column with the given name, or -1.
func (g Grid) ColumnIndex(name string) int {
	for i, column := range g.Columns {
		if column.Name == name {
			return i
		}
	}
	return -1
}

// CellFromValue converts a driver-scanned SQL value into a cell. Null stays
// distinct from an empty string, JSON is compacted, and binary values render as
// a short size marker rather than opaque bytes.
func CellFromValue(value any, kind CellKind) Cell {
	if value == nil {
		return NullCell()
	}
	switch v := value.(type) {
	case bool:
		return Cell{Text: strconv.FormatBool(v), Kind: CellBool}
	case time.Time:
		return Cell{Text: v.Format(time.RFC3339Nano), Kind: CellDatetime}
	case []byte:
		switch kind {
		case CellJSON:
			return Cell{Text: CompactJSONText(string(v)), Kind: CellJSON}
		case CellBinary:
			return Cell{Text: fmt.Sprintf("<binary %d bytes>", len(v)), Kind: CellBinary}
		default:
			return Cell{Text: string(v), Kind: kindOrText(kind)}
		}
	case string:
		switch kind {
		case CellJSON:
			return Cell{Text: CompactJSONText(v), Kind: CellJSON}
		case CellBinary:
			return Cell{Text: fmt.Sprintf("<binary %d bytes>", len(v)), Kind: CellBinary}
		}
		return Cell{Text: v, Kind: kindOrText(kind)}
	case float32, float64:
		return Cell{Text: fmt.Sprint(v), Kind: CellNumber}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return Cell{Text: fmt.Sprint(v), Kind: CellNumber}
	default:
		return Cell{Text: fmt.Sprint(v), Kind: kindOrText(kind)}
	}
}

func kindOrText(kind CellKind) CellKind {
	if kind == "" || kind == CellNull {
		return CellText
	}
	return kind
}

// CompactJSON renders v as compact JSON for nested documents and arrays.
func CompactJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CompactJSONText compacts a JSON document string. Invalid JSON is returned
// unchanged so the caller can still render the raw value.
func CompactJSONText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return text
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(trimmed)); err != nil {
		return text
	}
	return buf.String()
}
