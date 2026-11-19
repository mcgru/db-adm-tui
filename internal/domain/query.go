package domain

import "time"

// QueryRequest is one logical statement bound to the current database. Ref is
// the selected collection for MongoDB and is otherwise informational.
type QueryRequest struct {
	Statement string
	Database  string
	Ref       ObjectRef
}

// QueryPage is one materialized page of a forward-only query cursor. HasMore
// reports that the cursor still holds rows; CapReached reports that the
// 10,000-row window cap stopped the drain.
type QueryPage struct {
	Grid       Grid
	HasMore    bool
	CapReached bool
	Elapsed    time.Duration
}

// HistoryEntry is one executed statement. The app writes entries; the TUI
// only reads the statement list for Up/Down history.
type HistoryEntry struct {
	Time       time.Time     `json:"ts"`
	Profile    string        `json:"profile"`
	Engine     Engine        `json:"engine"`
	Statement  string        `json:"statement"`
	Mode       string        `json:"mode"`
	DurationMs int64         `json:"duration_ms"`
	Status     string        `json:"status"`
	Affected   int64         `json:"affected,omitempty"`
	Duration   time.Duration `json:"-"`
}
