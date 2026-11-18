package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

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
		return m.ensureStructure()
	case "Indexes":
		return m.ensureIndexes()
	}
	return nil
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
