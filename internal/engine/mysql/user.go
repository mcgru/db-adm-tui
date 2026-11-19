package mysql

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.UserAdmin = (*connection)(nil)

func (c *connection) CreateUser(ctx context.Context, spec domain.UserSpec) (domain.MutationResult, error) {
	account, err := mysqlUserAccount(spec.Name, spec.Host)
	if err != nil {
		return domain.MutationResult{}, err
	}
	if spec.Password == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password is empty")
	}
	if strings.ContainsAny(spec.Password, "\n\r") {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password must not contain line breaks")
	}
	statement := "CREATE USER " + account + " IDENTIFIED BY " + quoteMySQLLiteral(spec.Password)
	audit := "CREATE USER " + account + " IDENTIFIED BY '***'"
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
	statement := "DROP USER " + quoteMySQLAccount(name)
	result, err := c.conn.ExecContext(ctx, statement)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("user", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func mysqlUserAccount(name, host string) (string, error) {
	name = strings.TrimSpace(name)
	host = strings.TrimSpace(host)
	if name == "" {
		return "", domain.NewError(domain.KindConfig, "user", "the user name is empty")
	}
	if index := strings.LastIndex(name, "@"); index >= 0 {
		if host == "" {
			host = name[index+1:]
		}
		name = name[:index]
	}
	if host == "" {
		host = "%"
	}
	if name == "" || host == "" {
		return "", domain.NewError(domain.KindConfig, "user", "the user account is not valid")
	}
	return quoteMySQLAccount(name + "@" + host), nil
}

func quoteMySQLLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "'", "''")
	return "'" + value + "'"
}
