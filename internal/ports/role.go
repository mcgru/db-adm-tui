package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// RoleAdmin performs native role, membership, and privilege mutations.
type RoleAdmin interface {
	CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error)
	DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error)
	GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error)
	RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error)
	GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error)
	RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error)
}
