package domain

// Estimate sources for a data page. An empty source means the total is
// unavailable; the UI never renders that as zero.
const (
	EstimateCatalog    = "catalog estimate"
	EstimateCollection = "collection estimate"
)

// DataQuery asks for one offset window of a table or collection.
type DataQuery struct {
	Ref    ObjectRef
	Offset int
	Limit  int
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
