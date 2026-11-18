// Package mongo opens MongoDB connections through the official driver.
package mongo

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

const pingTimeout = 10 * time.Second

type connector struct{}

// NewConnector returns a MongoDB connector.
func NewConnector() ports.Connector { return connector{} }

// Close implements ports.Connector.
func (connector) Close() error { return nil }

// Connect opens one client, pings it with a 10 second budget, and disconnects
// on failure. The password never appears in an error.
func (connector) Connect(ctx context.Context, p domain.Profile, password string) (ports.Connection, error) {
	uri := buildURI(p, password)
	opts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(pingTimeout).
		SetServerSelectionTimeout(pingTimeout)
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, safeError("connect", password, err)
	}
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, safeError("ping", password, err)
	}
	return &connection{client: client, password: password}, nil
}

type connection struct {
	client   *mongo.Client
	password string
}

func (c *connection) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := c.client.Ping(pingCtx, nil); err != nil {
		return safeError("ping", c.password, err)
	}
	return nil
}

func (c *connection) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.client.Disconnect(ctx)
}

func buildURI(p domain.Profile, password string) string {
	if p.Source != nil && p.Source.Selector != "" && strings.HasPrefix(p.Source.Selector, "mongodb://") {
		return injectCredentials(p.Source.Selector, p.User, password)
	}
	u := url.URL{
		Scheme: "mongodb",
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
	return u.String()
}

// injectCredentials replays a non-secret selector with the resolved password.
// It does not use net/url for the authority because a MongoDB replica-set host
// list (host1:port1,host2:port2) is a valid URI authority but an invalid URL.
func injectCredentials(selector, user, password string) string {
	const prefix = "mongodb://"
	rest := strings.TrimPrefix(selector, prefix)
	if at := strings.Index(rest, "@"); at >= 0 {
		if slash := strings.Index(rest, "/"); slash < 0 || at < slash {
			rest = rest[at+1:]
		}
	}
	userinfo := ""
	switch {
	case password != "":
		userinfo = url.UserPassword(user, password).String() + "@"
	case user != "":
		userinfo = url.User(user).String() + "@"
	}
	return prefix + userinfo + rest
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
		var cmdErr mongo.CommandError
		var cmdPtr *mongo.CommandError
		switch {
		case errors.As(err, &cmdErr):
			switch cmdErr.Code {
			case 18:
				kind = domain.KindAuth
			case 13:
				kind = domain.KindPermission
			case 11, 2:
				kind = domain.KindConfig
			}
		case errors.As(err, &cmdPtr):
			switch cmdPtr.Code {
			case 18:
				kind = domain.KindAuth
			case 13:
				kind = domain.KindPermission
			case 11, 2:
				kind = domain.KindConfig
			}
		}
		message = domain.RedactSecret(domain.Redact(message), password)
		cause = errors.New(message)
	}
	return &domain.Error{Kind: kind, Op: op, Message: message, Err: cause}
}

var _ ports.Connection = (*connection)(nil)
