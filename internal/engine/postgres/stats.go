package postgres

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.StatsReader = (*connection)(nil)

func (c *connection) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	if ref.Kind != domain.KindTable {
		return domain.Structure{}, domain.NewError(domain.KindConfig, "structure", "this object has no structure view")
	}
	schema := ref.Schema
	if schema == "" {
		schema = "public"
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT a.attname,
       format_type(a.atttypid, a.atttypmod),
       NOT a.attnotnull,
       COALESCE(pg_get_expr(d.adbin, d.adrelid), ''),
       a.attnum
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
WHERE n.nspname = $1 AND c.relname = $2
  AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum`, schema, ref.Name)
	if err != nil {
		return domain.Structure{}, safeError("structure", c.password, err)
	}
	defer rows.Close()
	structure := domain.Structure{Source: domain.StructureCatalog}
	for rows.Next() {
		var field domain.StructureField
		var position int
		if err := rows.Scan(&field.Name, &field.Type, &field.Nullable, &field.Default, &position); err != nil {
			return domain.Structure{}, safeError("structure", c.password, err)
		}
		structure.Fields = append(structure.Fields, field)
	}
	if err := rows.Err(); err != nil {
		return domain.Structure{}, safeError("structure", c.password, err)
	}
	return structure, nil
}

func (c *connection) Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error) {
	if ref.Kind != domain.KindTable {
		return nil, domain.NewError(domain.KindConfig, "indexes", "this object has no index view")
	}
	schema := ref.Schema
	if schema == "" {
		schema = "public"
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT i.relname,
       ix.indisunique,
       am.amname,
       pg_get_indexdef(ix.indexrelid),
       pg_relation_size(ix.indexrelid),
       COALESCE(string_agg(a.attname, ',' ORDER BY k.ord), '')
FROM pg_index ix
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_am am ON am.oid = i.relam
LEFT JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
WHERE n.nspname = $1 AND t.relname = $2
GROUP BY i.relname, ix.indisunique, am.amname, ix.indexrelid
ORDER BY i.relname`, schema, ref.Name)
	if err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	defer rows.Close()
	var out []domain.Index
	for rows.Next() {
		var index domain.Index
		var columns string
		if err := rows.Scan(&index.Name, &index.Unique, &index.Method, &index.Definition, &index.SizeBytes, &columns); err != nil {
			return nil, safeError("indexes", c.password, err)
		}
		index.Columns = splitOrderKey(columns)
		out = append(out, index)
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	return out, nil
}
