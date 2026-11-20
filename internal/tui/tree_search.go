package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

// treeSearchDoubleTap is the window for the second slash that inserts the
// selected node name instead of a literal slash or dot.
const treeSearchDoubleTap = 500 * time.Millisecond

func (m *Model) openTreeSearch() {
	if m.screen != ScreenWorkspace || m.focus != FocusTree {
		return
	}
	if len(m.tree.rows) == 0 && m.tree.loadingRoot {
		m.setMessage(levelInfo, "tree is still loading")
		return
	}
	m.tree.searchPrompt = true
	m.tree.searchOpened = time.Now()
	m.tree.searchCursor = len([]rune(m.tree.searchPattern))
}

func (m *Model) clearTreeSearch() {
	m.tree.searchPrompt = false
	m.tree.searchPattern = ""
	m.tree.searchCursor = 0
	m.tree.searchOpened = time.Time{}
}

func (m Model) treeSearchVisible() bool {
	return m.tree.searchPrompt || m.tree.searchPattern != ""
}

func (m Model) treeSearchMatches() []int {
	pattern := m.tree.searchPattern
	if pattern == "" {
		return nil
	}
	var matches []int
	for i, row := range m.tree.rows {
		if row.Info != "" {
			continue
		}
		if containsFold(treeRowSearchText(row), pattern, m.caseInsensitive) {
			matches = append(matches, i)
		}
	}
	return matches
}

func treeRowSearchText(row treeRow) string {
	parts := []string{oneLine(row.Node.Name)}
	if row.Node.Ref.Schema != "" {
		parts = append(parts, oneLine(row.Node.Ref.Schema+"."+row.Node.Name))
	}
	return strings.Join(parts, " ")
}

// treeSearchInsertValue inserts the selected node value at the search cursor.
// A short double slash inserts the plain name; Ins inserts the qualified
// schema.name form when a schema exists.
func (m *Model) treeSearchInsertValue(full bool) bool {
	node, ok := m.tree.selectedNode()
	if !ok {
		return false
	}
	value := oneLine(node.Name)
	if full && node.Ref.Schema != "" {
		value = oneLine(node.Ref.Schema) + "." + value
	}
	if value == "" {
		return false
	}
	m.tree.searchPattern, m.tree.searchCursor = insertFilterText(
		m.tree.searchPattern, m.tree.searchCursor, value)
	m.tree.searchOpened = time.Time{}
	return true
}

func (m *Model) nextTreeMatch(delta int) tea.Cmd {
	matches := m.treeSearchMatches()
	if len(matches) == 0 {
		m.setMessage(levelInfo, "no tree matches")
		return nil
	}
	current := 0
	if delta < 0 {
		current = len(matches) - 1
		if index := sort.Search(len(matches), func(i int) bool {
			return matches[i] > m.tree.selected
		}); index > 0 {
			current = index - 1
		}
		if current >= 0 && matches[current] == m.tree.selected {
			current--
		}
		if current < 0 {
			current = len(matches) - 1
		}
	} else {
		index := sort.Search(len(matches), func(i int) bool {
			return matches[i] >= m.tree.selected
		})
		if index < len(matches) && matches[index] == m.tree.selected {
			index++
		}
		if index >= len(matches) {
			index = 0
		}
		current = index
	}
	m.tree.selected = matches[current]
	m.setMessage(levelInfo, fmt.Sprintf("match %d of %d", current+1, len(matches)))
	return nil
}

func (m Model) handleTreeSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.tree.searchPrompt {
		return m, nil, false
	}
	m.message = ""
	m.messageLevel = levelNone
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Quit, keymap.ForceQuit:
			m.clearTreeSearch()
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.ChordX:
			return m, nil, false
		case keymap.Tab, keymap.ShiftTab, keymap.Disconnect:
			m.clearTreeSearch()
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.Help:
			if msg.String() != "f1" {
				break
			}
			m.clearTreeSearch()
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.Cancel:
			m.clearTreeSearch()
			return m, nil, true
		case keymap.Search:
			if time.Since(m.tree.searchOpened) <= treeSearchDoubleTap && m.treeSearchInsertValue(false) {
				return m, nil, true
			}
		case keymap.InsertSelected:
			if m.treeSearchInsertValue(true) {
				return m, nil, true
			}
		case keymap.Confirm:
			if m.tree.searchPattern == "" {
				m.clearTreeSearch()
				return m, nil, true
			}
			m.tree.searchPrompt = false
			matches := m.treeSearchMatches()
			if len(matches) == 0 {
				m.setMessage(levelInfo, "no tree matches")
				return m, nil, true
			}
			m.tree.selected = matches[0]
			m.setMessage(levelInfo, fmt.Sprintf("match 1 of %d", len(matches)))
			return m, nil, true
		}
	}
	m.tree.searchOpened = time.Time{}
	m.tree.searchPattern, m.tree.searchCursor = editFilterText(m.tree.searchPattern, m.tree.searchCursor, msg)
	if matches := m.treeSearchMatches(); len(matches) > 0 {
		m.tree.selected = matches[0]
	}
	return m, nil, true
}

func (m Model) renderTreeSearchBar(width int) string {
	pattern := filterText(m.tree.searchPattern)
	var value string
	if m.tree.searchPrompt {
		runes := []rune(pattern)
		cursor := m.tree.searchCursor
		if cursor < 0 {
			cursor = 0
		}
		if cursor > len(runes) {
			cursor = len(runes)
		}
		value = m.styles.Text.Render(string(runes[:cursor])) +
			m.styles.Accent.Render("▏") +
			m.styles.Text.Render(string(runes[cursor:]))
	} else {
		value = m.styles.Text.Render(pattern)
	}
	label := m.styles.Accent.Render("/")
	var count string
	matches := m.treeSearchMatches()
	switch {
	case m.tree.searchPattern == "":
		count = m.styles.Muted.Render("type to search · Enter keep · Esc clear")
	case len(matches) == 0:
		count = m.styles.Error.Render("0 matches · Esc clear")
	default:
		position := 0
		for i, row := range matches {
			if row == m.tree.selected {
				position = i + 1
				break
			}
		}
		if position == 0 {
			count = m.styles.Muted.Render(fmt.Sprintf("%d matches · n/N next · Esc clear", len(matches)))
		} else {
			count = m.styles.Muted.Render(fmt.Sprintf("match %d of %d · n/N next · Esc clear", position, len(matches)))
		}
	}
	return Truncate(label+" "+value+"  "+count, width)
}

func (m Model) renderTreeSearchLabel(row treeRow, base lipgloss.Style) string {
	label := treeRowLabel(row)
	if m.tree.searchPattern == "" {
		return base.Render(label)
	}
	return highlightMatches(label, m.tree.searchPattern, m.caseInsensitive, base, m.styles.Match)
}
