package mongo

import (
	"context"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.ModeController = (*connection)(nil)

// SetReadOnly is a no-op: MongoDB has no session-level read-only setting, so
// read-only mode is enforced by the tool guard only.
func (c *connection) SetReadOnly(ctx context.Context, readOnly bool) error {
	return nil
}

// SessionReadOnly is always false for MongoDB.
func (c *connection) SessionReadOnly() bool {
	return false
}

// Elevated reports known server-wide roles or the anyAction privilege.
func (c *connection) Elevated(ctx context.Context) (bool, error) {
	status, err := c.connectionStatus(ctx)
	if err != nil {
		return false, safeError("mode", c.password, err)
	}
	return mongoElevated(status), nil
}

func mongoElevated(status mongoConnectionStatus) bool {
	for _, role := range status.AuthInfo.AuthenticatedUserRoles {
		switch strings.ToLower(role.Role) {
		case "root", "__system", "clusteradmin", "useradminanydatabase",
			"dbadminanydatabase", "readwriteanydatabase", "backup", "restore":
			return true
		}
	}
	for _, privilege := range status.AuthInfo.AuthenticatedUserPrivileges {
		for _, action := range privilege.Actions {
			if strings.EqualFold(action, "anyAction") {
				return true
			}
		}
	}
	return false
}
