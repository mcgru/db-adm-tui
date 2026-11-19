package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// RowAdmin inserts, updates, and deletes rows/documents. Update and delete
// accept one or more row key sets so the UI can batch a selection.
type RowAdmin interface {
	InsertRow(ctx context.Context, spec domain.RowInsert) (domain.MutationResult, error)
	UpdateRows(ctx context.Context, spec domain.RowUpdate) (domain.MutationResult, error)
	DeleteRows(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error)
}
