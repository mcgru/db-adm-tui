package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.DataReader = (*connection)(nil)

// Page implements ports.DataReader with schema-qualified, quoted identifiers
// and parameterized offset paging. The primary key supplies the stable order.
func (c *connection) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	if query.Ref.Kind != domain.KindTable {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "this object has no data view")
	}
	if query.Offset < 0 || query.Limit <= 0 {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "the page window is not valid")
	}
	if query.Filter != nil && strings.TrimSpace(query.Filter.Field) == "" {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "the filter field is not valid")
	}

	// A missing or failed order key still allows browsing; the UI warns that
	// rows may repeat or skip.
	orderKey, orderErr := newCatalog(c).OrderKey(ctx, query.Ref)
	if orderErr != nil {
		orderKey = ""
	}

	statement := buildDataSelect(query.Ref, orderKey, query.Filter)
	args := []any{query.Limit + 1, query.Offset}
	if query.Filter != nil {
		args = append(args, query.Filter.Pattern)
	}
	rows, err := c.conn.QueryContext(ctx, statement, args...)
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

// buildDataSelect builds the one table window statement. Identifiers are
// quoted; the window values and an optional case-aware regexp filter remain
// bound parameters. Filters target the base column, so a JSON subfield expands
// to its parent column and the value is cast to text for native regexp matching.
func buildDataSelect(ref domain.ObjectRef, orderKey string, filter *domain.DataFilter) string {
	schema := ref.Schema
	if schema == "" {
		schema = "public"
	}
	var b strings.Builder
	b.WriteString("SELECT * FROM ")
	b.WriteString(quoteIdentifier(schema))
	b.WriteString(".")
	b.WriteString(quoteIdentifier(ref.Name))
	if filter != nil {
		op := "~"
		if filter.CaseInsensitive {
			op = "~*"
		}
		b.WriteString(" WHERE CAST(")
		b.WriteString(quoteIdentifier(filter.Field))
		b.WriteString(" AS text) ")
		b.WriteString(op)
		b.WriteString(" $3")
	}
	if columns := splitOrderKey(orderKey); len(columns) > 0 {
		b.WriteString(" ORDER BY ")
		for i, column := range columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteIdentifier(column))
		}
	}
	b.WriteString(" LIMIT $1 OFFSET $2")
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
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
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
			kinds[i] = postgresCellKind(types[i].DatabaseTypeName())
		}
	}
	for rows.Next() {
		if len(grid.Rows) >= limit {
			// The caller asked for one row beyond the page to learn HasMore.
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
	schema := ref.Schema
	if schema == "" {
		schema = "public"
	}
	var reltuples int64
	err := c.conn.QueryRowContext(ctx, `
SELECT reltuples::bigint
FROM pg_class
WHERE oid = format('%I.%I', $1, $2)::regclass`, schema, ref.Name).Scan(&reltuples)
	if err != nil || reltuples < 0 {
		return -1, ""
	}
	return reltuples, domain.EstimateCatalog
}

func postgresCellKind(databaseType string) domain.CellKind {
	switch strings.ToUpper(strings.TrimSpace(databaseType)) {
	case "INT2", "INT4", "INT8", "FLOAT4", "FLOAT8", "NUMERIC", "DECIMAL", "MONEY", "OID":
		return domain.CellNumber
	case "BOOL":
		return domain.CellBool
	case "JSON", "JSONB":
		return domain.CellJSON
	case "BYTEA":
		return domain.CellBinary
	case "DATE", "TIME", "TIMETZ", "TIMESTAMP", "TIMESTAMPTZ":
		return domain.CellDatetime
	default:
		return domain.CellText
	}
}
