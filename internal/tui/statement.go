package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/help"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

func (m *Model) openStatementModal() {
	if m.screen != ScreenWorkspace || m.session == nil {
		return
	}
	m.completion = completionState{}
	m.escChord = false
	m.statementOrigin = m.query.Value()
	m.statementInput.SetValue(m.statementOrigin)
	m.statementInput.SetCursor(0)
	m.statementInput.Focus()
	m.modal = modalStatement
}

func (m *Model) closeStatementModal() {
	m.statementInput.Blur()
	m.statementOrigin = ""
	m.modal = modalNone
	m.escChord = false
	m.completion = completionState{}
}

func (m Model) handleStatementKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.completion.active {
		if key == "tab" {
			m.acceptCompletion()
			return m, nil
		}
		if next, cmd, handled := m.handleCompletionKey(msg); handled {
			return next, cmd
		}
	}
	if m.escChord {
		m.escChord = false
		if key == "tab" {
			return m, m.openCompletion(true)
		}
		if key == "esc" {
			return m, m.armEscChord()
		}
		m.closeStatementModal()
		return m, nil
	}
	if action, ok := keymap.Lookup(key); ok {
		switch action {
		case keymap.Cancel:
			return m, m.armEscChord()
		case keymap.Confirm:
			statement := strings.TrimSpace(m.statementInput.Value())
			m.closeStatementModal()
			m.query.SetValue(statement)
			m.query.SetCursor(len([]rune(statement)))
			return m.startQuery()
		case keymap.Completion:
			return m, m.openCompletion(true)
		case keymap.ContextHelp:
			m.openContextHelp()
			return m, nil
		case keymap.Tab:
			return m, m.openCompletion(true)
		case keymap.ShiftTab:
			return m, nil
		}
	}
	updated, cmd := m.statementInput.Update(msg)
	m.statementInput = updated
	if m.completion.active {
		m.refreshCompletion()
	}
	return m, cmd
}

func (m Model) viewStatement() string {
	width := modalWidth(m.width, 70)
	var body []string
	body = append(body, m.styles.Accent.Render(m.statementPrompt())+m.statementInput.View())
	body = append(body, "", m.actionLine("Enter", "execute", "Esc", "cancel"))
	return modalBox("edit statement", body, width, m.styles)
}

func (m Model) statementPrompt() string {
	if m.profileForStatus().Engine == domain.EngineMongoDB {
		return "MQL> "
	}
	return "SQL> "
}

func (m *Model) openContextHelp() {
	m.completion = completionState{}
	term := ""
	switch {
	case m.modal == modalStatement:
		_, _, word := completionWord(m.statementInput.Value(), m.statementInput.Position(),
			m.profileForStatus().Engine == domain.EngineMongoDB)
		term = word
		if term == "" {
			term = "query"
		}
	case m.focus == FocusQuery:
		_, _, word := completionWord(m.query.Value(), m.query.Position(),
			m.profileForStatus().Engine == domain.EngineMongoDB)
		term = word
		if term == "" {
			term = "query"
		}
	case m.focus == FocusTree:
		if node, ok := m.tree.selectedNode(); ok {
			term = string(node.Kind)
		}
		if term == "" {
			term = "tree"
		}
	case m.focus == FocusMain:
		term = strings.ToLower(m.details.currentTab())
		if term == "" {
			term = "details"
		}
	default:
		term = "status"
	}
	text, ok := help.Lookup(term)
	if !ok {
		m.setMessage(levelInfo, "no help available")
		return
	}
	m.helpTerm = term
	m.helpBody = text
	m.helpExtended = false
	m.helpScroll = 0
	m.helpReturn = m.modal
	if m.helpReturn == modalHelp {
		m.helpReturn = modalNone
	}
	m.modal = modalHelp
}

func (m Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.closeHelp()
		return m, nil
	case keymap.Confirm, keymap.ContextHelp:
		m.showExtendedHelp()
		return m, nil
	case keymap.Up:
		if m.helpExtended {
			m.helpScroll--
		}
	case keymap.Down:
		if m.helpExtended {
			m.helpScroll++
		}
	case keymap.PageUp:
		if m.helpExtended {
			m.helpScroll -= m.helpPage()
		}
	case keymap.PageDown:
		if m.helpExtended {
			m.helpScroll += m.helpPage()
		}
	case keymap.Home:
		if m.helpExtended {
			m.helpScroll = 0
		}
	case keymap.End:
		if m.helpExtended {
			m.helpScroll = m.helpLineCount()
		}
	}
	return m, nil
}

func (m *Model) closeHelp() {
	m.modal = m.helpReturn
	m.helpReturn = modalNone
	m.helpTerm = ""
	m.helpBody = ""
	m.helpExtended = false
	m.helpScroll = 0
}

func (m *Model) showExtendedHelp() {
	if m.helpExtended {
		return
	}
	body, ok := help.Extended(m.helpTerm)
	if !ok {
		body = m.helpBody + "\n\nNo extended examples for this term."
	}
	m.helpBody = body
	m.helpExtended = true
	m.helpScroll = 0
}

func (m Model) helpWidth() int {
	return modalWidth(m.width, 72)
}

func (m Model) helpLines() []string {
	return wrapHelpText(m.helpBody, m.helpWidth()-4)
}

func (m Model) helpLineCount() int {
	return len(m.helpLines())
}

func (m Model) helpPage() int {
	page := m.height - 8
	if page < 4 {
		page = 4
	}
	return page
}

func (m Model) viewHelp() string {
	width := m.helpWidth()
	inner := width - 4
	lines := m.helpLines()
	page := m.helpPage()
	scroll := m.helpScroll
	if scroll > len(lines)-page {
		scroll = len(lines) - page
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + page
	if end > len(lines) {
		end = len(lines)
	}
	var body []string
	for _, line := range lines[scroll:end] {
		switch {
		case line == "Examples":
			body = append(body, m.styles.Header.Render(line))
		case strings.HasSuffix(line, ":") && len(line) < inner:
			body = append(body, m.styles.Label.Render(line))
		default:
			body = append(body, m.styles.Text.Render(line))
		}
	}
	if len(lines) > page {
		body = append(body, m.styles.Muted.Render(
			fmt.Sprintf("%d-%d of %d", scroll+1, end, len(lines))))
	}
	body = append(body, "")
	if m.helpExtended {
		if len(lines) > page {
			body = append(body, m.actionLine("Up/Down", "scroll", "Esc", "close"))
		} else {
			body = append(body, m.styles.Accent.Render("Esc")+" "+m.styles.Muted.Render("close"))
		}
	} else {
		body = append(body, m.actionLine("Enter", "examples", "Ctrl+F1", "examples"))
		body = append(body, m.styles.Muted.Render("Esc closes"))
	}
	title := "help · " + m.helpTerm
	if m.helpExtended {
		title += " · examples"
	}
	return modalBox(title, body, width, m.styles)
}

// wrapHelpText preserves explicit line breaks and indentation, which matters
// for code examples, while wrapping lines that exceed the modal width.
func wrapHelpText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	var lines []string
	for _, raw := range strings.Split(text, "\n") {
		if raw == "" {
			lines = append(lines, "")
			continue
		}
		if lipgloss.Width(raw) <= width {
			lines = append(lines, raw)
			continue
		}
		indent := raw[:len(raw)-len(strings.TrimLeft(raw, " "))]
		words := strings.Fields(strings.TrimSpace(raw))
		current := indent
		for _, word := range words {
			if current == indent {
				current += word
				continue
			}
			if lipgloss.Width(current)+1+lipgloss.Width(word) <= width {
				current += " " + word
				continue
			}
			lines = append(lines, current)
			current = indent + "  " + word
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}
