package postgres

import (
	"context"
	"fmt"
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

func (c *connection) DeleteRow(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error) {
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
	if err := validRowTable(spec.Ref); err != nil {
		return "", nil, err
	}
	schema := spec.Ref.Schema
	if schema == "" {
		schema = "public"
	}
	table := quoteIdentifier(schema) + "." + quoteIdentifier(spec.Ref.Name)

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
		return "INSERT INTO " + table + " DEFAULT VALUES", nil, nil
	}
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return "INSERT INTO " + table + " (" + strings.Join(columns, ", ") +
		") VALUES (" + strings.Join(placeholders, ", ") + ")", args, nil
}

func buildRowDelete(spec domain.RowDelete) (string, []any, error) {
	if err := validRowTable(spec.Ref); err != nil {
		return "", nil, err
	}
	if len(spec.Keys) == 0 {
		return "", nil, domain.NewError(domain.KindConfig, "row", "the row has no primary key")
	}
	schema := spec.Ref.Schema
	if schema == "" {
		schema = "public"
	}
	table := quoteIdentifier(schema) + "." + quoteIdentifier(spec.Ref.Name)
	var predicates []string
	var args []any
	for i, key := range spec.Keys {
		column := strings.TrimSpace(key.Column)
		if column == "" {
			return "", nil, domain.NewError(domain.KindConfig, "row", "a key column is empty")
		}
		predicates = append(predicates, fmt.Sprintf("%s = $%d", quoteIdentifier(column), i+1))
		args = append(args, rowArgument(key))
	}
	return "DELETE FROM " + table + " WHERE " + strings.Join(predicates, " AND "), args, nil
}

func validRowTable(ref domain.ObjectRef) error {
	if ref.Kind != domain.KindTable || strings.TrimSpace(ref.Name) == "" {
		return domain.NewError(domain.KindConfig, "row", "the object has no writable rows")
	}
	return nil
}

func rowArgument(value domain.RowValue) any {
	if value.Null {
		return nil
	}
	return value.Value
}
