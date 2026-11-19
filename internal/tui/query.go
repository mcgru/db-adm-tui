package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

// resultsState is the query metadata shown above the active grid.
type resultsState struct {
	statement  string
	err        string
	elapsed    time.Duration
	capReached bool
}

func (m *Model) syncQueryFocus() {
	if m == nil {
		return
	}
	if m.screen == ScreenWorkspace && m.focus == FocusQuery {
		m.query.Focus()
		return
	}
	m.query.Blur()
}

func (m *Model) syncQueryPlaceholder() {
	if m == nil {
		return
	}
	m.query.Placeholder = queryPlaceholder(m.profileForStatus().Engine, m.currentQueryRef())
}

func (m Model) currentQueryRef() domain.ObjectRef {
	if m.resultsActive {
		return m.resultsSaved.node.Ref
	}
	if m.details.has {
		return m.details.node.Ref
	}
	return domain.ObjectRef{}
}

func (m Model) queryAvailability() (bool, string) {
	if m.session == nil {
		return false, "no active connection"
	}
	ref := m.currentQueryRef()
	if m.profileForStatus().Engine == domain.EngineMongoDB {
		if ref.Kind != domain.KindCollection && ref.Kind != domain.KindView {
			return false, "select a collection before running MQL"
		}
		if ref.Name == "" || ref.Database == "" {
			return false, "select a collection before running MQL"
		}
	}
	if m.queryDatabase(ref) == "" {
		return false, "select a database first"
	}
	return true, ""
}

func (m Model) queryDatabase(ref domain.ObjectRef) string {
	if m.profileForStatus().Engine == domain.EngineMongoDB && ref.Database != "" {
		return ref.Database
	}
	return m.currentDatabase()
}

func (m Model) startQuery() (tea.Model, tea.Cmd) {
	if m.screen != ScreenWorkspace || m.session == nil {
		return m, nil
	}
	if m.queryRunning {
		return m, nil
	}
	if m.resultsActive && m.session.ResultsOpen() {
		m.setMessage(levelInfo, "results open — Esc to release")
		return m, nil
	}
	statement := strings.TrimSpace(m.query.Value())
	if statement == "" {
		m.setMessage(levelInfo, "enter a statement before running it")
		return m, nil
	}
	if available, hint := m.queryAvailability(); !available {
		m.setMessage(levelInfo, hint)
		return m, nil
	}
	if err := m.session.ValidateQuery(statement); err != nil {
		if domain.KindOf(err) == domain.KindGuard {
			m.setMessage(levelError, "Blocked: "+oneLine(err.Error()))
			return m, nil
		}
		m.setMessage(levelError, oneLine(err.Error()))
		return m, nil
	}
	ref := m.currentQueryRef()
	if m.resultsActive {
		m.leaveResults()
	}
	m.enterResults(statement)
	m.queryGen++
	gen := m.queryGen
	m.queryRunning = true
	m.queryStarted = time.Now()
	m.data.loading = true
	m.setMessage(levelInfo, "running 00:00:00 · Esc cancels")
	m.recordHistoryLocal(statement)

	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	request := domain.QueryRequest{
		Statement: statement,
		Database:  m.queryDatabase(ref),
		Ref:       ref,
	}
	run := func() tea.Msg {
		page, err := session.ExecuteQuery(ctx, request, dataPageSize)
		return queryDoneMsg{gen: gen, statement: statement, page: page, err: err}
	}
	return m, tea.Batch(run, m.queryTickCmd(gen))
}

func (m *Model) enterResults(statement string) {
	if m.data.cancel != nil {
		m.data.cancel()
		m.data.cancel = nil
	}
	if m.details.exactCancel != nil {
		m.details.exactCancel()
		m.details.exactCancel = nil
	}
	m.details.exactLoading = false
	m.exactGen++
	m.resultsSaved = m.details
	m.resultsData = m.data
	m.resultsData.cancel = nil
	m.resultsData.loading = false
	m.resultsDataGen = m.dataGen
	m.dataGen++
	m.data = newDataState()
	m.data.has = true
	m.resultsActive = true
	m.results = resultsState{statement: statement}
	m.escChord = false
	m.details.tabs = []string{"Results"}
	m.details.tab = 0
	m.details.offset = 0
	m.resultsGen++
	m.completion = completionState{}
}

func (m *Model) leaveResults() {
	if !m.resultsActive {
		return
	}
	if m.session != nil {
		m.session.CloseResults()
	}
	m.resultsActive = false
	m.queryRunning = false
	m.dataGen++
	m.data = m.resultsData
	m.dataGen = m.resultsDataGen + 1
	m.details = m.resultsSaved
	m.results = resultsState{}
	m.resultsSaved = detailsState{}
	m.resultsData = dataState{}
	m.resultsGen++
	m.queryGen++
	m.completion = completionState{}
	m.syncQueryFocus()
}

func (m Model) cancelResults() (tea.Model, tea.Cmd) {
	if m.queryRunning {
		if m.session != nil {
			m.session.CancelQuery()
		}
		m.queryRunning = false
		m.queryGen++
		m.data.loading = false
		m.setMessage(levelInfo, "cancelled · "+formatElapsed(time.Since(m.queryStarted)))
		if !m.data.loaded || len(m.data.rows) == 0 {
			m.leaveResults()
		}
		return m, nil
	}
	if m.data.loading {
		if m.session != nil {
			m.session.CancelQuery()
			m.session.CloseResults()
		}
		m.resultsGen++
		m.dataGen++
		m.data.loading = false
		m.data.hasMore = false
		m.setMessage(levelInfo, "cancelled")
		if len(m.data.rows) == 0 {
			m.leaveResults()
		}
		return m, nil
	}
	m.leaveResults()
	return m, nil
}

func (m Model) handleQueryDone(msg queryDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.queryGen {
		return m, nil
	}
	m.queryRunning = false
	m.data.loading = false
	if msg.err != nil {
		kind := domain.KindOf(msg.err)
		if errors.Is(msg.err, context.Canceled) {
			kind = domain.KindCancelled
		}
		if kind == domain.KindCancelled {
			m.setMessage(levelInfo, "cancelled")
			m.leaveResults()
			return m, nil
		}
		if kind == domain.KindGuard || kind == domain.KindConfig {
			m.setMessage(levelError, oneLine(msg.err.Error()))
			m.leaveResults()
			return m, nil
		}
		m.results.err = oneLine(msg.err.Error())
		m.results.elapsed = time.Since(m.queryStarted)
		m.setMessage(levelError, "query failed")
		return m, nil
	}
	m.results.elapsed = msg.page.Elapsed
	m.results.capReached = msg.page.CapReached
	m.data.columns = append([]domain.Column(nil), msg.page.Grid.Columns...)
	m.data.rows = append([]domain.Row(nil), msg.page.Grid.Rows...)
	m.data.windowStart = 0
	m.data.cursorRow = 0
	m.data.cursorCol = 0
	m.data.colOffset = 0
	m.data.rowOffset = 0
	m.data.hasMore = msg.page.HasMore
	m.data.loaded = true
	m.data.recount()
	m.ensureDataViewport()
	if warning := m.session.LastWarning(); warning != "" {
		m.setMessage(levelWarning, warning)
		return m, nil
	}
	m.setMessage(levelSuccess, m.resultsSummary())
	return m, nil
}

func (m Model) handleResultsPage(msg resultsPageMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.resultsGen {
		return m, nil
	}
	m.data.loading = false
	if msg.err != nil {
		kind := domain.KindOf(msg.err)
		if errors.Is(msg.err, context.Canceled) {
			kind = domain.KindCancelled
		}
		if kind == domain.KindCancelled {
			m.setMessage(levelInfo, "cancelled")
			return m, nil
		}
		m.data.err = oneLine(msg.err.Error())
		m.data.hasMore = false
		m.setMessage(levelError, "query failed")
		return m, nil
	}
	m.data.mergePage(msg.page.Grid, false)
	m.data.hasMore = msg.page.HasMore
	m.results.capReached = msg.page.CapReached
	before := len(m.data.rows)
	m.data.capWindow(false)
	if len(m.data.rows) < before {
		m.results.capReached = true
		m.data.hasMore = false
	}
	m.ensureDataViewport()
	if msg.page.CapReached {
		m.setMessage(levelWarning, "cap reached · Esc releases the results")
		return m, nil
	}
	m.message = ""
	m.messageLevel = levelNone
	return m, nil
}

func (m *Model) nextResultsPage(limit int) tea.Cmd {
	if m.session == nil || !m.resultsActive || m.queryRunning || m.data.loading || !m.data.hasMore {
		return nil
	}
	if limit <= 0 {
		limit = dataPageSize
	}
	m.resultsGen++
	gen := m.resultsGen
	m.data.loading = true
	m.setMessage(levelInfo, "loading rows…")
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		page, err := session.ResultsNext(ctx, limit)
		return resultsPageMsg{gen: gen, page: page, err: err}
	}
}

func (m *Model) fetchAllResults() tea.Cmd {
	if m.session == nil || !m.resultsActive || m.queryRunning || m.data.loading || !m.data.hasMore {
		if m.resultsActive && !m.data.hasMore {
			m.setMessage(levelInfo, "end of data")
		}
		return nil
	}
	remaining := dataMaxWindowRows - len(m.data.rows)
	if remaining <= 0 {
		m.setMessage(levelWarning, "cap reached · Esc releases the results")
		return nil
	}
	return m.nextResultsPage(remaining)
}

func (m *Model) queryTickCmd(gen uint64) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return queryTickMsg{gen: gen}
	})
}

const escChordWindow = 300 * time.Millisecond

// armEscChord delays the no-op Esc path briefly so the next Tab can open
// completion instead of moving focus. Esc actions with an immediate
// consequence (closing completion, leaving Results, cancelling a query) never
// arm the chord.
func (m *Model) armEscChord() tea.Cmd {
	m.escChord = true
	m.escSeq++
	seq := m.escSeq
	return tea.Tick(escChordWindow, func(time.Time) tea.Msg {
		return escChordTimeoutMsg{seq: seq}
	})
}

func (m *Model) clearEscChord() {
	m.escChord = false
}

func (m Model) handleEscChordTimeout(msg escChordTimeoutMsg) (tea.Model, tea.Cmd) {
	if !m.escChord || msg.seq != m.escSeq {
		return m, nil
	}
	m.escChord = false
	if m.modal == modalStatement {
		m.closeStatementModal()
	}
	return m, nil
}

func (m Model) handleQueryTick(msg queryTickMsg) (tea.Model, tea.Cmd) {
	if !m.queryRunning || msg.gen != m.queryGen {
		return m, nil
	}
	return m, m.queryTickCmd(msg.gen)
}

func (m *Model) recordHistoryLocal(statement string) {
	if len(m.history) > 0 && m.history[len(m.history)-1] == statement {
		m.historyIndex = len(m.history)
		m.historyDraft = ""
		return
	}
	m.history = append(m.history, statement)
	m.historyIndex = len(m.history)
	m.historyDraft = ""
}

func (m *Model) historyPrev() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIndex > len(m.history) {
		m.historyIndex = len(m.history)
	}
	if m.historyIndex == len(m.history) {
		m.historyDraft = m.query.Value()
	}
	if m.historyIndex > 0 {
		m.historyIndex--
	}
	m.query.SetValue(m.history[m.historyIndex])
	m.query.SetCursor(len([]rune(m.history[m.historyIndex])))
}

func (m *Model) historyNext() {
	if len(m.history) == 0 || m.historyIndex >= len(m.history) {
		return
	}
	m.historyIndex++
	if m.historyIndex == len(m.history) {
		m.query.SetValue(m.historyDraft)
		m.query.SetCursor(len([]rune(m.historyDraft)))
		return
	}
	m.query.SetValue(m.history[m.historyIndex])
	m.query.SetCursor(len([]rune(m.history[m.historyIndex])))
}

func (m Model) resultsSummary() string {
	if !m.resultsActive {
		return ""
	}
	if m.queryRunning {
		return "running " + formatElapsed(time.Since(m.queryStarted)) + " · Esc cancels"
	}
	if m.results.err != "" {
		return "error · Esc returns to " + oneLine(queryObjectName(m.resultsSaved.node))
	}
	elapsed := formatElapsed(m.results.elapsed)
	if !m.data.loaded {
		return "not loaded"
	}
	if len(m.data.rows) == 0 {
		return "0 rows · " + elapsed
	}
	noun := "rows"
	if len(m.data.rows) == 1 {
		noun = "row"
	}
	text := formatCount(int64(len(m.data.rows))) + " " + noun + " · " + elapsed
	if m.results.capReached {
		text += " · cap reached"
	} else if m.data.hasMore {
		text += " · more available"
	}
	return text
}

func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	hours := int(d / time.Hour)
	minutes := int((d % time.Hour) / time.Minute)
	seconds := int((d % time.Minute) / time.Second)
	millis := int((d % time.Second) / time.Millisecond)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

func queryObjectName(node domain.Node) string {
	switch {
	case node.Kind == domain.KindDatabase || node.Kind == domain.KindSchema:
		if node.Name != "" {
			return node.Name
		}
		return "the object"
	case node.Ref.Schema != "" && node.Ref.Name != "":
		return node.Ref.Schema + "." + node.Ref.Name
	case node.Ref.Database != "" && node.Ref.Name != "":
		return node.Ref.Database + "." + node.Ref.Name
	case node.Name != "":
		return node.Name
	default:
		return "the object"
	}
}

func (m Model) handleQueryKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.query.Focus()
	m.query.Placeholder = queryPlaceholder(m.profileForStatus().Engine, m.currentQueryRef())
	key := msg.String()
	action, ok := keymap.Lookup(key)
	if m.queryRunning {
		if ok {
			switch action {
			case keymap.Cancel:
				return m.cancelResults()
			case keymap.Tab, keymap.ShiftTab, keymap.Disconnect, keymap.Quit, keymap.ForceQuit:
				return m.apply(action)
			}
		}
		return m, nil
	}
	if m.escChord {
		m.escChord = false
		if key == "tab" {
			return m, m.openCompletion(false)
		}
	}
	m.message = ""
	m.messageLevel = levelNone
	if ok {
		switch action {
		case keymap.Tab, keymap.ShiftTab, keymap.Disconnect, keymap.Quit, keymap.ForceQuit:
			return m.apply(action)
		case keymap.ContextHelp:
			m.openContextHelp()
			return m, nil
		case keymap.Completion:
			return m, m.openCompletion(false)
		case keymap.Cancel:
			if m.completion.active {
				m.completion = completionState{}
				return m, nil
			}
			if m.resultsActive {
				return m.apply(keymap.Cancel)
			}
			return m, m.armEscChord()
		case keymap.Confirm:
			return m.startQuery()
		case keymap.Up:
			m.historyPrev()
			return m, nil
		case keymap.Down:
			m.historyNext()
			return m, nil
		}
	}
	updated, cmd := m.query.Update(msg)
	m.query = updated
	if m.completion.active {
		m.refreshCompletion()
	}
	return m, cmd
}
