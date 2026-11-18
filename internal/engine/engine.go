// Package engine dispatches connection attempts to the engine adapters.
package engine

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/engine/mongo"
	"github.com/mcgru/db-adm-tui/internal/engine/mysql"
	"github.com/mcgru/db-adm-tui/internal/engine/postgres"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

// Connector is the composition-root connector for all supported engines.
type Connector struct{}

// NewConnector returns a connector covering PostgreSQL, MySQL/MariaDB, and
// MongoDB.
func NewConnector() ports.Connector { return Connector{} }

// Connect dispatches to the engine adapter.
func (Connector) Connect(ctx context.Context, p domain.Profile, password string) (ports.SessionBundle, error) {
	switch p.Engine {
	case domain.EnginePostgreSQL:
		return postgres.NewConnector().Connect(ctx, p, password)
	case domain.EngineMySQL:
		return mysql.NewConnector().Connect(ctx, p, password)
	case domain.EngineMongoDB:
		return mongo.NewConnector().Connect(ctx, p, password)
	default:
		return ports.SessionBundle{}, domain.NewError(domain.KindConfig, "connect", "the engine is not supported")
	}
}

// Close implements ports.Connector.
func (Connector) Close() error { return nil }
