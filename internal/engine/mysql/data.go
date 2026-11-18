package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.DataReader = (*connection)(nil)

// Page implements ports.DataReader with backtick-quoted identifiers and
// parameterized offset paging. The primary key supplies the stable order.
func (c *connection) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	if query.Ref.Kind != domain.KindTable && query.Ref.Kind != domain.KindCollection {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "this object has no data view")
	}
	if query.Offset < 0 || query.Limit <= 0 {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "the page window is not valid")
	}

	orderKey, orderErr := c.OrderKey(ctx, query.Ref)
	if orderErr != nil {
		orderKey = ""
	}

	statement := buildDataSelect(query.Ref, orderKey)
	rows, err := c.conn.QueryContext(ctx, statement, query.Limit+1, query.Offset)
	if err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}
	defer rows.Close()

	grid, hasMore, err := scanDataRows(rows, query.Limit)
	if err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}
	if err := rows.Close(); err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}
	if err := rows.Err(); err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}

	total, source := c.estimate(ctx, query.Ref)
	return domain.DataPage{
		Grid:           grid,
		Offset:         query.Offset,
		HasMore:        hasMore,
		EstimatedTotal: total,
		EstimateSource: source,
		OrderKey:       orderKey,
		OrderKeyKnown:  orderErr == nil,
	}, nil
}

// buildDataSelect builds the one table window statement with backtick
// identifiers and bound LIMIT/OFFSET values.
func buildDataSelect(ref domain.ObjectRef, orderKey string) string {
	var b strings.Builder
	b.WriteString("SELECT * FROM ")
	b.WriteString(quoteIdentifier(ref.Database))
	b.WriteString(".")
	b.WriteString(quoteIdentifier(ref.Name))
	if columns := splitOrderKey(orderKey); len(columns) > 0 {
		b.WriteString(" ORDER BY ")
		for i, column := range columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteIdentifier(column))
		}
	}
	b.WriteString(" LIMIT ? OFFSET ?")
	return b.String()
}

func splitOrderKey(key string) []string {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	parts := strings.Split(key, ",")
	columns := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			columns = append(columns, part)
		}
	}
	return columns
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func scanDataRows(rows *sql.Rows, limit int) (domain.Grid, bool, error) {
	names, err := rows.Columns()
	if err != nil {
		return domain.Grid{}, false, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return domain.Grid{}, false, err
	}
	grid := domain.Grid{Columns: make([]domain.Column, len(names))}
	kinds := make([]domain.CellKind, len(names))
	for i, name := range names {
		grid.Columns[i] = domain.Column{Name: name}
		if i < len(types) {
			kinds[i] = mysqlCellKind(types[i].DatabaseTypeName())
		}
	}
	for rows.Next() {
		if len(grid.Rows) >= limit {
			return grid, true, nil
		}
		values := make([]any, len(names))
		targets := make([]any, len(names))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return domain.Grid{}, false, err
		}
		cells := make([]domain.Cell, len(names))
		for i := range values {
			cells[i] = domain.CellFromValue(values[i], kinds[i])
		}
		grid.Rows = append(grid.Rows, domain.Row{Cells: cells})
	}
	return grid, false, nil
}

func (c *connection) estimate(ctx context.Context, ref domain.ObjectRef) (int64, string) {
	var total sql.NullInt64
	err := c.conn.QueryRowContext(ctx, `
SELECT TABLE_ROWS
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, ref.Database, ref.Name).Scan(&total)
	if err != nil || !total.Valid || total.Int64 < 0 {
		return -1, ""
	}
	return total.Int64, domain.EstimateCatalog
}

func mysqlCellKind(databaseType string) domain.CellKind {
	switch strings.ToUpper(strings.TrimSpace(databaseType)) {
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT",
		"DECIMAL", "NUMERIC", "FLOAT", "DOUBLE", "REAL", "YEAR":
		return domain.CellNumber
	case "BOOL", "BOOLEAN":
		return domain.CellBool
	case "JSON":
		return domain.CellJSON
	case "BINARY", "VARBINARY", "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BIT":
		return domain.CellBinary
	case "DATE", "TIME", "DATETIME", "TIMESTAMP":
		return domain.CellDatetime
	default:
		return domain.CellText
	}
}
