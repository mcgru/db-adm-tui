package postgres

import (
	"context"
	"sort"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RoleMatrixReader = (*connection)(nil)

func (c *connection) RoleMatrix(ctx context.Context) (domain.RoleMatrix, error) {
	principals, byName, err := c.matrixPrincipals(ctx)
	if err != nil {
		return domain.RoleMatrix{}, err
	}
	matrix := domain.RoleMatrix{
		Principals: append([]domain.RoleRef(nil), principals...),
		Roles:      append([]domain.RoleRef(nil), principals...),
		Members:    map[string]map[string]bool{},
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT member.rolname, granted.rolname
FROM pg_auth_members m
JOIN pg_roles member ON member.oid = m.member
JOIN pg_roles granted ON granted.oid = m.roleid
ORDER BY member.rolname, granted.rolname`)
	if err != nil {
		return domain.RoleMatrix{}, safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var memberName, roleName string
		if err := rows.Scan(&memberName, &roleName); err != nil {
			return domain.RoleMatrix{}, safeError("role-matrix", c.password, err)
		}
		member, ok := byName[memberName]
		if !ok {
			continue
		}
		role, ok := byName[roleName]
		if !ok {
			continue
		}
		if matrix.Members[member.Key()] == nil {
			matrix.Members[member.Key()] = map[string]bool{}
		}
		matrix.Members[member.Key()][role.Key()] = true
	}
	if err := rows.Err(); err != nil {
		return matrix, safeError("role-matrix", c.password, err)
	}
	return matrix, nil
}

func (c *connection) PrivilegeMatrix(ctx context.Context, objects []domain.ObjectRef) (domain.PrivilegeMatrix, error) {
	principals, byName, err := c.matrixPrincipals(ctx)
	if err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	matrix := domain.PrivilegeMatrix{
		Principals: append([]domain.RoleRef(nil), principals...),
		Objects:    append([]domain.ObjectRef(nil), objects...),
		Cells:      map[string]map[string][]string{},
	}
	databases := map[string]domain.ObjectRef{}
	schemas := map[string]domain.ObjectRef{}
	tables := map[string]domain.ObjectRef{}
	for _, object := range objects {
		switch object.Kind {
		case domain.KindDatabase:
			name := object.Database
			if name == "" {
				name = object.Name
			}
			databases[name] = object
		case domain.KindSchema:
			name := object.Schema
			if name == "" {
				name = object.Name
			}
			schemas[name] = object
		case domain.KindTable, domain.KindView:
			tables[object.Schema+"\x00"+object.Name] = object
		}
	}
	record := func(grantee string, object domain.ObjectRef, privilege string) {
		principal, ok := byName[grantee]
		if !ok {
			return
		}
		if matrix.Cells[principal.Key()] == nil {
			matrix.Cells[principal.Key()] = map[string][]string{}
		}
		matrix.Cells[principal.Key()][object.Key()] = appendUnique(
			matrix.Cells[principal.Key()][object.Key()], privilege)
	}
	if err := c.matrixDatabaseGrants(ctx, databases, record); err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	if err := c.matrixSchemaGrants(ctx, schemas, record); err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	if err := c.matrixTableGrants(ctx, tables, record); err != nil {
		return domain.PrivilegeMatrix{}, err
	}
	for principal := range matrix.Cells {
		for object := range matrix.Cells[principal] {
			sort.Strings(matrix.Cells[principal][object])
		}
	}
	return matrix, nil
}

func (c *connection) matrixPrincipals(ctx context.Context) ([]domain.RoleRef, map[string]domain.RoleRef, error) {
	rows, err := c.conn.QueryContext(ctx, `
SELECT rolname, rolcanlogin
FROM pg_roles
WHERE rolname NOT LIKE 'pg\_%' ESCAPE '\'
ORDER BY rolname`)
	if err != nil {
		return nil, nil, safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	var principals []domain.RoleRef
	byName := map[string]domain.RoleRef{}
	for rows.Next() {
		var name string
		var login bool
		if err := rows.Scan(&name, &login); err != nil {
			return nil, nil, safeError("role-matrix", c.password, err)
		}
		kind := domain.KindRole
		if login {
			kind = domain.KindUser
		}
		ref := domain.RoleRef{
			Engine: domain.EnginePostgreSQL,
			Name:   name,
			Kind:   kind,
		}
		principals = append(principals, ref)
		byName[name] = ref
	}
	if err := rows.Err(); err != nil {
		return nil, nil, safeError("role-matrix", c.password, err)
	}
	return principals, byName, nil
}

func (c *connection) matrixDatabaseGrants(ctx context.Context, objects map[string]domain.ObjectRef,
	record func(string, domain.ObjectRef, string)) error {
	if len(objects) == 0 {
		return nil
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT COALESCE(grantee.rolname, 'PUBLIC'), d.datname, acl.privilege_type
FROM pg_database d
CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl, acldefault('d', d.datdba))) acl
LEFT JOIN pg_roles grantee ON grantee.oid = acl.grantee
WHERE d.datname = current_database()`)
	if err != nil {
		return safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var grantee, database, privilege string
		if err := rows.Scan(&grantee, &database, &privilege); err != nil {
			return safeError("role-matrix", c.password, err)
		}
		object, ok := objects[database]
		if !ok {
			continue
		}
		record(grantee, object, privilege)
	}
	if err := rows.Err(); err != nil {
		return safeError("role-matrix", c.password, err)
	}
	return nil
}

func (c *connection) matrixSchemaGrants(ctx context.Context, objects map[string]domain.ObjectRef,
	record func(string, domain.ObjectRef, string)) error {
	if len(objects) == 0 {
		return nil
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT COALESCE(grantee.rolname, 'PUBLIC'), n.nspname, acl.privilege_type
FROM pg_namespace n
CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl, acldefault('n', n.nspowner))) acl
LEFT JOIN pg_roles grantee ON grantee.oid = acl.grantee
WHERE n.nspname NOT LIKE 'pg\_%' ESCAPE '\'
  AND n.nspname <> 'information_schema'
ORDER BY n.nspname`)
	if err != nil {
		return safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var grantee, schema, privilege string
		if err := rows.Scan(&grantee, &schema, &privilege); err != nil {
			return safeError("role-matrix", c.password, err)
		}
		object, ok := objects[schema]
		if !ok {
			continue
		}
		record(grantee, object, privilege)
	}
	if err := rows.Err(); err != nil {
		return safeError("role-matrix", c.password, err)
	}
	return nil
}

func (c *connection) matrixTableGrants(ctx context.Context, objects map[string]domain.ObjectRef,
	record func(string, domain.ObjectRef, string)) error {
	if len(objects) == 0 {
		return nil
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT COALESCE(grantee.rolname, 'PUBLIC'), n.nspname, c.relname, acl.privilege_type
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl, acldefault('r', c.relowner))) acl
LEFT JOIN pg_roles grantee ON grantee.oid = acl.grantee
WHERE c.relkind IN ('r', 'p', 'v', 'm', 'f')
  AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\'
  AND n.nspname <> 'information_schema'
ORDER BY n.nspname, c.relname`)
	if err != nil {
		return safeError("role-matrix", c.password, err)
	}
	defer rows.Close()
	for rows.Next() {
		var grantee, schema, table, privilege string
		if err := rows.Scan(&grantee, &schema, &table, &privilege); err != nil {
			return safeError("role-matrix", c.password, err)
		}
		object, ok := objects[schema+"\x00"+table]
		if !ok {
			continue
		}
		record(grantee, object, privilege)
	}
	if err := rows.Err(); err != nil {
		return safeError("role-matrix", c.password, err)
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
