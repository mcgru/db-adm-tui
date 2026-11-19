package domain

import "strings"

// Estimate sources for a data page. An empty source means the total is
// unavailable; the UI never renders that as zero.
const (
	EstimateCatalog    = "catalog estimate"
	EstimateCollection = "collection estimate"
)

// ValidMongoPath reports whether field is a usable dotted document path for a
// read-only regexp filter. Segments must be non-empty and must not use the
// reserved $ operator marker.
func ValidMongoPath(field string) bool {
	if field == "" || strings.ContainsRune(field, 0) {
		return false
	}
	for _, segment := range strings.Split(field, ".") {
		if segment == "" || strings.HasPrefix(segment, "$") {
			return false
		}
	}
	return true
}

// DataFilter is one read-only server-side regexp filter over a single column
// or field. Field is the engine-facing path: a base column for SQL, a dotted
// path for MongoDB.
type DataFilter struct {
	Field           string
	Pattern         string
	CaseInsensitive bool
}

// DataQuery asks for one offset window of a table or collection. A nil Filter
// means the unfiltered table or collection.
type DataQuery struct {
	Ref    ObjectRef
	Offset int
	Limit  int
	Filter *DataFilter
}

// DataPage is one engine data page. HasMore is explicit so the cursor never
// guesses; EstimatedTotal is meaningful only when EstimateSource is set.
type DataPage struct {
	Grid           Grid
	Offset         int
	HasMore        bool
	EstimatedTotal int64
	EstimateSource string
	OrderKey       string
	OrderKeyKnown  bool
}

// EstimateAvailable reports whether the page carries a usable total estimate.
func (p DataPage) EstimateAvailable() bool {
	return p.EstimateSource != "" && p.EstimatedTotal >= 0
}
