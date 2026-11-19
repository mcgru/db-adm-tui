package tui

import (
	"context"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

func (m *Model) resizeTree(delta int) {
	if m.screen != ScreenWorkspace {
		return
	}
	current := layout.ClampTreeWidth(m.width, m.treeWidth)
	next := layout.ClampTreeWidth(m.width, current+delta)
	if next == current && m.treeWidth == next {
		return
	}
	m.treeWidth = next
	m.persistLayout()
}

func (m *Model) resizeQuery(delta int) {
	if m.screen != ScreenWorkspace {
		return
	}
	current := layout.ClampQueryHeight(m.queryHeight)
	next := layout.ClampQueryHeight(current + delta)
	if next == current && m.queryHeight == next {
		return
	}
	m.queryHeight = next
	m.persistLayout()
}

func (m *Model) persistLayout() {
	if m.svc == nil || m.layoutKey == "" {
		return
	}
	m.svc.SaveLayout(context.Background(), m.layoutKey, domain.LayoutSettings{
		TreeWidth:   m.treeWidth,
		QueryHeight: m.queryHeight,
	})
}
