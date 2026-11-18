package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.StatsReader = (*connection)(nil)

func (c *connection) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	if ref.Kind != domain.KindTable {
		return domain.Structure{}, domain.NewError(domain.KindConfig, "structure", "this object has no structure view")
	}
	rows, err := c.conn.QueryContext(ctx, `
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COALESCE(COLUMN_DEFAULT, ''), ORDINAL_POSITION
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION`, ref.Database, ref.Name)
	if err != nil {
		return domain.Structure{}, safeError("structure", c.password, err)
	}
	defer rows.Close()
	structure := domain.Structure{Source: domain.StructureCatalog}
	for rows.Next() {
		var field domain.StructureField
		var nullable string
		var defaultValue sql.NullString
		var position int
		if err := rows.Scan(&field.Name, &field.Type, &nullable, &defaultValue, &position); err != nil {
			return domain.Structure{}, safeError("structure", c.password, err)
		}
		field.Nullable = strings.EqualFold(nullable, "YES")
		if defaultValue.Valid {
			field.Default = defaultValue.String
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
	rows, err := c.conn.QueryContext(ctx, `
SELECT INDEX_NAME, NON_UNIQUE, INDEX_TYPE, COLUMN_NAME, CARDINALITY
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX`, ref.Database, ref.Name)
	if err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	defer rows.Close()
	var out []domain.Index
	cardinality := map[string]int64{}
	for rows.Next() {
		var name, indexType string
		var nonUnique int
		var column sql.NullString
		var card sql.NullInt64
		if err := rows.Scan(&name, &nonUnique, &indexType, &column, &card); err != nil {
			return nil, safeError("indexes", c.password, err)
		}
		if len(out) == 0 || out[len(out)-1].Name != name {
			out = append(out, domain.Index{
				Name:      name,
				Unique:    nonUnique == 0,
				Method:    indexType,
				SizeBytes: -1,
			})
		}
		index := &out[len(out)-1]
		if column.Valid {
			index.Columns = append(index.Columns, column.String)
		}
		if card.Valid && card.Int64 > cardinality[name] {
			cardinality[name] = card.Int64
		}
	}
	if err := rows.Err(); err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	for i := range out {
		if value, ok := cardinality[out[i].Name]; ok && value > 0 {
			out[i].Stats = fmt.Sprintf("cardinality %d", value)
		} else {
			out[i].Stats = "unavailable"
		}
	}
	return out, nil
}
