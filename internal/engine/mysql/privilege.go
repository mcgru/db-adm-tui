package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.PrivilegeReader = (*connection)(nil)

// Privileges reports native SHOW GRANTS output plus granted and active roles.
// MySQL does not expose portable effective checks, so the report says so
// instead of inventing a cross-engine vocabulary.
func (c *connection) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	principal := ""
	current := true
	switch ref.Kind {
	case domain.KindUser, domain.KindRole:
		principal = ref.Name
		current = false
	}
	if principal == "" {
		if err := c.conn.QueryRowContext(ctx, "SELECT CURRENT_USER()").Scan(&principal); err != nil {
			return domain.PrivilegeReport{}, safeError("privileges", c.password, err)
		}
	}
	report := domain.PrivilegeReport{
		Principal: principal,
		Scope:     mysqlPrivilegeScope(ref),
		Note:      "effective checks unavailable; native grants and role state are shown",
	}
	report.Sections = append(report.Sections, c.showGrants(ctx, principal))
	report.Sections = append(report.Sections, c.activeRoles(ctx, principal, current)...)
	return report, nil
}

func mysqlPrivilegeScope(ref domain.ObjectRef) string {
	switch ref.Kind {
	case domain.KindTable:
		if ref.Database != "" {
			return "table " + ref.Database + "." + ref.Name
		}
		return "table " + ref.Name
	case domain.KindDatabase, domain.KindSchema:
		return "database " + ref.Database
	case domain.KindUser, domain.KindRole:
		return string(ref.Kind) + " " + ref.Name
	case domain.KindSecurity:
		return "connection"
	default:
		return string(ref.Kind)
	}
}

func (c *connection) showGrants(ctx context.Context, principal string) domain.PrivilegeSection {
	statement := "SHOW GRANTS FOR CURRENT_USER()"
	if principal != "" {
		user, host := splitMySQLPrincipal(principal)
		statement = "SHOW GRANTS FOR '" + quoteMySQLString(user) + "'@'" + quoteMySQLString(host) + "'"
	}
	rows, err := c.conn.QueryContext(ctx, statement)
	if err != nil {
		return domain.PrivilegeSection{Title: "grants", Err: mysqlPrivilegeError(c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return domain.PrivilegeSection{Title: "grants", Err: mysqlPrivilegeError(c.password, err)}
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return domain.PrivilegeSection{Title: "grants", Err: mysqlPrivilegeError(c.password, err)}
	}
	return domain.PrivilegeSection{Title: "grants", Lines: lines}
}

func (c *connection) activeRoles(ctx context.Context, principal string, current bool) []domain.PrivilegeSection {
	var sections []domain.PrivilegeSection
	if current {
		var active sql.NullString
		if err := c.conn.QueryRowContext(ctx, "SELECT CURRENT_ROLE()").Scan(&active); err != nil {
			sections = append(sections, domain.PrivilegeSection{
				Title: "active roles", Err: mysqlPrivilegeError(c.password, err),
			})
		} else {
			var lines []string
			if active.Valid && active.String != "" {
				lines = append(lines, active.String)
			}
			sections = append(sections, domain.PrivilegeSection{Title: "active roles", Lines: lines})
		}
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT ROLE_NAME, ROLE_HOST, IS_GRANTABLE
FROM information_schema.APPLICABLE_ROLES
WHERE GRANTEE = ?
ORDER BY ROLE_NAME, ROLE_HOST`, principal)
	if err != nil {
		sections = append(sections, domain.PrivilegeSection{
			Title: "granted roles", Err: mysqlPrivilegeError(c.password, err),
		})
		return sections
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var role, host, grantable string
		if err := rows.Scan(&role, &host, &grantable); err != nil {
			sections = append(sections, domain.PrivilegeSection{
				Title: "granted roles", Err: mysqlPrivilegeError(c.password, err),
			})
			return sections
		}
		line := role + "@" + host
		if grantable == "YES" {
			line += " · grantable"
		}
		lines = append(lines, line)
	}
	sections = append(sections, domain.PrivilegeSection{Title: "granted roles", Lines: lines})
	return sections
}

func splitMySQLPrincipal(principal string) (string, string) {
	if index := strings.LastIndex(principal, "@"); index >= 0 {
		return principal[:index], principal[index+1:]
	}
	return principal, "%"
}

func quoteMySQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func mysqlPrivilegeError(password string, err error) string {
	safe := safeError("privileges", password, err)
	if domain.KindOf(safe) == domain.KindPermission {
		return "not permitted. This account cannot read this section."
	}
	return safe.Error()
}
