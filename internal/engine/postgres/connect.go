// Package postgres opens PostgreSQL connections through pgx's database/sql
// adapter.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

const (
	pingTimeout = 10 * time.Second
	sslRequire  = "require"
	sslDisable  = "disable"
)

type connector struct{}

// NewConnector returns a PostgreSQL connector.
func NewConnector() ports.Connector { return connector{} }

// Close implements ports.Connector. PostgreSQL connections are owned by their
// session, so the factory holds no resources.
func (connector) Close() error { return nil }

// Connect opens one database/sql connection, pings it with a 10 second budget,
// and closes it on failure. TLS is required by default. A profile with
// AllowNoTLS retries the same connection without TLS after a TLS-level failure.
// The password is never placed in an error.
func (connector) Connect(ctx context.Context, p domain.Profile, password string) (ports.SessionBundle, error) {
	modes := connectionModes(p)
	var lastErr error
	for i, sslmode := range modes {
		bundle, err := connectOnce(ctx, p, password, sslmode)
		if err == nil {
			return bundle, nil
		}
		lastErr = err
		if i+1 < len(modes) && !retryWithoutTLS(err) {
			return ports.SessionBundle{}, err
		}
	}
	return ports.SessionBundle{}, lastErr
}

func connectionModes(p domain.Profile) []string {
	modes := []string{sslRequire}
	if p.AllowNoTLS {
		modes = append(modes, sslDisable)
	}
	return modes
}

// retryWithoutTLS only falls back for connection-level failures. Auth,
// permission, and configuration errors would fail the same way without TLS and
// must not double the login attempts.
func retryWithoutTLS(err error) bool {
	return domain.KindOf(err) == domain.KindEngine
}

func connectOnce(ctx context.Context, p domain.Profile, password, sslmode string) (ports.SessionBundle, error) {
	db, err := sql.Open("pgx", buildDSN(p, password, sslmode))
	if err != nil {
		return ports.SessionBundle{}, safeError("connect", password, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	conn, err := db.Conn(pingCtx)
	if err != nil {
		_ = db.Close()
		return ports.SessionBundle{}, safeError("connect", password, err)
	}
	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		_ = db.Close()
		return ports.SessionBundle{}, safeError("ping", password, err)
	}
	live := &connection{conn: conn, db: db, password: password}
	return ports.SessionBundle{Session: live, Catalog: newCatalog(live), DataReader: live, StatsReader: live}, nil
}

type connection struct {
	conn     *sql.Conn
	db       *sql.DB
	password string
}

func (c *connection) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := c.conn.PingContext(pingCtx); err != nil {
		return safeError("ping", c.password, err)
	}
	return nil
}

func (c *connection) Close() error {
	err := c.conn.Close()
	if dbErr := c.db.Close(); err == nil {
		err = dbErr
	}
	return err
}

func buildDSN(p domain.Profile, password, sslmode string) string {
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Database != "" {
		u.Path = "/" + p.Database
	}
	switch {
	case password != "":
		u.User = url.UserPassword(p.User, password)
	case p.User != "":
		u.User = url.User(p.User)
	}
	q := url.Values{}
	q.Set("sslmode", sslmode)
	// Resolution belongs to the profile adapter. An empty passfile stops pgx
	// from reading PGPASSFILE/~/.pgpass a second time behind our back.
	q.Set("passfile", "")
	u.RawQuery = q.Encode()
	return u.String()
}

func safeError(op, password string, err error) *domain.Error {
	kind := domain.KindEngine
	message := err.Error()
	cause := err
	if errors.Is(err, context.DeadlineExceeded) {
		kind = domain.KindTimeout
		message = "connection timed out"
	} else if errors.Is(err, context.Canceled) {
		kind = domain.KindCancelled
		message = "connection cancelled"
	} else {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "28P01", "28000":
				kind = domain.KindAuth
			case "42501":
				kind = domain.KindPermission
			case "3D000":
				kind = domain.KindConfig
			}
		}
		message = domain.RedactSecret(domain.Redact(message), password)
		// Never expose the raw driver error through Unwrap: it may repeat the
		// DSN or password.
		cause = errors.New(message)
	}
	return &domain.Error{Kind: kind, Op: op, Message: message, Err: cause}
}

var _ ports.Connection = (*connection)(nil)
