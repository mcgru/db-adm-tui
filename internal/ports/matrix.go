package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// RoleMatrixReader loads structured membership and privilege grids.
type RoleMatrixReader interface {
	RoleMatrix(ctx context.Context) (domain.RoleMatrix, error)
	PrivilegeMatrix(ctx context.Context, objects []domain.ObjectRef) (domain.PrivilegeMatrix, error)
}
