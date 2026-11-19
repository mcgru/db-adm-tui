package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.PrivilegeReader = (*connection)(nil)

// Privileges reports native PostgreSQL grants, memberships, attributes, and
// has_* effective checks for the selected context or principal.
func (c *connection) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	principal := ""
	switch ref.Kind {
	case domain.KindUser, domain.KindRole:
		principal = ref.Name
	}
	if principal == "" {
		if err := c.conn.QueryRowContext(ctx, "SELECT current_user").Scan(&principal); err != nil {
			return domain.PrivilegeReport{}, safeError("privileges", c.password, err)
		}
	}
	report := domain.PrivilegeReport{
		Principal: principal,
		Scope:     postgresPrivilegeScope(ref),
	}
	roles := map[string]bool{}
	if attrs, err := c.roleAttributes(ctx, principal); err != nil {
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "attributes", Err: privilegeSectionError("privileges", c.password, err),
		})
	} else {
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "attributes", Lines: attrs,
		})
	}
	if memberships, rolesFound, err := c.memberships(ctx, principal); err != nil {
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "memberships", Err: privilegeSectionError("privileges", c.password, err),
		})
	} else {
		for role := range rolesFound {
			roles[role] = true
		}
		report.Sections = append(report.Sections, domain.PrivilegeSection{
			Title: "memberships", Lines: memberships,
		})
	}
	switch ref.Kind {
	case domain.KindTable:
		schema := ref.Schema
		if schema == "" {
			schema = "public"
		}
		report.Sections = append(report.Sections,
			c.tableGrants(ctx, principal, roles, schema, ref.Name)...)
		report.Sections = append(report.Sections,
			c.columnGrants(ctx, principal, roles, schema, ref.Name))
		report.Sections = append(report.Sections, c.tableEffective(ctx, principal, schema, ref.Name))
	case domain.KindDatabase:
		report.Sections = append(report.Sections,
			c.databaseGrants(ctx, principal, roles, ref.Database)...)
		report.Sections = append(report.Sections, c.databaseEffective(ctx, principal, ref.Database))
	case domain.KindSchema:
		report.Sections = append(report.Sections,
			c.schemaGrants(ctx, principal, roles, ref.Schema)...)
		report.Sections = append(report.Sections, c.schemaEffective(ctx, principal, ref.Schema))
	case domain.KindUser, domain.KindRole:
		report.Sections = append(report.Sections, c.roleTableGrants(ctx, principal))
	case domain.KindSecurity:
		// Attributes and memberships are the connection-level view.
	}
	return report, nil
}

func postgresPrivilegeScope(ref domain.ObjectRef) string {
	switch ref.Kind {
	case domain.KindTable:
		schema := ref.Schema
		if schema == "" {
			schema = "public"
		}
		return "table " + schema + "." + ref.Name
	case domain.KindDatabase:
		return "database " + ref.Database
	case domain.KindSchema:
		return "schema " + ref.Schema
	case domain.KindUser, domain.KindRole:
		return string(ref.Kind) + " " + ref.Name
	case domain.KindSecurity:
		return "connection"
	default:
		return string(ref.Kind)
	}
}

func (c *connection) roleAttributes(ctx context.Context, principal string) ([]string, error) {
	rows, err := c.conn.QueryContext(ctx, `
SELECT rolsuper, rolcreaterole, rolcreatedb, rolcanlogin,
       rolreplication, rolbypassrls, rolinherit
FROM pg_roles
WHERE rolname = $1`, principal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	var super, createrole, createdb, login, replication, bypassRLS, inherit bool
	if err := rows.Scan(&super, &createrole, &createdb, &login, &replication, &bypassRLS, &inherit); err != nil {
		return nil, err
	}
	bools := func(value bool, yes, no string) string {
		if value {
			return yes
		}
		return no
	}
	return []string{strings.Join([]string{
		bools(super, "SUPERUSER", "NOSUPERUSER"),
		bools(createrole, "CREATEROLE", "NOCREATEROLE"),
		bools(createdb, "CREATEDB", "NOCREATEDB"),
		bools(login, "LOGIN", "NOLOGIN"),
		bools(replication, "REPLICATION", "NOREPLICATION"),
		bools(bypassRLS, "BYPASSRLS", "NOBYPASSRLS"),
		bools(inherit, "INHERIT", "NOINHERIT"),
	}, " · ")}, nil
}

func (c *connection) memberships(ctx context.Context, principal string) ([]string, map[string]bool, error) {
	rows, err := c.conn.QueryContext(ctx, `
SELECT r.rolname, m.admin_option, pg_has_role($1, r.oid, 'USAGE')
FROM pg_auth_members m
JOIN pg_roles r ON r.oid = m.roleid
WHERE m.member = (SELECT oid FROM pg_roles WHERE rolname = $1)
ORDER BY r.rolname`, principal)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var lines []string
	roles := map[string]bool{}
	for rows.Next() {
		var role string
		var admin, inherited bool
		if err := rows.Scan(&role, &admin, &inherited); err != nil {
			return nil, nil, err
		}
		roles[role] = true
		line := "role " + role
		if inherited {
			line += " · inherited"
		} else {
			line += " · requires SET ROLE"
		}
		if admin {
			line += " · admin option"
		}
		lines = append(lines, line)
	}
	return lines, roles, rows.Err()
}

func (c *connection) tableGrants(ctx context.Context, principal string, roles map[string]bool, schema, table string) []domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT grantee, privilege_type, is_grantable
FROM information_schema.table_privileges
WHERE table_schema = $1 AND table_name = $2
ORDER BY grantee, privilege_type`, schema, table)
	if err != nil {
		return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var grantee, privilege, grantable string
		if err := rows.Scan(&grantee, &privilege, &grantable); err != nil {
			return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
		}
		if grantee != principal && grantee != "PUBLIC" && !roles[grantee] {
			continue
		}
		line := fmt.Sprintf("%s on %s.%s", privilege, schema, table)
		switch {
		case grantee == principal:
			line += " · direct"
		case grantee == "PUBLIC":
			line += " · PUBLIC"
		default:
			line += " · via role " + grantee
		}
		if grantable == "YES" {
			line += " · grantable"
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
	}
	return []domain.PrivilegeSection{{Title: "grants", Lines: lines}}
}

func (c *connection) columnGrants(ctx context.Context, principal string, roles map[string]bool, schema, table string) domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT grantee, column_name, privilege_type, is_grantable
FROM information_schema.column_privileges
WHERE table_schema = $1 AND table_name = $2
ORDER BY grantee, column_name, privilege_type`, schema, table)
	if err != nil {
		return domain.PrivilegeSection{Title: "column grants", Err: privilegeSectionError("privileges", c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var grantee, column, privilege, grantable string
		if err := rows.Scan(&grantee, &column, &privilege, &grantable); err != nil {
			return domain.PrivilegeSection{Title: "column grants", Err: privilegeSectionError("privileges", c.password, err)}
		}
		if grantee != principal && grantee != "PUBLIC" && !roles[grantee] {
			continue
		}
		line := fmt.Sprintf("%s (%s) on %s.%s", privilege, column, schema, table)
		switch {
		case grantee == principal:
			line += " · direct"
		case grantee == "PUBLIC":
			line += " · PUBLIC"
		default:
			line += " · via role " + grantee
		}
		if grantable == "YES" {
			line += " · grantable"
		}
		lines = append(lines, line)
	}
	return domain.PrivilegeSection{Title: "column grants", Lines: lines}
}

func (c *connection) tableEffective(ctx context.Context, principal, schema, table string) domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT t.p, has_table_privilege($1, format('%I.%I', $2, $3), t.p)
FROM unnest(ARRAY['SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER']) AS t(p)`,
		principal, schema, table)
	if err != nil {
		return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var privilege string
		var allowed bool
		if err := rows.Scan(&privilege, &allowed); err != nil {
			return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
		}
		lines = append(lines, fmt.Sprintf("%s · effective %s", privilege, yesNo(allowed)))
	}
	return domain.PrivilegeSection{Title: "effective", Lines: lines}
}

func (c *connection) databaseGrants(ctx context.Context, principal string, roles map[string]bool, database string) []domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT COALESCE(r.rolname, 'PUBLIC'), a.privilege_type
FROM pg_database d, aclexplode(d.datacl) a
LEFT JOIN pg_roles r ON r.oid = a.grantee
WHERE d.datname = $1
ORDER BY 1, 2`, database)
	if err != nil {
		return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var grantee, privilege string
		if err := rows.Scan(&grantee, &privilege); err != nil {
			return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
		}
		if grantee != principal && grantee != "PUBLIC" && !roles[grantee] {
			continue
		}
		line := fmt.Sprintf("%s on database %s", privilege, database)
		switch {
		case grantee == principal:
			line += " · direct"
		case grantee == "PUBLIC":
			line += " · PUBLIC"
		default:
			line += " · via role " + grantee
		}
		lines = append(lines, line)
	}
	return []domain.PrivilegeSection{{Title: "grants", Lines: lines}}
}

func (c *connection) databaseEffective(ctx context.Context, principal, database string) domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT t.p, has_database_privilege($1, $2, t.p)
FROM unnest(ARRAY['CONNECT','CREATE','TEMPORARY']) AS t(p)`, principal, database)
	if err != nil {
		return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var privilege string
		var allowed bool
		if err := rows.Scan(&privilege, &allowed); err != nil {
			return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
		}
		lines = append(lines, fmt.Sprintf("%s · effective %s", privilege, yesNo(allowed)))
	}
	return domain.PrivilegeSection{Title: "effective", Lines: lines}
}

func (c *connection) schemaGrants(ctx context.Context, principal string, roles map[string]bool, schema string) []domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT COALESCE(r.rolname, 'PUBLIC'), a.privilege_type
FROM pg_namespace n, aclexplode(n.nspacl) a
LEFT JOIN pg_roles r ON r.oid = a.grantee
WHERE n.nspname = $1
ORDER BY 1, 2`, schema)
	if err != nil {
		return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var grantee, privilege string
		if err := rows.Scan(&grantee, &privilege); err != nil {
			return []domain.PrivilegeSection{{Title: "grants", Err: privilegeSectionError("privileges", c.password, err)}}
		}
		if grantee != principal && grantee != "PUBLIC" && !roles[grantee] {
			continue
		}
		line := fmt.Sprintf("%s on schema %s", privilege, schema)
		switch {
		case grantee == principal:
			line += " · direct"
		case grantee == "PUBLIC":
			line += " · PUBLIC"
		default:
			line += " · via role " + grantee
		}
		lines = append(lines, line)
	}
	return []domain.PrivilegeSection{{Title: "grants", Lines: lines}}
}

func (c *connection) schemaEffective(ctx context.Context, principal, schema string) domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT t.p, has_schema_privilege($1, $2, t.p)
FROM unnest(ARRAY['USAGE','CREATE']) AS t(p)`, principal, schema)
	if err != nil {
		return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var privilege string
		var allowed bool
		if err := rows.Scan(&privilege, &allowed); err != nil {
			return domain.PrivilegeSection{Title: "effective", Err: privilegeSectionError("privileges", c.password, err)}
		}
		lines = append(lines, fmt.Sprintf("%s · effective %s", privilege, yesNo(allowed)))
	}
	return domain.PrivilegeSection{Title: "effective", Lines: lines}
}

func (c *connection) roleTableGrants(ctx context.Context, principal string) domain.PrivilegeSection {
	rows, err := c.conn.QueryContext(ctx, `
SELECT table_schema, table_name, privilege_type
FROM information_schema.role_table_grants
WHERE grantee = $1
ORDER BY table_schema, table_name, privilege_type`, principal)
	if err != nil {
		return domain.PrivilegeSection{Title: "table grants", Err: privilegeSectionError("privileges", c.password, err)}
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var schema, table, privilege string
		if err := rows.Scan(&schema, &table, &privilege); err != nil {
			return domain.PrivilegeSection{Title: "table grants", Err: privilegeSectionError("privileges", c.password, err)}
		}
		lines = append(lines, fmt.Sprintf("%s on %s.%s", privilege, schema, table))
	}
	return domain.PrivilegeSection{Title: "table grants", Lines: lines}
}

func privilegeSectionError(op, password string, err error) string {
	safe := safeError(op, password, err)
	if domain.KindOf(safe) == domain.KindPermission {
		return "not permitted. This account cannot read this section."
	}
	return safe.Error()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
