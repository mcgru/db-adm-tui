package domain

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

// RowDelete identifies one row/document by its native primary key or _id.
type RowDelete struct {
	Ref  ObjectRef
	Keys []RowValue
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
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ", "
		}
		out += part
	}
	return out
}
