package domain

// Structure sources.
const (
	StructureCatalog = "catalog"
	StructureSample  = "sample"
)

// StructureField is one table column or one sampled document field.
type StructureField struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	// Present and Sampled describe MongoDB field presence. SQL structures
	// leave them zero.
	Present int
	Sampled int
}

// Structure is a table's columns or a collection's sampled field list.
type Structure struct {
	Fields  []StructureField
	Sampled int
	Source  string
}

// Index is one engine index. SizeBytes is -1 when the engine cannot expose a
// per-index size. Stats carries engine-specific extra information such as
// cardinality, sparse, or TTL.
type Index struct {
	Name       string
	Columns    []string
	Unique     bool
	Method     string
	SizeBytes  int64
	Definition string
	Stats      string
}
