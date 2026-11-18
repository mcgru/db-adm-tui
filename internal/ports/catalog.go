package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// Catalog exposes lazy tree navigation for one engine.
type Catalog interface {
	Databases(ctx context.Context) ([]domain.Node, error)
	Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error)
	Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error)
	OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error)
}

// SessionBundle is the capability set returned by one connection.
type SessionBundle struct {
	Session    Connection
	Catalog    Catalog
	DataReader DataReader
}
