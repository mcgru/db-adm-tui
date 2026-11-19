package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// StatsReader exposes statistics, structure, index metadata, and exact counts
// for one object.
type StatsReader interface {
	Statistics(ctx context.Context, ref domain.ObjectRef) (domain.Statistics, error)
	Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error)
	Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error)
	ExactCount(ctx context.Context, ref domain.ObjectRef) (int64, error)
}
