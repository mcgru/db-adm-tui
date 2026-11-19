package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleAdmin = (*connection)(nil)

var mysqlRolePrivileges = map[domain.ObjectKind]map[string]bool{
	domain.KindDatabase: {
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "INDEX": true,
		"REFERENCES": true, "CREATE TEMPORARY TABLES": true, "LOCK TABLES": true,
		"EXECUTE": true, "EVENT": true, "TRIGGER": true, "ALL PRIVILEGES": true,
	},
	domain.KindSchema: {
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "INDEX": true,
		"REFERENCES": true, "CREATE TEMPORARY TABLES": true, "LOCK TABLES": true,
		"EXECUTE": true, "EVENT": true, "TRIGGER": true, "ALL PRIVILEGES": true,
	},
	domain.KindTable: {
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "INDEX": true,
		"REFERENCES": true, "TRIGGER": true, "ALL PRIVILEGES": true,
	},
}

func (c *connection) CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	statement, err := mysqlRoleStatement("CREATE ROLE", role, "")
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	statement, err := mysqlRoleStatement("DROP ROLE", role, "")
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	statement, err := mysqlMembershipStatement("GRANT", membership)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	statement, err := mysqlMembershipStatement("REVOKE", membership)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	statement, err := mysqlPrivilegeStatement("GRANT", grant)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	statement, err := mysqlPrivilegeStatement("REVOKE", grant)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) execRole(ctx context.Context, statement string) (domain.MutationResult, error) {
	result, err := c.conn.ExecContext(ctx, statement)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("role", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func mysqlRoleStatement(verb string, role domain.RoleRef, suffix string) (string, error) {
	if strings.TrimSpace(role.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role name is empty")
	}
	return verb + " " + quoteMySQLAccount(role.Name) + suffix, nil
}

func mysqlMembershipStatement(verb string, membership domain.RoleMembership) (string, error) {
	if strings.TrimSpace(membership.Role.Name) == "" || strings.TrimSpace(membership.Member.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role or member name is empty")
	}
	if membership.Role.Name == membership.Member.Name {
		return "", domain.NewError(domain.KindConfig, "role", "a role cannot be granted to itself")
	}
	if verb == "REVOKE" {
		return "REVOKE " + quoteMySQLAccount(membership.Role.Name) +
			" FROM " + quoteMySQLAccount(membership.Member.Name), nil
	}
	return "GRANT " + quoteMySQLAccount(membership.Role.Name) +
		" TO " + quoteMySQLAccount(membership.Member.Name), nil
}

func mysqlPrivilegeStatement(verb string, grant domain.PrivilegeGrant) (string, error) {
	if strings.TrimSpace(grant.Role.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role name is empty")
	}
	allowed := mysqlRolePrivileges[grant.Target.Kind]
	if allowed == nil {
		return "", domain.NewError(domain.KindConfig, "role", "this target cannot receive privileges")
	}
	privileges, err := mysqlNativePrivileges(grant.Privileges, allowed)
	if err != nil {
		return "", err
	}
	target, err := mysqlGrantTarget(grant.Target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s ON %s TO %s", verb, privileges, target,
		quoteMySQLAccount(grant.Role.Name)), nil
}

func mysqlNativePrivileges(values []string, allowed map[string]bool) (string, error) {
	if len(values) == 0 {
		return "", domain.NewError(domain.KindConfig, "role", "choose at least one privilege")
	}
	seen := map[string]bool{}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.ToUpper(strings.TrimSpace(value))
		if name == "ALL" {
			name = "ALL PRIVILEGES"
		}
		if !allowed[name] {
			return "", domain.NewError(domain.KindConfig, "role",
				"unsupported privilege "+value+" for this target")
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		parts = append(parts, name)
	}
	return strings.Join(parts, ", "), nil
}

func mysqlGrantTarget(target domain.ObjectRef) (string, error) {
	database := target.Database
	if database == "" && target.Kind != domain.KindDatabase && target.Kind != domain.KindSchema {
		database = target.Schema
	}
	switch target.Kind {
	case domain.KindDatabase, domain.KindSchema:
		if database == "" {
			database = target.Name
		}
		if database == "" {
			return "", domain.NewError(domain.KindConfig, "role", "the database is empty")
		}
		return quoteIdentifier(database) + ".*", nil
	case domain.KindTable:
		if database == "" || target.Name == "" {
			return "", domain.NewError(domain.KindConfig, "role", "the table is not fully qualified")
		}
		return quoteIdentifier(database) + "." + quoteIdentifier(target.Name), nil
	default:
		return "", domain.NewError(domain.KindConfig, "role", "unsupported MySQL grant target")
	}
}

func splitMySQLAccount(name string) (string, string) {
	if index := strings.LastIndex(name, "@"); index >= 0 {
		return name[:index], name[index+1:]
	}
	return name, "%"
}

func quoteMySQLAccount(name string) string {
	user, host := splitMySQLAccount(name)
	return "'" + quoteMySQLString(user) + "'@'" + quoteMySQLString(host) + "'"
}
