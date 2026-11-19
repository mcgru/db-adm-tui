package mysql

import (
	"context"
	"sort"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleMatrixReader = (*connection)(nil)

func (c *connection) RoleMatrix(ctx context.Context) (domain.RoleMatrix, error) {
	principals, byName, roleNames, mariadb, err := c.matrixPrincipals(ctx)
	if err != nil {
		return domain.RoleMatrix{}, err
	}
	matrix := domain.RoleMatrix{
		Principals: append([]domain.RoleRef(nil), principals...),
		Members:    map[string]map[string]bool{},
	}
	for _, principal := range principals {
		if principal.Kind == domain.KindRole {
			matrix.Roles = append(matrix.Roles, principal)
		}
	}
	statement := `
SELECT FROM_USER, FROM_HOST, TO_USER, TO_HOST
FROM mysql.role_edges
ORDER BY TO_USER, TO_HOST, FROM_USER, FROM_HOST`
	if mariadb {
		statement = `SELECT Role, '', User, Host FROM mysql.roles_mapping`
	}
	rows, err := c.conn.QueryContext(ctx, statement)
	if err != nil {
		return domain.RoleMatrix{}, safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var roleUser, roleHost, memberUser, memberHost string
		if err := rows.Scan(&roleUser, &roleHost, &memberUser, &memberHost); err != nil {
			return domain.RoleMatrix{}, safeError("role-matrix", c.password, err)
		}
		roleName := roleUser
		if roleHost != "" {
			roleName = roleUser + "@" + roleHost
		}
		if mariadb {
			if full, ok := roleNames[roleUser]; ok {
				roleName = full
			}
		}
		member := memberUser + "@" + memberHost
		role, ok := byName[roleName]
		if !ok {
			continue
		}
		memberRef, ok := byName[member]
		if !ok {
			continue
		}
		if matrix.Members[memberRef.Key()] == nil {
			matrix.Members[memberRef.Key()] = map[string]bool{}
		}
		matrix.Members[memberRef.Key()][role.Key()] = true
	}
	if err := rows.Err(); err != nil {
		return matrix, safeError("role-matrix", c.password, err)
	}
	return matrix, nil
}

func (c *connection) PrivilegeMatrix(ctx context.Context, objects []domain.ObjectRef) (domain.PrivilegeMatrix, error) {
	principals, byName, _, _, err := c.matrixPrincipals(ctx)
	if err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	matrix := domain.PrivilegeMatrix{
		Principals: append([]domain.RoleRef(nil), principals...),
		Objects:    append([]domain.ObjectRef(nil), objects...),
		Cells:      map[string]map[string][]string{},
	}
	databases := map[string]domain.ObjectRef{}
	tables := map[string]domain.ObjectRef{}
	for _, object := range objects {
		switch object.Kind {
		case domain.KindDatabase:
			name := object.Database
			if name == "" {
				name = object.Name
			}
			databases[name] = object
		case domain.KindTable, domain.KindView:
			tables[object.Database+"\x00"+object.Name] = object
		}
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT GRANTEE, TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE
FROM information_schema.TABLE_PRIVILEGES
UNION ALL
SELECT GRANTEE, TABLE_SCHEMA, '', PRIVILEGE_TYPE
FROM information_schema.SCHEMA_PRIVILEGES`)
	if err != nil {
		return domain.PrivilegeMatrix{}, safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var grantee, database, table, privilege string
		if err := rows.Scan(&grantee, &database, &table, &privilege); err != nil {
			return domain.PrivilegeMatrix{}, safeError("role-matrix", c.password, err)
		}
		principal, ok := byName[normalizeMySQLGrantee(grantee)]
		if !ok {
			continue
		}
		var object domain.ObjectRef
		if table == "" {
			object, ok = databases[database]
		} else {
			object, ok = tables[database+"\x00"+table]
		}
		if !ok {
			continue
		}
		if matrix.Cells[principal.Key()] == nil {
			matrix.Cells[principal.Key()] = map[string][]string{}
		}
		matrix.Cells[principal.Key()][object.Key()] = append(
			matrix.Cells[principal.Key()][object.Key()], privilege)
	}
	if err := rows.Err(); err != nil {
		return domain.PrivilegeMatrix{}, safeError("role-matrix", c.password, err)
	}
	for principal := range matrix.Cells {
		for object := range matrix.Cells[principal] {
			sort.Strings(matrix.Cells[principal][object])
		}
	}
	return matrix, nil
}

func (c *connection) matrixPrincipals(ctx context.Context) ([]domain.RoleRef, map[string]domain.RoleRef, map[string]string, bool, error) {
	mariadb, err := c.isMariaDB(ctx)
	if err != nil {
		return nil, nil, nil, false, err
	}
	roles := map[string]bool{}
	if mariadb {
		rows, err := c.conn.QueryContext(ctx, `
SELECT User, Host, is_role
FROM mysql.user
ORDER BY User, Host`)
		if err != nil {
			return nil, nil, nil, false, safeError("role-matrix", c.password, err)
		}
		for rows.Next() {
			var user, host, isRole string
			if err := rows.Scan(&user, &host, &isRole); err != nil {
				rows.Close()
				return nil, nil, nil, false, safeError("role-matrix", c.password, err)
			}
			if strings.EqualFold(isRole, "Y") {
				roles[user+"@"+host] = true
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, nil, nil, false, safeError("role-matrix", c.password, err)
		}
		rows.Close()
	} else {
		roleAccounts, err := c.mysqlRoleAccounts(ctx)
		if err != nil {
			return nil, nil, nil, false, err
		}
		roles = roleAccounts
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT User, Host
FROM mysql.user
ORDER BY User, Host`)
	if err != nil {
		return nil, nil, nil, false, safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	var principals []domain.RoleRef
	byName := map[string]domain.RoleRef{}
	roleNames := map[string]string{}
	for rows.Next() {
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			return nil, nil, nil, false, safeError("role-matrix", c.password, err)
		}
		name := user + "@" + host
		kind := domain.KindUser
		if roles[name] {
			kind = domain.KindRole
		}
		ref := domain.RoleRef{Engine: domain.EngineMySQL, Name: name, Kind: kind}
		principals = append(principals, ref)
		byName[name] = ref
		if kind == domain.KindRole {
			base := user
			if _, ok := roleNames[base]; !ok {
				roleNames[base] = name
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, safeError("role-matrix", c.password, err)
	}
	sort.Slice(principals, func(i, j int) bool { return principals[i].Name < principals[j].Name })
	return principals, byName, roleNames, mariadb, nil
}

func normalizeMySQLGrantee(value string) string {
	value = strings.ReplaceAll(value, "`", "")
	value = strings.ReplaceAll(value, "'", "")
	return value
}
