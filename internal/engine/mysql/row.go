package mysql

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RowAdmin = (*connection)(nil)

func (c *connection) InsertRow(ctx context.Context, spec domain.RowInsert) (domain.MutationResult, error) {
	statement, args, err := buildRowInsert(spec)
	if err != nil {
		return domain.MutationResult{}, err
	}
	result, err := c.conn.ExecContext(ctx, statement, args...)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func (c *connection) UpdateRows(ctx context.Context, spec domain.RowUpdate) (domain.MutationResult, error) {
	statement, args, err := buildRowUpdate(spec)
	if err != nil {
		return domain.MutationResult{}, err
	}
	result, err := c.conn.ExecContext(ctx, statement, args...)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func (c *connection) DeleteRows(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error) {
	statement, args, err := buildRowDelete(spec)
	if err != nil {
		return domain.MutationResult{}, err
	}
	result, err := c.conn.ExecContext(ctx, statement, args...)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	affected, _ := result.RowsAffected()
	return domain.MutationResult{Statement: statement, Affected: affected}, nil
}

func buildRowInsert(spec domain.RowInsert) (string, []any, error) {
	table, err := mysqlRowTable(spec.Ref)
	if err != nil {
		return "", nil, err
	}
	var columns []string
	var args []any
	seen := map[string]bool{}
	for _, value := range spec.Values {
		column := strings.TrimSpace(value.Column)
		if column == "" {
			return "", nil, domain.NewError(domain.KindConfig, "row", "a column name is empty")
		}
		if seen[column] {
			return "", nil, domain.NewError(domain.KindConfig, "row", "duplicate column "+column)
		}
		seen[column] = true
		if value.Default {
			continue
		}
		columns = append(columns, quoteIdentifier(column))
		args = append(args, rowArgument(value))
	}
	if len(columns) == 0 {
		return "INSERT INTO " + table + " () VALUES ()", nil, nil
	}
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return "INSERT INTO " + table + " (" + strings.Join(columns, ", ") +
		") VALUES (" + strings.Join(placeholders, ", ") + ")", args, nil
}

func buildRowUpdate(spec domain.RowUpdate) (string, []any, error) {
	table, err := mysqlRowTable(spec.Ref)
	if err != nil {
		return "", nil, err
	}
	column := strings.TrimSpace(spec.Column)
	if column == "" {
		return "", nil, domain.NewError(domain.KindConfig, "row", "the update column is empty")
	}
	set := quoteIdentifier(column) + " = "
	var args []any
	if spec.Value.Default {
		set += "DEFAULT"
	} else {
		set += "?"
		args = append(args, rowArgument(spec.Value))
	}
	predicate, keyArgs, err := buildRowPredicate(spec.Rows)
	if err != nil {
		return "", nil, err
	}
	args = append(args, keyArgs...)
	return "UPDATE " + table + " SET " + set + " WHERE " + predicate, args, nil
}

func buildRowDelete(spec domain.RowDelete) (string, []any, error) {
	table, err := mysqlRowTable(spec.Ref)
	if err != nil {
		return "", nil, err
	}
	predicate, args, err := buildRowPredicate(spec.Rows)
	if err != nil {
		return "", nil, err
	}
	return "DELETE FROM " + table + " WHERE " + predicate, args, nil
}

func buildRowPredicate(rows [][]domain.RowValue) (string, []any, error) {
	if len(rows) == 0 {
		return "", nil, domain.NewError(domain.KindConfig, "row", "the row has no primary key")
	}
	first := rows[0]
	if len(first) == 0 {
		return "", nil, domain.NewError(domain.KindConfig, "row", "the row has no primary key")
	}
	names := make([]string, len(first))
	for i, key := range first {
		name := strings.TrimSpace(key.Column)
		if name == "" {
			return "", nil, domain.NewError(domain.KindConfig, "row", "a key column is empty")
		}
		names[i] = name
	}

	var args []any
	var groups []string
	for _, row := range rows {
		if len(row) != len(names) {
			return "", nil, domain.NewError(domain.KindConfig, "row", "key columns differ between selected rows")
		}
		var parts []string
		for i, key := range row {
			if strings.TrimSpace(key.Column) != names[i] {
				return "", nil, domain.NewError(domain.KindConfig, "row", "key columns differ between selected rows")
			}
			if key.Null {
				return "", nil, domain.NewError(domain.KindConfig, "row", "a key value is empty")
			}
			parts = append(parts, quoteIdentifier(names[i])+" = ?")
			args = append(args, rowArgument(key))
		}
		groups = append(groups, "("+strings.Join(parts, " AND ")+")")
	}
	return strings.Join(groups, " OR "), args, nil
}

func mysqlRowTable(ref domain.ObjectRef) (string, error) {
	if (ref.Kind != domain.KindTable && ref.Kind != domain.KindCollection) ||
		strings.TrimSpace(ref.Database) == "" || strings.TrimSpace(ref.Name) == "" {
		return "", domain.NewError(domain.KindConfig, "row", "the object has no writable rows")
	}
	return quoteIdentifier(ref.Database) + "." + quoteIdentifier(ref.Name), nil
}

func rowArgument(value domain.RowValue) any {
	if value.Null {
		return nil
	}
	return value.Value
}
