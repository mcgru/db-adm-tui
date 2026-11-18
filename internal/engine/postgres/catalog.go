package postgres

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.Catalog = (*catalog)(nil)

type catalog struct {
	conn *connection
}

// newCatalog builds the catalog over the live connection; Connect wires it.
func newCatalog(conn *connection) *catalog { return &catalog{conn: conn} }

func (c *catalog) Databases(ctx context.Context) ([]domain.Node, error) {
	rows, err := c.conn.conn.QueryContext(ctx, `
SELECT datname
FROM pg_database
WHERE datallowconn AND NOT datistemplate
  AND has_database_privilege(datname, 'CONNECT')
ORDER BY datname`)
	if err != nil {
		return nil, c.err("databases", err)
	}
	defer rows.Close()
	var out []domain.Node
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, c.err("databases", err)
		}
		out = append(out, domain.Node{
			Ref:        domain.ObjectRef{Kind: domain.KindDatabase, Database: name, Name: name},
			Name:       name,
			Kind:       domain.KindDatabase,
			Expandable: true,
		})
	}
	return out, c.err("databases", rows.Err())
}

func (c *catalog) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	switch ref.Kind {
	case domain.KindDatabase:
		return c.schemas(ctx, ref.Database)
	case domain.KindSchema:
		return c.objects(ctx, ref.Database, ref.Schema)
	default:
		return nil, domain.NewError(domain.KindConfig, "catalog", "this node has no children")
	}
}

func (c *catalog) schemas(ctx context.Context, database string) ([]domain.Node, error) {
	rows, err := c.conn.conn.QueryContext(ctx, `
SELECT nspname
FROM pg_namespace
WHERE nspname <> 'information_schema'
  AND nspname NOT LIKE 'pg\_%' ESCAPE '\'
  AND has_schema_privilege(oid, 'USAGE')
ORDER BY nspname`)
	if err != nil {
		return nil, c.err("schemas", err)
	}
	defer rows.Close()
	var out []domain.Node
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, c.err("schemas", err)
		}
		out = append(out, domain.Node{
			Ref:        domain.ObjectRef{Kind: domain.KindSchema, Database: database, Schema: name, Name: name},
			Name:       name,
			Kind:       domain.KindSchema,
			Expandable: true,
		})
	}
	return out, c.err("schemas", rows.Err())
}

func (c *catalog) objects(ctx context.Context, database, schema string) ([]domain.Node, error) {
	rows, err := c.conn.conn.QueryContext(ctx, `
SELECT c.relname, c.relkind::text,
  CASE WHEN c.relkind = 'S' THEN false
       ELSE has_table_privilege(c.oid, 'SELECT') END AS can_read,
  CASE WHEN c.relkind = 'S' THEN false
       ELSE (has_table_privilege(c.oid, 'INSERT')
             OR has_table_privilege(c.oid, 'UPDATE')
             OR has_table_privilege(c.oid, 'DELETE')
             OR has_table_privilege(c.oid, 'TRUNCATE')) END AS can_write
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = $1 AND c.relkind IN ('r', 'p', 'f', 'v', 'm', 'S')
ORDER BY c.relname`, schema)
	if err != nil {
		return nil, c.err("objects", err)
	}
	defer rows.Close()
	var out []domain.Node
	for rows.Next() {
		var name, relkind string
		var canRead, canWrite bool
		if err := rows.Scan(&name, &relkind, &canRead, &canWrite); err != nil {
			return nil, c.err("objects", err)
		}
		kind := postgresObjectKind(relkind)
		node := domain.Node{
			Ref:  domain.ObjectRef{Kind: kind, Database: database, Schema: schema, Name: name},
			Name: name,
			Kind: kind,
		}
		if kind != domain.KindSequence {
			node.Access = domain.AccessFrom(canRead, canWrite)
		}
		out = append(out, node)
	}
	return out, c.err("objects", rows.Err())
}

func (c *catalog) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	kind := ref.Kind
	rows, err := c.conn.conn.QueryContext(ctx, `
SELECT rolname, rolcanlogin, rolsuper, rolinherit, rolcreaterole, rolcreatedb,
       rolreplication, rolbypassrls
FROM pg_roles
WHERE rolname NOT LIKE 'pg\_%' ESCAPE '\'
ORDER BY rolname`)
	if err != nil {
		return nil, c.err("principals", err)
	}
	defer rows.Close()
	var out []domain.Principal
	for rows.Next() {
		var name string
		var login, super, inherit, createrole, createdb, replication, bypassRLS bool
		if err := rows.Scan(&name, &login, &super, &inherit, &createrole, &createdb, &replication, &bypassRLS); err != nil {
			return nil, c.err("principals", err)
		}
		if kind == domain.KindUser && !login {
			continue
		}
		if kind == domain.KindRole && login {
			continue
		}
		var attrs []string
		if login {
			attrs = append(attrs, "LOGIN")
		} else {
			attrs = append(attrs, "NOLOGIN")
		}
		if super {
			attrs = append(attrs, "SUPERUSER")
		}
		if createrole {
			attrs = append(attrs, "CREATEROLE")
		}
		if createdb {
			attrs = append(attrs, "CREATEDB")
		}
		if replication {
			attrs = append(attrs, "REPLICATION")
		}
		if bypassRLS {
			attrs = append(attrs, "BYPASSRLS")
		}
		if !inherit {
			attrs = append(attrs, "INHERIT FALSE")
		}
		pkind := domain.KindUser
		if !login {
			pkind = domain.KindRole
		}
		out = append(out, domain.Principal{Name: name, Kind: pkind, Scope: "cluster", Attributes: attrs})
	}
	return out, c.err("principals", rows.Err())
}

func (c *catalog) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	if ref.Kind != domain.KindTable && ref.Kind != domain.KindCollection {
		return "", nil
	}
	schema := ref.Schema
	if schema == "" {
		schema = "public"
	}
	rows, err := c.conn.conn.QueryContext(ctx, `
SELECT a.attname
FROM pg_index i
JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
WHERE i.indrelid = format('%I.%I', $1, $2)::regclass
  AND i.indisprimary
ORDER BY array_position(i.indkey, a.attnum)`, schema, ref.Name)
	if err != nil {
		return "", c.err("order-key", err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", c.err("order-key", err)
		}
		cols = append(cols, name)
	}
	return strings.Join(cols, ","), c.err("order-key", rows.Err())
}

func postgresObjectKind(relkind string) domain.ObjectKind {
	switch relkind {
	case "v", "m":
		return domain.KindView
	case "S":
		return domain.KindSequence
	default:
		return domain.KindTable
	}
}

func (c *catalog) err(op string, err error) error {
	if err == nil {
		return nil
	}
	return safeError(op, c.conn.password, err)
}
