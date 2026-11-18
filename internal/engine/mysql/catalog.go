package mysql

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.Catalog = (*connection)(nil)

const databasesQuery = `
SELECT SCHEMA_NAME
FROM information_schema.SCHEMATA
ORDER BY SCHEMA_NAME`

const tablesQuery = `
SELECT TABLE_NAME, TABLE_TYPE
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = ?
ORDER BY TABLE_NAME`

const currentUserQuery = `SELECT CURRENT_USER()`

const enabledRolesQuery = `
SELECT ROLE_NAME, ROLE_HOST
FROM information_schema.ENABLED_ROLES`

const currentRoleQuery = `SELECT CURRENT_ROLE()`

const globalPrivilegesQuery = `
SELECT PRIVILEGE_TYPE
FROM information_schema.USER_PRIVILEGES
WHERE GRANTEE = ?`

const schemaPrivilegesQuery = `
SELECT TABLE_SCHEMA, PRIVILEGE_TYPE
FROM information_schema.SCHEMA_PRIVILEGES
WHERE GRANTEE = ?`

const tablePrivilegesQuery = `
SELECT TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE
FROM information_schema.TABLE_PRIVILEGES
WHERE GRANTEE = ?`

const primaryKeyQuery = `
SELECT COLUMN_NAME
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = ?
  AND TABLE_NAME = ?
  AND INDEX_NAME = 'PRIMARY'
ORDER BY SEQ_IN_INDEX`

// Databases implements ports.Catalog. MySQL and MariaDB expose every database
// on the one live connection.
func (c *connection) Databases(ctx context.Context) ([]domain.Node, error) {
	rows, err := c.conn.QueryContext(ctx, databasesQuery)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	defer rows.Close()
	var nodes []domain.Node
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, safeError("catalog", c.password, err)
		}
		nodes = append(nodes, databaseNode(name))
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	return nodes, nil
}

// Children implements ports.Catalog. MySQL has no schema layer: tables,
// views, and MariaDB sequences hang directly under the database.
func (c *connection) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	if ref.Kind != domain.KindDatabase {
		return nil, domain.NewError(domain.KindConfig, "catalog", "the node has no catalog children")
	}
	rows, err := c.conn.QueryContext(ctx, tablesQuery, ref.Database)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	type table struct {
		name string
		kind domain.ObjectKind
	}
	var tables []table
	for rows.Next() {
		var name, tableType string
		if err := rows.Scan(&name, &tableType); err != nil {
			rows.Close()
			return nil, safeError("catalog", c.password, err)
		}
		kind, ok := mysqlObjectKind(tableType)
		if !ok {
			continue
		}
		tables = append(tables, table{name: name, kind: kind})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}

	// Access markers are auxiliary. A failed discovery omits every marker for
	// this database instead of inventing a "no access" state.
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		names = append(names, table.name)
	}
	access := c.tableAccess(ctx, ref.Database, names)

	nodes := make([]domain.Node, 0, len(tables))
	for _, table := range tables {
		node := domain.Node{
			Ref: domain.ObjectRef{
				Engine:   domain.EngineMySQL,
				Database: ref.Database,
				Name:     table.name,
				Kind:     table.kind,
			},
			Name:   table.name,
			Kind:   table.kind,
			Access: domain.AccessUnknown,
		}
		if table.kind.AccessMarker() {
			if state, ok := access[table.name]; ok {
				node.Access = state
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Principals implements ports.Catalog. Native names are user@host; MariaDB
// exposes roles explicitly, MySQL 8 exposes them through role_edges.
func (c *connection) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	if ref.Kind != domain.KindUsers && ref.Kind != domain.KindRoles {
		return nil, domain.NewError(domain.KindConfig, "catalog", "the node has no principals")
	}
	mariadb, err := c.isMariaDB(ctx)
	if err != nil {
		return nil, err
	}
	if mariadb {
		return c.mariaDBPrincipals(ctx, ref.Kind)
	}
	return c.mysqlPrincipals(ctx, ref.Kind)
}

func (c *connection) mariaDBPrincipals(ctx context.Context, kind domain.ObjectKind) ([]domain.Principal, error) {
	rows, err := c.conn.QueryContext(ctx, `
SELECT User, Host, is_role, plugin
FROM mysql.user
ORDER BY User, Host`)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	defer rows.Close()
	var principals []domain.Principal
	for rows.Next() {
		var user, host, isRole, plugin string
		if err := rows.Scan(&user, &host, &isRole, &plugin); err != nil {
			return nil, safeError("catalog", c.password, err)
		}
		role := strings.EqualFold(isRole, "Y")
		if kind == domain.KindRoles && !role {
			continue
		}
		if kind == domain.KindUsers && role {
			continue
		}
		principalKind := domain.KindUser
		if role {
			principalKind = domain.KindRole
		}
		principals = append(principals, domain.Principal{
			Name:       user + "@" + host,
			Kind:       principalKind,
			Scope:      host,
			Attributes: mysqlAttributes(plugin, ""),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	return principals, nil
}

func (c *connection) mysqlPrincipals(ctx context.Context, kind domain.ObjectKind) ([]domain.Principal, error) {
	roles, roleErr := c.mysqlRoleAccounts(ctx)
	if roleErr != nil {
		return nil, roleErr
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT User, Host, account_locked, plugin
FROM mysql.user
ORDER BY User, Host`)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	defer rows.Close()
	var principals []domain.Principal
	for rows.Next() {
		var user, host, locked, plugin string
		if err := rows.Scan(&user, &host, &locked, &plugin); err != nil {
			return nil, safeError("catalog", c.password, err)
		}
		isRole := roles[user+"@"+host]
		if kind == domain.KindRoles && !isRole {
			continue
		}
		if kind == domain.KindUsers && isRole {
			continue
		}
		principalKind := domain.KindUser
		if isRole {
			principalKind = domain.KindRole
		}
		principals = append(principals, domain.Principal{
			Name:       user + "@" + host,
			Kind:       principalKind,
			Scope:      host,
			Attributes: mysqlAttributes(plugin, locked),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	return principals, nil
}

func (c *connection) mysqlRoleAccounts(ctx context.Context) (map[string]bool, error) {
	rows, err := c.conn.QueryContext(ctx, `
SELECT DISTINCT FROM_USER, FROM_HOST
FROM mysql.role_edges`)
	if err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	defer rows.Close()
	roles := map[string]bool{}
	for rows.Next() {
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			return nil, safeError("catalog", c.password, err)
		}
		roles[user+"@"+host] = true
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("catalog", c.password, err)
	}
	return roles, nil
}

// OrderKey implements ports.Catalog with the PRIMARY index columns.
func (c *connection) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	if ref.Kind != domain.KindTable && ref.Kind != domain.KindView {
		return "", nil
	}
	rows, err := c.conn.QueryContext(ctx, primaryKeyQuery, ref.Database, ref.Name)
	if err != nil {
		return "", safeError("catalog", c.password, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return "", safeError("catalog", c.password, err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return "", safeError("catalog", c.password, err)
	}
	return strings.Join(columns, ", "), nil
}

func (c *connection) isMariaDB(ctx context.Context) (bool, error) {
	var version string
	if err := c.conn.QueryRowContext(ctx, `SELECT VERSION()`).Scan(&version); err != nil {
		return false, safeError("catalog", c.password, err)
	}
	return strings.Contains(strings.ToLower(version), "mariadb"), nil
}

// tableAccess maps each visible table in a schema to its marker. Any privilege
// discovery failure returns nil so every marker is omitted.
func (c *connection) tableAccess(ctx context.Context, schema string, names []string) map[string]domain.Access {
	grantees, ok := c.grantees(ctx)
	if !ok {
		return nil
	}
	global := map[string]bool{}
	schemas := map[string]map[string]bool{}
	tables := map[string]map[string]bool{}
	for _, grantee := range grantees {
		priv, err := c.privileges(ctx, grantee)
		if err != nil {
			return nil
		}
		mergePrivileges(global, priv.global)
		for scope, set := range priv.schemas {
			if schemas[scope] == nil {
				schemas[scope] = map[string]bool{}
			}
			mergePrivileges(schemas[scope], set)
		}
		for table, set := range priv.tables {
			if tables[table] == nil {
				tables[table] = map[string]bool{}
			}
			mergePrivileges(tables[table], set)
		}
	}

	result := map[string]domain.Access{}
	for _, name := range names {
		key := schema + "\x00" + name
		read := global["SELECT"] || schemas[schema]["SELECT"] || tables[key]["SELECT"]
		write := anyWrite(global) || anyWrite(schemas[schema]) || anyWrite(tables[key])
		result[name] = domain.AccessFrom(read, write)
	}
	return result
}

type mysqlPrivilegeSet struct {
	global  map[string]bool
	schemas map[string]map[string]bool
	tables  map[string]map[string]bool
}

func (c *connection) privileges(ctx context.Context, grantee string) (mysqlPrivilegeSet, error) {
	result := mysqlPrivilegeSet{
		global:  map[string]bool{},
		schemas: map[string]map[string]bool{},
		tables:  map[string]map[string]bool{},
	}

	rows, err := c.conn.QueryContext(ctx, globalPrivilegesQuery, grantee)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var privilege string
		if err := rows.Scan(&privilege); err != nil {
			rows.Close()
			return result, err
		}
		result.global[strings.ToUpper(privilege)] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}

	rows, err = c.conn.QueryContext(ctx, schemaPrivilegesQuery, grantee)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var schema, privilege string
		if err := rows.Scan(&schema, &privilege); err != nil {
			rows.Close()
			return result, err
		}
		if result.schemas[schema] == nil {
			result.schemas[schema] = map[string]bool{}
		}
		result.schemas[schema][strings.ToUpper(privilege)] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}

	rows, err = c.conn.QueryContext(ctx, tablePrivilegesQuery, grantee)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var schema, table, privilege string
		if err := rows.Scan(&schema, &table, &privilege); err != nil {
			rows.Close()
			return result, err
		}
		key := schema + "\x00" + table
		if result.tables[key] == nil {
			result.tables[key] = map[string]bool{}
		}
		result.tables[key][strings.ToUpper(privilege)] = true
	}
	err = rows.Err()
	rows.Close()
	return result, err
}

// grantees returns the SQL grantees whose privileges apply: the current
// account plus any roles enabled for the session.
func (c *connection) grantees(ctx context.Context) ([]string, bool) {
	var current string
	if err := c.conn.QueryRowContext(ctx, currentUserQuery).Scan(&current); err != nil {
		return nil, false
	}
	user, host, found := strings.Cut(current, "@")
	if !found {
		return nil, false
	}
	grantees := []string{quoteGrantee(user, host)}

	rows, err := c.conn.QueryContext(ctx, enabledRolesQuery)
	if err == nil {
		for rows.Next() {
			var role, roleHost string
			if err := rows.Scan(&role, &roleHost); err != nil {
				rows.Close()
				return nil, false
			}
			grantees = append(grantees, quoteGrantee(role, roleHost))
		}
		if rows.Err() != nil {
			rows.Close()
			return nil, false
		}
		rows.Close()
		return grantees, true
	}

	// Older MariaDB has no ENABLED_ROLES view; CURRENT_ROLE() lists the active
	// roles in role@host form.
	var active string
	if err := c.conn.QueryRowContext(ctx, currentRoleQuery).Scan(&active); err != nil {
		return nil, false
	}
	active = strings.TrimSpace(active)
	if active == "" || strings.EqualFold(active, "NONE") {
		return grantees, true
	}
	for _, entry := range strings.Split(active, ",") {
		entry = strings.TrimSpace(strings.Trim(entry, "'\""))
		role, roleHost, found := strings.Cut(entry, "@")
		if !found || role == "" {
			return nil, false
		}
		grantees = append(grantees, quoteGrantee(role, roleHost))
	}
	return grantees, true
}

func quoteGrantee(user, host string) string {
	return "'" + strings.ReplaceAll(user, "'", "''") + "'@'" + strings.ReplaceAll(host, "'", "''") + "'"
}

func mergePrivileges(dst, src map[string]bool) {
	for privilege := range src {
		dst[privilege] = true
	}
}

func anyWrite(privileges map[string]bool) bool {
	// The marker means row-level write capability; DDL-only grants do not
	// allow writing rows. Column-level grants are not inspected in v1.
	for _, privilege := range []string{"INSERT", "UPDATE", "DELETE"} {
		if privileges[privilege] {
			return true
		}
	}
	return false
}

func mysqlAttributes(plugin, locked string) []string {
	var attrs []string
	if plugin != "" {
		attrs = append(attrs, strings.ToUpper(plugin))
	}
	if strings.EqualFold(locked, "Y") {
		attrs = append(attrs, "LOCKED")
	}
	return attrs
}

func databaseNode(name string) domain.Node {
	return domain.Node{
		Ref: domain.ObjectRef{
			Engine:   domain.EngineMySQL,
			Database: name,
			Name:     name,
			Kind:     domain.KindDatabase,
		},
		Name:       name,
		Kind:       domain.KindDatabase,
		Expandable: true,
	}
}

func mysqlObjectKind(tableType string) (domain.ObjectKind, bool) {
	switch strings.ToUpper(tableType) {
	case "BASE TABLE":
		return domain.KindTable, true
	case "VIEW", "SYSTEM VIEW":
		return domain.KindView, true
	case "SEQUENCE":
		return domain.KindSequence, true
	default:
		return "", false
	}
}
