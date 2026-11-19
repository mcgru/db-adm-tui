package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// UserAdmin creates and drops native users. Password-bearing operations must
// return redacted MutationResult statements.
type UserAdmin interface {
	CreateUser(ctx context.Context, spec domain.UserSpec) (domain.MutationResult, error)
	DropUser(ctx context.Context, user domain.RoleRef) (domain.MutationResult, error)
}
