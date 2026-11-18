// Package ports defines the small capability interfaces shared by the
// application and its adapters.
package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// ProfileStore persists profiles. Implementations must never store secrets.
type ProfileStore interface {
	Load(ctx context.Context) ([]domain.Profile, error)
	Save(ctx context.Context, profiles []domain.Profile) error
}

// Connection is one live session.
type Connection interface {
	Ping(ctx context.Context) error
	Close() error
}

// Connector opens one database connection for a profile. The password is
// resolved by the application immediately before the call and is never stored.
type Connector interface {
	Connect(ctx context.Context, profile domain.Profile, password string) (SessionBundle, error)
	Close() error
}
