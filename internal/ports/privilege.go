package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// PrivilegeReader exposes native grants and memberships for one context or
// principal.
type PrivilegeReader interface {
	Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error)
}
