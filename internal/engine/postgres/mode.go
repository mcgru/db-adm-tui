package postgres

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.ModeController = (*connection)(nil)

// SetReadOnly sets PostgreSQL's default_transaction_read_only for the session.
func (c *connection) SetReadOnly(ctx context.Context, readOnly bool) error {
	value := "off"
	if readOnly {
		value = "on"
	}
	if _, err := c.conn.ExecContext(ctx, "SET default_transaction_read_only = "+value); err != nil {
		return safeError("mode", c.password, err)
	}
	c.readOnly.Store(readOnly)
	return nil
}

// SessionReadOnly reports the last session-level setting applied successfully.
func (c *connection) SessionReadOnly() bool {
	return c.readOnly.Load()
}

// Elevated reports PostgreSQL role attributes that make the account a
// superuser or a cluster-level administrator.
func (c *connection) Elevated(ctx context.Context) (bool, error) {
	var elevated bool
	err := c.conn.QueryRowContext(ctx, `
SELECT rolsuper OR rolcreaterole OR rolcreatedb
FROM pg_roles
WHERE rolname = current_user`).Scan(&elevated)
	if err != nil {
		return false, safeError("mode", c.password, err)
	}
	return elevated, nil
}
