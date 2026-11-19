// Package mysql opens MySQL and MariaDB connections through go-sql-driver.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

const pingTimeout = 10 * time.Second

type connector struct{}

// NewConnector returns a MySQL/MariaDB connector.
func NewConnector() ports.Connector { return connector{} }

// Close implements ports.Connector.
func (connector) Close() error { return nil }

// Connect opens one database/sql connection, pings it with a 10 second budget,
// and closes it on failure.
func (connector) Connect(ctx context.Context, p domain.Profile, password string) (ports.SessionBundle, error) {
	cfg := buildConfig(p, password)

	db, err := sql.Open("mysql", cfg.FormatDSN())
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
	return ports.SessionBundle{Session: live, Catalog: live, DataReader: live, StatsReader: live, PrivilegeReader: live, QueryExecutor: live, ModeController: live, RoleAdmin: live, RoleMatrix: live}, nil
}

func buildConfig(p domain.Profile, password string) *mysqldriver.Config {
	cfg := mysqldriver.NewConfig()
	cfg.User = p.User
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	cfg.DBName = p.Database
	cfg.Timeout = pingTimeout
	cfg.ReadTimeout = 0
	cfg.WriteTimeout = 0
	cfg.ParseTime = true
	return cfg
}

type connection struct {
	conn     *sql.Conn
	db       *sql.DB
	password string
	readOnly atomic.Bool
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
		var myErr *mysqldriver.MySQLError
		if errors.As(err, &myErr) {
			switch myErr.Number {
			case 1045:
				kind = domain.KindAuth
			case 1044, 1142:
				kind = domain.KindPermission
			case 1049:
				kind = domain.KindConfig
			}
		}
		message = domain.RedactSecrets(message, password)
		cause = errors.New(message)
	}
	return &domain.Error{Kind: kind, Op: op, Message: message, Err: cause}
}

var _ ports.Connection = (*connection)(nil)
