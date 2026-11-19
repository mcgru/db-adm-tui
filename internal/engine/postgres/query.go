package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.QueryExecutor = (*connection)(nil)

type queryCursor struct {
	rows     *sql.Rows
	columns  []domain.Column
	kinds    []domain.CellKind
	pending  *domain.Row
	closed   bool
	password string
}

// Query opens one forward-only SQL cursor on the session connection.
func (c *connection) Query(ctx context.Context, request domain.QueryRequest) (ports.QueryCursor, error) {
	if strings.TrimSpace(request.Statement) == "" {
		return nil, domain.NewError(domain.KindConfig, "query", "the statement is empty")
	}
	rows, err := c.conn.QueryContext(ctx, request.Statement)
	if err != nil {
		return nil, safeError("query", c.password, err)
	}
	names, err := rows.Columns()
	if err != nil {
		_ = rows.Close()
		return nil, safeError("query", c.password, err)
	}
	names = domain.UniqueColumnNames(names)
	types, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, safeError("query", c.password, err)
	}
	kinds := make([]domain.CellKind, len(names))
	for i := range names {
		if i < len(types) {
			kinds[i] = postgresCellKind(types[i].DatabaseTypeName())
		}
	}
	columns := make([]domain.Column, len(names))
	for i, name := range names {
		columns[i] = domain.Column{Name: name}
	}
	return &queryCursor{rows: rows, columns: columns, kinds: kinds, password: c.password}, nil
}

func (q *queryCursor) Next(ctx context.Context, limit int) (domain.QueryPage, error) {
	if q == nil || q.rows == nil || q.closed {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result cursor is closed")
	}
	if limit <= 0 {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result window is not valid")
	}
	grid := domain.Grid{Columns: q.columns}
	for len(grid.Rows) < limit {
		row, ok, err := q.nextRow()
		if err != nil {
			return domain.QueryPage{}, safeError("query", q.password, err)
		}
		if !ok {
			return domain.QueryPage{Grid: grid}, nil
		}
		grid.Rows = append(grid.Rows, row)
	}
	row, ok, err := q.nextRow()
	if err != nil {
		return domain.QueryPage{}, safeError("query", q.password, err)
	}
	if ok {
		q.pending = &row
	}
	return domain.QueryPage{Grid: grid, HasMore: ok}, nil
}

func (q *queryCursor) nextRow() (domain.Row, bool, error) {
	if q.pending != nil {
		row := *q.pending
		q.pending = nil
		return row, true, nil
	}
	if !q.rows.Next() {
		return domain.Row{}, false, q.rows.Err()
	}
	values := make([]any, len(q.kinds))
	targets := make([]any, len(q.kinds))
	for i := range values {
		targets[i] = &values[i]
	}
	if err := q.rows.Scan(targets...); err != nil {
		return domain.Row{}, false, err
	}
	cells := make([]domain.Cell, len(values))
	for i := range values {
		cells[i] = domain.CellFromValue(values[i], q.kinds[i])
	}
	return domain.Row{Cells: cells}, true, nil
}

func (q *queryCursor) Close(ctx context.Context) error {
	if q == nil || q.closed {
		return nil
	}
	q.closed = true
	if q.rows == nil {
		return nil
	}
	return q.rows.Close()
}
