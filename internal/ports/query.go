package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// QueryExecutor opens one forward-only result cursor. The caller owns the
// cursor and closes it when Esc releases the session lease.
type QueryExecutor interface {
	Query(ctx context.Context, request domain.QueryRequest) (QueryCursor, error)
}

// QueryCursor materializes result rows in batches. Next returns at most limit
// rows; HasMore reports that the cursor has more. Close is idempotent.
type QueryCursor interface {
	Next(ctx context.Context, limit int) (domain.QueryPage, error)
	Close(ctx context.Context) error
}

// HistoryStore persists executed statements per profile. Writes are
// best-effort; a failure must never block a query.
type HistoryStore interface {
	Load(ctx context.Context, profile string) ([]domain.HistoryEntry, error)
	Append(ctx context.Context, entry domain.HistoryEntry) error
}
