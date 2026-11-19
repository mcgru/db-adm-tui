package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// RowAdmin inserts and deletes one native row/document at a time.
type RowAdmin interface {
	InsertRow(ctx context.Context, spec domain.RowInsert) (domain.MutationResult, error)
	DeleteRow(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error)
}
