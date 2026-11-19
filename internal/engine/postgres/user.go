package postgres

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.UserAdmin = (*connection)(nil)

func (c *connection) CreateUser(ctx context.Context, spec domain.UserSpec) (domain.MutationResult, error) {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the user name is empty")
	}
	if spec.Password == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password is empty")
	}
	if strings.ContainsAny(spec.Password, "\n\r") {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password must not contain line breaks")
	}
	statement := "CREATE ROLE " + quoteIdentifier(name) + " LOGIN PASSWORD " + quoteLiteral(spec.Password)
	audit := "CREATE ROLE " + quoteIdentifier(name) + " LOGIN PASSWORD '***'"
	result, err := c.conn.ExecContext(ctx, statement)
	if err != nil {
		return domain.MutationResult{Statement: audit}, safeError("user", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: audit, Affected: affected}, nil
}

func (c *connection) DropUser(ctx context.Context, user domain.RoleRef) (domain.MutationResult, error) {
	name := strings.TrimSpace(user.Name)
	if name == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the user name is empty")
	}
	statement := "DROP ROLE " + quoteIdentifier(name)
	result, err := c.conn.ExecContext(ctx, statement)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("user", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
