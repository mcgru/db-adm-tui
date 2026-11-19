package app

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
	"github.com/mcgru/db-adm-tui/internal/profile"
)

const metadataTimeout = 30 * time.Second

// Session is the application-owned live connection. It serializes all engine
// operations and owns the only credential copy kept in memory.
type Session struct {
	mu         sync.Mutex
	bundle     ports.SessionBundle
	profile    domain.Profile
	credential Credential
	connector  ports.Connector
	resolver   *profile.Resolver
	dead       bool
	display    atomic.Value
	alive      atomic.Bool
}

func newSession(bundle ports.SessionBundle, p domain.Profile, cred Credential, connector ports.Connector, resolver *profile.Resolver) *Session {
	s := &Session{
		bundle:     bundle,
		profile:    p,
		credential: cred,
		connector:  connector,
		resolver:   resolver,
	}
	s.display.Store(p)
	s.alive.Store(true)
	return s
}

// ProfileSnapshot returns the current profile, including the switched database.
func (s *Session) ProfileSnapshot() domain.Profile {
	if s == nil {
		return domain.Profile{}
	}
	value := s.display.Load()
	if value == nil {
		return domain.Profile{}
	}
	p, _ := value.(domain.Profile)
	return p
}

// CurrentDatabase returns the database of the live connection.
func (s *Session) CurrentDatabase() string {
	return s.ProfileSnapshot().Database
}

// Alive reports whether the session can still be used.
func (s *Session) Alive() bool {
	if s == nil {
		return false
	}
	return s.alive.Load()
}

// Close closes the underlying connection and marks the session dead.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

func (s *Session) closeLocked() error {
	if s.dead {
		return nil
	}
	s.dead = true
	s.alive.Store(false)
	if s.bundle.Session == nil {
		return nil
	}
	err := s.bundle.Session.Close()
	s.bundle = ports.SessionBundle{}
	return err
}

// Ping checks that the session is still alive.
func (s *Session) Ping(ctx context.Context) error {
	if s == nil {
		return domain.NewError(domain.KindConfig, "ping", "no active connection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead || s.bundle.Session == nil {
		return domain.NewError(domain.KindConfig, "ping", "no active connection")
	}
	return s.bundle.Session.Ping(ctx)
}

// Databases returns the databases visible to the connected user.
func (s *Session) Databases(ctx context.Context) ([]domain.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ready(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.Catalog.Databases(ctx)
}

// Children returns one node's children, switching the PostgreSQL database
// first when needed.
func (s *Session) Children(ctx context.Context, ref domain.ObjectRef) ([]domain.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ready(); err != nil {
		return nil, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.Catalog.Children(ctx, ref)
}

// Principals returns native users or roles for the Security node.
func (s *Session) Principals(ctx context.Context, ref domain.ObjectRef) ([]domain.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ready(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.Catalog.Principals(ctx, ref)
}

// OrderKey returns the stable paging order for an object.
func (s *Session) OrderKey(ctx context.Context, ref domain.ObjectRef) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ready(); err != nil {
		return "", err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.Catalog.OrderKey(ctx, ref)
}

// Page returns one read-only data window. The caller owns cancellation; user
// data pages have no default timeout.
func (s *Session) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyData(); err != nil {
		return domain.DataPage{}, err
	}
	if err := s.ensureDatabaseLocked(ctx, query.Ref.Database); err != nil {
		return domain.DataPage{}, err
	}
	if err := s.readyData(); err != nil {
		return domain.DataPage{}, err
	}
	return s.bundle.DataReader.Page(ctx, query)
}

// Structure returns the columns or sampled fields of one object.
func (s *Session) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyStats(); err != nil {
		return domain.Structure{}, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return domain.Structure{}, err
	}
	if err := s.readyStats(); err != nil {
		return domain.Structure{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.StatsReader.Structure(ctx, ref)
}

// Indexes returns the object's index metadata.
func (s *Session) Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyStats(); err != nil {
		return nil, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return nil, err
	}
	if err := s.readyStats(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.StatsReader.Indexes(ctx, ref)
}

// Statistics returns the estimated count and size metadata for one object.
func (s *Session) Statistics(ctx context.Context, ref domain.ObjectRef) (domain.Statistics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyStats(); err != nil {
		return domain.Statistics{}, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return domain.Statistics{}, err
	}
	if err := s.readyStats(); err != nil {
		return domain.Statistics{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.StatsReader.Statistics(ctx, ref)
}

// ExactCount computes the whole-object count. The caller owns cancellation;
// an exact count has no default timeout.
func (s *Session) ExactCount(ctx context.Context, ref domain.ObjectRef) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyStats(); err != nil {
		return 0, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return 0, err
	}
	if err := s.readyStats(); err != nil {
		return 0, err
	}
	return s.bundle.StatsReader.ExactCount(ctx, ref)
}

func (s *Session) ready() error {
	if s.dead || s.bundle.Session == nil || s.bundle.Catalog == nil {
		return domain.NewError(domain.KindConfig, "session", "no active connection")
	}
	return nil
}

func (s *Session) readyData() error {
	if s.dead || s.bundle.Session == nil || s.bundle.DataReader == nil {
		return domain.NewError(domain.KindConfig, "data", "data browsing is not available")
	}
	return nil
}

func (s *Session) readyStats() error {
	if s.dead || s.bundle.Session == nil || s.bundle.StatsReader == nil {
		return domain.NewError(domain.KindConfig, "stats", "object metadata is not available")
	}
	return nil
}

// ensureDatabaseLocked switches the one PostgreSQL connection to db. The
// caller holds s.mu.
func (s *Session) ensureDatabaseLocked(ctx context.Context, db string) error {
	if s.profile.Engine != domain.EnginePostgreSQL || db == "" || db == s.profile.Database {
		return nil
	}
	return s.switchDatabaseLocked(ctx, db)
}

// switchDatabaseLocked closes the current connection, re-resolves the source,
// and opens the same profile against db. The caller holds s.mu.
func (s *Session) switchDatabaseLocked(ctx context.Context, db string) error {
	next := s.profile
	next.Database = db
	secret := s.credential.Password
	if next.Source != nil && next.Source.Kind != domain.SourcePrompt {
		resolved, _, err := s.resolver.ResolveSource(next)
		if err != nil {
			_ = s.closeLocked()
			return connectionError(next, err, "")
		}
		secret = resolved
	} else if !s.credential.Provided {
		_ = s.closeLocked()
		return domain.NewError(domain.KindConfig, "switch-database", "a password is required to switch databases")
	}

	_ = s.bundle.Session.Close()
	s.bundle = ports.SessionBundle{}
	s.dead = false
	s.alive.Store(false)

	ctx, cancel := context.WithTimeout(ctx, connectBudget)
	defer cancel()
	bundle, err := s.connector.Connect(ctx, next, secret)
	if err != nil {
		s.dead = true
		return connectionError(next, err, secret)
	}
	s.bundle = bundle
	s.profile = next
	s.credential = Credential{Password: secret, Provided: true}
	s.display.Store(next)
	s.alive.Store(true)
	return nil
}
