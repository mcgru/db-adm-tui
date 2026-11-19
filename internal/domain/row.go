package domain

import "strings"

// RowValue is one column value entered in a structured row form. Value is text;
// the engine casts it to the native column type. Null and Default are mutually
// exclusive; Default omits the column from the statement.
type RowValue struct {
	Column  string
	Value   string
	Null    bool
	Default bool
	Kind    CellKind
}

// RowInsert is one structured row/document insert.
type RowInsert struct {
	Ref      ObjectRef
	Values   []RowValue
	Document string // MongoDB extended JSON document
}

// RowDelete identifies one or more rows/documents by their native primary key
// or _id. Each inner slice is one row's key set.
type RowDelete struct {
	Ref  ObjectRef
	Rows [][]RowValue
}

// RowUpdate updates one column/field on one or more rows/documents. Each inner
// slice is one row's key set.
type RowUpdate struct {
	Ref    ObjectRef
	Column string
	Value  RowValue
	Rows   [][]RowValue
}

// RowKeyLabel renders a safe one-line key summary for confirmations.
func RowKeyLabel(keys []RowValue) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := key.Value
		if key.Null {
			value = "null"
		}
		parts = append(parts, key.Column+"="+value)
	}
	return strings.Join(parts, ", ")
}
