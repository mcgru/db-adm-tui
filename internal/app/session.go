package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/guard"
	"github.com/mcgru/db-adm-tui/internal/ports"
	"github.com/mcgru/db-adm-tui/internal/profile"
)

const metadataTimeout = 30 * time.Second

// resultsMaxRows is the materialized result window cap.
const resultsMaxRows = 10000

// Session is the application-owned live connection. It serializes all engine
// operations and owns the only credential copy kept in memory.
type Session struct {
	mu         sync.Mutex
	bundle     ports.SessionBundle
	profile    domain.Profile
	credential Credential
	connector  ports.Connector
	resolver   *profile.Resolver
	history    ports.HistoryStore
	historyKey string
	dead       bool
	display    atomic.Value
	alive      atomic.Bool
	readOnly   atomic.Bool

	queryCancelMu sync.Mutex
	queryCancel   context.CancelFunc
	queryCtx      context.Context
	cursor        ports.QueryCursor
	resultRows    int
	resultStart   time.Time
	resultsOpen   atomic.Bool
	historyWarn   string
	modeWarn      string
}

func newSession(bundle ports.SessionBundle, p domain.Profile, cred Credential, connector ports.Connector, resolver *profile.Resolver, history ports.HistoryStore) *Session {
	s := &Session{
		bundle:     bundle,
		profile:    p,
		credential: cred,
		connector:  connector,
		resolver:   resolver,
		history:    history,
		historyKey: p.Key(),
	}
	s.display.Store(p)
	s.alive.Store(true)
	s.readOnly.Store(true)
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

// ReadOnly reports the tool's current read-only mode. Every new session starts
// read-only, and every database switch resets it.
func (s *Session) ReadOnly() bool {
	if s == nil {
		return true
	}
	return s.readOnly.Load()
}

// SetReadOnly applies the mode to the native session layer where supported and
// then updates the tool mode. MongoDB has no session layer; its guard still
// applies while read-only.
func (s *Session) SetReadOnly(ctx context.Context, readOnly bool) error {
	if s == nil {
		return domain.NewError(domain.KindConfig, "mode", "no active connection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead || s.bundle.Session == nil {
		return domain.NewError(domain.KindConfig, "mode", "no active connection")
	}
	if err := s.readyForEngine(); err != nil {
		return err
	}
	if s.readOnly.Load() == readOnly {
		return nil
	}
	if err := s.applyReadOnlyLocked(ctx, readOnly); err != nil {
		return s.modeError(err)
	}
	return nil
}

// Elevated reports native superuser/root or server-admin rights.
func (s *Session) Elevated(ctx context.Context) (bool, error) {
	if s == nil {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead || s.bundle.Session == nil || s.bundle.ModeController == nil {
		return false, nil
	}
	if err := s.readyForEngine(); err != nil {
		return false, err
	}
	elevated, err := s.bundle.ModeController.Elevated(ctx)
	if err != nil {
		return false, s.modeError(err)
	}
	return elevated, nil
}

func (s *Session) applyReadOnlyLocked(ctx context.Context, readOnly bool) error {
	if s.bundle.ModeController != nil {
		if err := s.bundle.ModeController.SetReadOnly(ctx, readOnly); err != nil {
			return err
		}
	}
	s.readOnly.Store(readOnly)
	return nil
}

// initializeReadOnly applies the native read-only layer even though the local
// state already starts true, so a new connection cannot miss the second layer.
func (s *Session) initializeReadOnly(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead || s.bundle.Session == nil {
		return domain.NewError(domain.KindConfig, "mode", "no active connection")
	}
	return s.applyReadOnlyLocked(ctx, true)
}

func (s *Session) modeError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*domain.Error); ok {
		return err
	}
	message := domain.RedactSecret(err.Error(), s.credential.Password)
	return domain.NewError(domain.KindEngine, "mode", message)
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
	s.cancelQuery()
	s.closeResultsLocked()
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
	if s.cursor != nil {
		return domain.NewError(domain.KindConfig, "ping", "results open — Esc to release")
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

// Privileges returns native grants and memberships for the selected context
// or principal.
func (s *Session) Privileges(ctx context.Context, ref domain.ObjectRef) (domain.PrivilegeReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.readyPrivileges(); err != nil {
		return domain.PrivilegeReport{}, err
	}
	if err := s.ensureDatabaseLocked(ctx, ref.Database); err != nil {
		return domain.PrivilegeReport{}, err
	}
	if err := s.readyPrivileges(); err != nil {
		return domain.PrivilegeReport{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	return s.bundle.PrivilegeReader.Privileges(ctx, ref)
}

// ValidateQuery applies the local guard and returns a safe reason before any
// engine call. It never touches the connection.
func (s *Session) ValidateQuery(statement string) error {
	if s == nil {
		return domain.NewError(domain.KindConfig, "query", "no active connection")
	}
	if strings.TrimSpace(statement) == "" {
		return domain.NewError(domain.KindConfig, "query", "the statement is empty")
	}
	decision := guard.CheckSQLMode(statement, s.ReadOnly())
	if s.ProfileSnapshot().Engine == domain.EngineMongoDB {
		decision = guard.CheckMongoMode(statement, s.ReadOnly())
	}
	if !decision.Allowed {
		return guardError(decision)
	}
	return nil
}

func guardError(decision guard.Decision) error {
	message := decision.Reason
	if decision.ModeHint {
		message += " Ctrl+X Ctrl+W to switch."
	}
	return domain.NewError(domain.KindGuard, "query", message)
}

// ExecuteQuery validates and opens one forward-only query cursor, returning
// the first page. The cursor stays open while it has more rows.
func (s *Session) ExecuteQuery(ctx context.Context, request domain.QueryRequest, limit int) (domain.QueryPage, error) {
	if limit <= 0 {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result window is not valid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead || s.bundle.Session == nil || s.bundle.QueryExecutor == nil {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "query execution is not available")
	}
	if s.cursor != nil {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "results open — Esc to release")
	}
	if err := s.validateLocked(request.Statement); err != nil {
		return domain.QueryPage{}, err
	}
	queryCtx, cancel := context.WithCancel(ctx)
	start := time.Now()
	cursor, err := s.bundle.QueryExecutor.Query(queryCtx, request)
	if err != nil {
		cancel()
		s.recordHistoryLocked(request.Statement, start, queryStatus(err))
		return domain.QueryPage{}, s.redactQueryError(err)
	}
	page, err := cursor.Next(queryCtx, limit)
	if err != nil {
		_ = cursor.Close(context.Background())
		cancel()
		s.recordHistoryLocked(request.Statement, start, queryStatus(err))
		return domain.QueryPage{}, s.redactQueryError(err)
	}
	page.Elapsed = time.Since(start)
	s.queryCtx = queryCtx
	s.resultRows = len(page.Grid.Rows)
	s.resultStart = start
	if page.HasMore {
		s.cursor = cursor
		s.setQueryCancel(cancel)
		s.resultsOpen.Store(true)
	} else {
		_ = cursor.Close(context.Background())
		cancel()
		s.closeResultsLocked()
	}
	s.recordHistoryLocked(request.Statement, start, "ok")
	return page, nil
}

// ResultsNext drains the open query cursor into the next page. The app owns
// the 10,000-row window cap.
func (s *Session) ResultsNext(ctx context.Context, limit int) (domain.QueryPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor == nil || s.queryCtx == nil {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "no query results are open")
	}
	remaining := resultsMaxRows - s.resultRows
	if remaining <= 0 {
		s.closeResultsLocked()
		return domain.QueryPage{CapReached: true}, nil
	}
	if limit <= 0 || limit > remaining {
		limit = remaining
	}
	page, err := s.cursor.Next(s.queryCtx, limit)
	if err != nil {
		s.closeResultsLocked()
		return domain.QueryPage{}, s.redactQueryError(err)
	}
	page.Elapsed = time.Since(s.resultStart)
	s.resultRows += len(page.Grid.Rows)
	if page.HasMore && s.resultRows >= resultsMaxRows {
		page.HasMore = false
		page.CapReached = true
		s.closeResultsLocked()
		return page, nil
	}
	if !page.HasMore {
		s.closeResultsLocked()
	}
	return page, nil
}

// CancelQuery interrupts a running query or paging call without waiting for
// the serialized session lock. The cursor is cleaned up when the call returns.
func (s *Session) CancelQuery() {
	s.queryCancelMu.Lock()
	cancel := s.queryCancel
	s.queryCancel = nil
	s.queryCancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// CloseResults releases the open cursor and its session lease.
func (s *Session) CloseResults() {
	s.cancelQuery()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeResultsLocked()
}

// ResultsOpen reports whether a query cursor still holds the session lease.
func (s *Session) ResultsOpen() bool {
	if s == nil {
		return false
	}
	return s.resultsOpen.Load()
}

// LastWarning returns the most recent non-fatal history write failure.
func (s *Session) LastWarning() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	warning := s.modeWarn
	s.modeWarn = ""
	if history := s.historyWarn; history != "" {
		if warning != "" {
			warning += " " + history
		} else {
			warning = history
		}
		s.historyWarn = ""
	}
	return warning
}

func (s *Session) validateLocked(statement string) error {
	if strings.TrimSpace(statement) == "" {
		return domain.NewError(domain.KindConfig, "query", "the statement is empty")
	}
	decision := guard.CheckSQLMode(statement, s.readOnly.Load())
	if s.profile.Engine == domain.EngineMongoDB {
		decision = guard.CheckMongoMode(statement, s.readOnly.Load())
	}
	if !decision.Allowed {
		return guardError(decision)
	}
	return nil
}

func (s *Session) setQueryCancel(cancel context.CancelFunc) {
	s.queryCancelMu.Lock()
	s.queryCancel = cancel
	s.queryCancelMu.Unlock()
}

func (s *Session) cancelQuery() {
	s.queryCancelMu.Lock()
	cancel := s.queryCancel
	s.queryCancel = nil
	s.queryCancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Session) closeResultsLocked() {
	if s.cursor != nil {
		_ = s.cursor.Close(context.Background())
		s.cursor = nil
	}
	s.cancelQuery()
	s.queryCtx = nil
	s.resultRows = 0
	s.resultStart = time.Time{}
	s.resultsOpen.Store(false)
}

func (s *Session) recordHistoryLocked(statement string, start time.Time, status string) {
	if s.history == nil {
		return
	}
	duration := time.Since(start)
	mode := "r/o"
	if !s.readOnly.Load() {
		mode = "r/w"
	}
	entry := domain.HistoryEntry{
		Time:       time.Now().UTC(),
		Profile:    s.historyKey,
		Engine:     s.profile.Engine,
		Statement:  domain.RedactSecret(statement, s.credential.Password),
		Mode:       mode,
		DurationMs: duration.Milliseconds(),
		Duration:   duration,
		Status:     status,
	}
	if err := s.history.Append(context.Background(), entry); err != nil {
		s.historyWarn = "history write failed: " + domain.RedactSecrets(err.Error(), s.credential.Password)
	}
}

func queryStatus(err error) string {
	if errors.Is(err, context.Canceled) || domain.KindOf(err) == domain.KindCancelled {
		return "cancelled"
	}
	return "error"
}

func (s *Session) redactQueryError(err error) error {
	if err == nil {
		return nil
	}
	message := domain.RedactSecret(err.Error(), s.credential.Password)
	kind := domain.KindOf(err)
	switch {
	case errors.Is(err, context.Canceled):
		kind = domain.KindCancelled
		message = "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		kind = domain.KindTimeout
		message = "query timed out"
	}
	return &domain.Error{
		Kind:    kind,
		Op:      "query",
		Message: message,
	}
}

func (s *Session) ready() error {
	if s.dead || s.bundle.Session == nil || s.bundle.Catalog == nil {
		return domain.NewError(domain.KindConfig, "session", "no active connection")
	}
	return s.readyForEngine()
}

func (s *Session) readyForEngine() error {
	if s.cursor != nil {
		return domain.NewError(domain.KindConfig, "session", "results open — Esc to release")
	}
	return nil
}

func (s *Session) readyData() error {
	if s.dead || s.bundle.Session == nil || s.bundle.DataReader == nil {
		return domain.NewError(domain.KindConfig, "data", "data browsing is not available")
	}
	return s.readyForEngine()
}

func (s *Session) readyStats() error {
	if s.dead || s.bundle.Session == nil || s.bundle.StatsReader == nil {
		return domain.NewError(domain.KindConfig, "stats", "object metadata is not available")
	}
	return s.readyForEngine()
}

func (s *Session) readyPrivileges() error {
	if s.dead || s.bundle.Session == nil || s.bundle.PrivilegeReader == nil {
		return domain.NewError(domain.KindConfig, "privileges", "privilege inspection is not available")
	}
	return s.readyForEngine()
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
	s.readOnly.Store(true)
	if err := s.applyReadOnlyLocked(ctx, true); err != nil {
		s.modeWarn = "Read-only session could not be enabled: " + s.modeError(err).Error()
	}
	return nil
}
