package ports

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// LayoutStore persists per-profile workspace dimensions. Writes are
// best-effort and never block the UI.
type LayoutStore interface {
	Load(ctx context.Context, profile string) (domain.LayoutSettings, error)
	Save(ctx context.Context, profile string, settings domain.LayoutSettings) error
}
