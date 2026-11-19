package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

type statisticsMsg struct {
	gen        uint64
	key        string
	statistics domain.Statistics
	err        error
}

type exactCountMsg struct {
	gen   uint64
	key   string
	count int64
	err   error
}

type exactTickMsg struct {
	gen uint64
}

type privilegesMsg struct {
	gen        uint64
	key        string
	privileges domain.PrivilegeReport
	err        error
}

type structureMsg struct {
	gen       uint64
	key       string
	structure domain.Structure
	err       error
}

type indexesMsg struct {
	gen     uint64
	key     string
	indexes []domain.Index
	err     error
}

// ensureTabInfo starts the metadata request for the active context tab.
func (m *Model) ensureTabInfo() tea.Cmd {
	switch m.details.currentTab() {
	case "Details":
		return tea.Batch(m.ensureStatistics(), m.ensureStructure())
	case "Indexes":
		return m.ensureIndexes()
	case "Privileges":
		return m.ensurePrivileges()
	}
	return nil
}

func (m *Model) ensurePrivileges() tea.Cmd {
	if m.session == nil || !m.details.has {
		return nil
	}
	d := &m.details
	if d.privilegesLoading || d.privilegesLoaded {
		return nil
	}
	d.privilegesLoading = true
	d.privilegesErr = ""
	gen := m.detailsGen
	ref := d.node.Ref
	key := ref.Key()
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		report, err := session.Privileges(ctx, ref)
		return privilegesMsg{gen: gen, key: key, privileges: report, err: err}
	}
}

func (m *Model) ensureStatistics() tea.Cmd {
	if m.session == nil || !m.details.has {
		return nil
	}
	d := &m.details
	if d.node.Kind != domain.KindTable && d.node.Kind != domain.KindCollection {
		return nil
	}
	if d.statisticsLoading || d.statisticsLoaded {
		return nil
	}
	d.statisticsLoading = true
	d.statisticsErr = ""
	gen := m.detailsGen
	ref := d.node.Ref
	key := ref.Key()
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		statistics, err := session.Statistics(ctx, ref)
		return statisticsMsg{gen: gen, key: key, statistics: statistics, err: err}
	}
}

func (m *Model) ensureStructure() tea.Cmd {
	if m.session == nil || !m.details.has {
		return nil
	}
	d := &m.details
	if d.node.Kind != domain.KindTable && d.node.Kind != domain.KindCollection {
		return nil
	}
	if d.structureLoading || d.structureLoaded {
		return nil
	}
	d.structureLoading = true
	d.structureErr = ""
	gen := m.detailsGen
	ref := d.node.Ref
	key := ref.Key()
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		structure, err := session.Structure(ctx, ref)
		return structureMsg{gen: gen, key: key, structure: structure, err: err}
	}
}

func (m *Model) ensureIndexes() tea.Cmd {
	if m.session == nil || !m.details.has {
		return nil
	}
	d := &m.details
	if d.node.Kind != domain.KindTable && d.node.Kind != domain.KindCollection {
		return nil
	}
	if d.indexesLoading || d.indexesLoaded {
		return nil
	}
	d.indexesLoading = true
	d.indexesErr = ""
	gen := m.detailsGen
	ref := d.node.Ref
	key := ref.Key()
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		indexes, err := session.Indexes(ctx, ref)
		return indexesMsg{gen: gen, key: key, indexes: indexes, err: err}
	}
}

func (m Model) handleStructure(msg structureMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.detailsGen {
		return m, nil
	}
	if !m.details.has || m.details.node.Ref.Key() != msg.key {
		return m, nil
	}
	m.details.structureLoading = false
	if msg.err != nil {
		m.details.structureErr = oneLine(msg.err.Error())
		m.details.structure = domain.Structure{}
		return m, nil
	}
	m.details.structure = msg.structure
	m.details.structureLoaded = true
	return m, nil
}

func (m Model) handleIndexes(msg indexesMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.detailsGen {
		return m, nil
	}
	if !m.details.has || m.details.node.Ref.Key() != msg.key {
		return m, nil
	}
	m.details.indexesLoading = false
	if msg.err != nil {
		m.details.indexesErr = oneLine(msg.err.Error())
		m.details.indexes = nil
		return m, nil
	}
	m.details.indexes = msg.indexes
	m.details.indexesLoaded = true
	return m, nil
}

func (m Model) handlePrivileges(msg privilegesMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.detailsGen {
		return m, nil
	}
	if !m.details.has || m.details.node.Ref.Key() != msg.key {
		return m, nil
	}
	m.details.privilegesLoading = false
	if msg.err != nil {
		m.details.privilegesErr = oneLine(msg.err.Error())
		m.details.privileges = domain.PrivilegeReport{}
		return m, nil
	}
	m.details.privileges = msg.privileges
	m.details.privilegesLoaded = true
	return m, nil
}

func (m Model) handleStatistics(msg statisticsMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.detailsGen {
		return m, nil
	}
	if !m.details.has || m.details.node.Ref.Key() != msg.key {
		return m, nil
	}
	m.details.statisticsLoading = false
	if msg.err != nil {
		m.details.statisticsErr = oneLine(msg.err.Error())
		m.details.statistics = domain.Statistics{}
		return m, nil
	}
	m.details.statistics = msg.statistics
	m.details.statisticsLoaded = true
	return m, nil
}

// startExactCount runs one whole-object count. The caller cancels with Esc;
// the result is dropped when the selected object or count generation changes.
func (m *Model) startExactCount() tea.Cmd {
	if m.session == nil || !m.details.has {
		return nil
	}
	node := m.details.node
	if node.Kind != domain.KindTable && node.Kind != domain.KindCollection {
		m.setMessage(levelInfo, "exact count needs a table or collection")
		return nil
	}
	d := &m.details
	if d.exactLoading {
		return nil
	}
	if d.exactCancel != nil {
		d.exactCancel()
	}
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	m.exactGen++
	gen := m.exactGen
	ref := node.Ref
	key := ref.Key()
	session := m.session
	d.exactCancel = cancel
	d.exactLoading = true
	d.exactErr = ""
	d.exactStarted = time.Now()
	m.setMessage(levelInfo, "counting exact rows… Esc cancels")
	return tea.Batch(
		func() tea.Msg {
			count, err := session.ExactCount(ctx, ref)
			return exactCountMsg{gen: gen, key: key, count: count, err: err}
		},
		tea.Tick(time.Second, func(time.Time) tea.Msg {
			return exactTickMsg{gen: gen}
		}),
	)
}

func (m *Model) cancelExactCount() {
	d := &m.details
	if d.exactCancel != nil {
		d.exactCancel()
		d.exactCancel = nil
	}
	m.exactGen++
	d.exactLoading = false
	d.exactErr = ""
	m.setMessage(levelInfo, "cancelled")
}

func (m Model) handleExactCount(msg exactCountMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.exactGen {
		return m, nil
	}
	if !m.details.has || m.details.node.Ref.Key() != msg.key {
		return m, nil
	}
	d := &m.details
	d.exactLoading = false
	if d.exactCancel != nil {
		d.exactCancel()
		d.exactCancel = nil
	}
	if msg.err != nil {
		if domain.KindOf(msg.err) == domain.KindCancelled || errors.Is(msg.err, context.Canceled) {
			m.setMessage(levelInfo, "cancelled")
			return m, nil
		}
		d.exactErr = oneLine(msg.err.Error())
		m.setMessage(levelError, d.exactErr)
		return m, nil
	}
	d.exactCount = msg.count
	d.exactSet = true
	d.exactErr = ""
	m.setMessage(levelSuccess, "exact count "+formatCount(msg.count))
	return m, nil
}

func (m Model) handleExactTick(msg exactTickMsg) (tea.Model, tea.Cmd) {
	if !m.details.exactLoading || msg.gen != m.exactGen {
		return m, nil
	}
	return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
		return exactTickMsg{gen: msg.gen}
	})
}
