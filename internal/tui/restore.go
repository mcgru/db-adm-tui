package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

type restoreState struct {
	ref domain.ObjectRef
}

// savePlace records the selected object for the next connection. Grid cursor
// and scroll state are deliberately not saved.
func (m *Model) savePlace() {
	if m.svc == nil || m.session == nil || m.screen != ScreenWorkspace {
		return
	}
	node, ok := m.tree.selectedNode()
	if !ok || !node.IsObject() {
		return
	}
	ref := node.Ref
	_ = m.svc.SaveLastObject(context.Background(), m.active, &ref)
}

// startRestore begins walking the tree to the profile's saved object.
func (m *Model) startRestore() tea.Cmd {
	if m.restore != nil || m.session == nil || m.active.LastObject == nil {
		return nil
	}
	ref := *m.active.LastObject
	if ref.Database == "" || ref.Name == "" {
		return nil
	}
	m.restore = &restoreState{ref: ref}
	return m.advanceRestore()
}

func (m *Model) advanceRestore() tea.Cmd {
	restore := m.restore
	if restore == nil {
		return nil
	}
	ref := restore.ref
	dbKey, ok := m.findTreeChild(m.tree.roots, domain.KindDatabase, ref.Database)
	if !ok {
		m.restore = nil
		return nil
	}
	if cmd := m.expandRestoreNode(dbKey); cmd != nil {
		return cmd
	}
	parentKey := dbKey
	if m.engine() == domain.EnginePostgreSQL {
		schema := ref.Schema
		if schema == "" {
			schema = "public"
		}
		if schema == "public" {
			key, ok := m.findTreeChild(m.childrenOf(dbKey), domain.KindSchema, "public")
			if !ok {
				m.restore = nil
				return nil
			}
			parentKey = key
		} else {
			groupKey, ok := m.findTreeChild(m.childrenOf(dbKey), domain.KindSchemas, "schemas")
			if !ok {
				m.restore = nil
				return nil
			}
			if cmd := m.expandRestoreNode(groupKey); cmd != nil {
				return cmd
			}
			key, ok := m.findTreeChild(m.childrenOf(groupKey), domain.KindSchema, schema)
			if !ok {
				m.restore = nil
				return nil
			}
			parentKey = key
		}
	}
	if cmd := m.expandRestoreNode(parentKey); cmd != nil {
		return cmd
	}
	objectKey, ok := m.findTreeChild(m.childrenOf(parentKey), ref.Kind, ref.Name)
	if !ok {
		m.restore = nil
		return nil
	}
	m.tree.selectKey(objectKey)
	m.restore = nil
	m.setMessage(levelInfo, "restored "+oneLine(ref.Name))
	return nil
}

func (m *Model) expandRestoreNode(key string) tea.Cmd {
	node, ok := m.tree.node(key)
	if !ok {
		m.restore = nil
		return nil
	}
	if !node.Loaded && !node.Loading {
		if !node.Node.Expandable {
			m.restore = nil
			return nil
		}
		m.tree.setLoading(key, true)
		return m.loadChildrenCmd(node.Node)
	}
	if node.Loading {
		return nil
	}
	if !node.Expanded {
		m.tree.expand(key)
	}
	return nil
}

func (m Model) childrenOf(key string) []string {
	if node, ok := m.tree.node(key); ok {
		return node.Children
	}
	return nil
}

func (m Model) findTreeChild(keys []string, kind domain.ObjectKind, name string) (string, bool) {
	for _, key := range keys {
		node, ok := m.tree.node(key)
		if !ok {
			continue
		}
		if node.Node.Kind == kind && node.Node.Name == name {
			return key, true
		}
	}
	return "", false
}
