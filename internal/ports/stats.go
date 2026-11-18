package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// StatsReader exposes structure and index metadata for one object.
type StatsReader interface {
	Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error)
	Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error)
}
