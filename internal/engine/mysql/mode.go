package mysql

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.ModeController = (*connection)(nil)

// SetReadOnly sets MySQL/MariaDB's session transaction mode.
func (c *connection) SetReadOnly(ctx context.Context, readOnly bool) error {
	statement := "SET SESSION TRANSACTION READ WRITE"
	if readOnly {
		statement = "SET SESSION TRANSACTION READ ONLY"
	}
	if _, err := c.conn.ExecContext(ctx, statement); err != nil {
		return safeError("mode", c.password, err)
	}
	c.readOnly.Store(readOnly)
	return nil
}

// SessionReadOnly reports the last session-level setting applied successfully.
func (c *connection) SessionReadOnly() bool {
	return c.readOnly.Load()
}

// Elevated reports root or broad server/admin grants for the current account.
func (c *connection) Elevated(ctx context.Context) (bool, error) {
	var principal string
	if err := c.conn.QueryRowContext(ctx, "SELECT CURRENT_USER()").Scan(&principal); err != nil {
		return false, safeError("mode", c.password, err)
	}
	user, _ := splitMySQLPrincipal(principal)
	if strings.EqualFold(user, "root") {
		return true, nil
	}
	rows, err := c.conn.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false, safeError("mode", c.password, err)
	}
	defer rows.Close()
	var grants []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return false, safeError("mode", c.password, err)
		}
		grants = append(grants, line)
	}
	if err := rows.Err(); err != nil {
		return false, safeError("mode", c.password, err)
	}
	return mysqlElevated(grants), nil
}

func mysqlElevated(grants []string) bool {
	for _, line := range grants {
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "ALL PRIVILEGES ON *.*") ||
			strings.Contains(upper, "WITH GRANT OPTION") {
			return true
		}
		for _, token := range []string{
			" SUPER", " SYSTEM_VARIABLES_ADMIN", " SYSTEM_USER",
			" CREATE USER", " CREATE ROLE", " SHUTDOWN",
		} {
			if strings.Contains(upper, token) {
				return true
			}
		}
	}
	return false
}
