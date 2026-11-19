package domain

// LayoutSettings are the persisted per-profile workspace dimensions.
type LayoutSettings struct {
	TreeWidth   int `json:"tree_width"`
	QueryHeight int `json:"query_height"`
}
