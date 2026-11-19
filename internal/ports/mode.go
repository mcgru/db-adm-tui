package ports

import "context"

// ModeController owns the native read-only session layer and the elevated
// account check for one connection. MongoDB is a no-op for SetReadOnly and
// reports SessionReadOnly false.
type ModeController interface {
	SetReadOnly(ctx context.Context, readOnly bool) error
	SessionReadOnly() bool
	Elevated(ctx context.Context) (bool, error)
}
