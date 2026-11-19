package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleAdmin = (*connection)(nil)

var pgRolePrivileges = map[domain.ObjectKind]map[string]bool{
	domain.KindDatabase: {"CONNECT": true, "CREATE": true, "TEMPORARY": true, "ALL": true},
	domain.KindSchema:   {"USAGE": true, "CREATE": true, "ALL": true},
	domain.KindTable: {
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"TRUNCATE": true, "REFERENCES": true, "TRIGGER": true, "ALL": true,
	},
	domain.KindView: {
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"TRUNCATE": true, "REFERENCES": true, "TRIGGER": true, "ALL": true,
	},
}

func (c *connection) CreateRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	statement, err := pgRoleStatement("CREATE ROLE", role, "")
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) DropRole(ctx context.Context, role domain.RoleRef) (domain.MutationResult, error) {
	statement, err := pgRoleStatement("DROP ROLE", role, "")
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) GrantRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	statement, err := pgMembershipStatement("GRANT", membership)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) RevokeRole(ctx context.Context, membership domain.RoleMembership) (domain.MutationResult, error) {
	statement, err := pgMembershipStatement("REVOKE", membership)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) GrantPrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	statement, err := pgPrivilegeStatement("GRANT", grant)
	if err != nil {
		return domain.MutationResult{}, err
	}
	return c.execRole(ctx, statement)
}

func (c *connection) RevokePrivileges(ctx context.Context, grant domain.PrivilegeGrant) (domain.MutationResult, error) {
	statement, err := pgPrivilegeStatement("REVOKE", grant)
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

func pgRoleStatement(verb string, role domain.RoleRef, suffix string) (string, error) {
	if strings.TrimSpace(role.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role name is empty")
	}
	return verb + " " + quoteIdentifier(role.Name) + suffix, nil
}

func pgMembershipStatement(verb string, membership domain.RoleMembership) (string, error) {
	if strings.TrimSpace(membership.Role.Name) == "" || strings.TrimSpace(membership.Member.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role or member name is empty")
	}
	if membership.Role.Name == membership.Member.Name {
		return "", domain.NewError(domain.KindConfig, "role", "a role cannot be granted to itself")
	}
	preposition := " TO "
	if verb == "REVOKE" {
		preposition = " FROM "
	}
	return verb + " " + quoteIdentifier(membership.Role.Name) + preposition +
		quoteIdentifier(membership.Member.Name), nil
}

func pgPrivilegeStatement(verb string, grant domain.PrivilegeGrant) (string, error) {
	if strings.TrimSpace(grant.Role.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "role", "the role name is empty")
	}
	allowed := pgRolePrivileges[grant.Target.Kind]
	if allowed == nil {
		return "", domain.NewError(domain.KindConfig, "role", "this target cannot receive privileges")
	}
	privileges, err := nativePrivileges(grant.Privileges, allowed)
	if err != nil {
		return "", err
	}
	target, err := pgTarget(grant.Target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s ON %s TO %s", verb, privileges, target,
		quoteIdentifier(grant.Role.Name)), nil
}

func nativePrivileges(values []string, allowed map[string]bool) (string, error) {
	if len(values) == 0 {
		return "", domain.NewError(domain.KindConfig, "role", "choose at least one privilege")
	}
	seen := map[string]bool{}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.ToUpper(strings.TrimSpace(value))
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

func pgTarget(target domain.ObjectRef) (string, error) {
	switch target.Kind {
	case domain.KindDatabase:
		name := target.Database
		if name == "" {
			name = target.Name
		}
		if name == "" {
			return "", domain.NewError(domain.KindConfig, "role", "the database is empty")
		}
		return "DATABASE " + quoteIdentifier(name), nil
	case domain.KindSchema:
		name := target.Schema
		if name == "" {
			name = target.Name
		}
		if name == "" {
			return "", domain.NewError(domain.KindConfig, "role", "the schema is empty")
		}
		return "SCHEMA " + quoteIdentifier(name), nil
	case domain.KindTable, domain.KindView:
		if target.Schema == "" || target.Name == "" {
			return "", domain.NewError(domain.KindConfig, "role", "the table is not fully qualified")
		}
		return "TABLE " + quoteIdentifier(target.Schema) + "." + quoteIdentifier(target.Name), nil
	default:
		return "", domain.NewError(domain.KindConfig, "role", "unsupported PostgreSQL grant target")
	}
}
