package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// DataReader fetches read-only offset windows from a table or collection.
type DataReader interface {
	Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error)
}
