package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

type wrappedQuery struct {
	lines      []string
	cursorLine int
	cursorCol  int
}

// wrapQueryValue hard-wraps the logical statement at width while keeping the
// cursor mapped to the wrapped line and column.
func wrapQueryValue(value string, width, cursor int) wrappedQuery {
	if width < 1 {
		width = 1
	}
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	result := wrappedQuery{}
	lineStart := 0
	col := 0
	for i := range runes {
		if col == width {
			result.lines = append(result.lines, string(runes[lineStart:i]))
			lineStart = i
			col = 0
		}
		if i == cursor {
			result.cursorLine = len(result.lines)
			result.cursorCol = col
		}
		col++
	}
	if lineStart < len(runes) || len(result.lines) == 0 {
		result.lines = append(result.lines, string(runes[lineStart:]))
	}
	if cursor == len(runes) {
		result.cursorLine = len(result.lines) - 1
		result.cursorCol = len([]rune(result.lines[len(result.lines)-1]))
	}
	return result
}

func queryInputWidth(width int) int {
	width -= 8 // two-cell marker + "SQL/MQL ›" + one space
	if width < 1 {
		return 1
	}
	return width
}

func (m Model) queryCursor(width int) (int, int) {
	if strings.TrimSpace(m.query.Value()) == "" {
		return 0, 0
	}
	wrapped := wrapQueryValue(m.query.Value(), queryInputWidth(width), m.query.Position())
	return wrapped.cursorLine, wrapped.cursorCol
}

func (m Model) viewQuery(width int) string {
	return m.viewQueryHeight(width, layout.ClampQueryHeight(m.queryHeight))
}

func (m Model) viewQueryHeight(width, height int) string {
	height = layout.ClampQueryHeight(height)
	engine := m.profileForStatus().Engine
	label := "SQL"
	if engine == domain.EngineMongoDB {
		label = "MQL"
	}
	marker := "  "
	promptStyle := m.styles.ModeRO
	valueStyle := m.styles.Text
	if m.focus == FocusQuery {
		marker = m.styles.Focused.Render("▸ ")
		promptStyle = m.styles.Focused
	}
	prompt := promptStyle.Render(label) + " " + m.styles.Accent.Render("›")
	firstPrefix := marker + prompt + " "
	indent := strings.Repeat(" ", 8)
	available := queryInputWidth(width)

	if ok, hint := m.queryAvailability(); !ok && strings.TrimSpace(m.query.Value()) == "" {
		first := Pad(firstPrefix+m.styles.Muted.Render(Truncate(hint, available)), width)
		return joinQueryLines(first, indent, height, width)
	}
	if height <= 1 {
		return m.viewQuerySingle(width, marker, prompt, valueStyle)
	}

	value := m.query.Value()
	if strings.TrimSpace(value) == "" {
		placeholder := queryPlaceholder(engine, m.currentQueryRef())
		line := Truncate(placeholder, available)
		var rendered string
		if m.focus == FocusQuery {
			cursorChar := " "
			rest := line
			if line != "" {
				runes := []rune(line)
				cursorChar = string(runes[0])
				rest = string(runes[1:])
			}
			rendered = m.styles.Cursor.Render(cursorChar) + m.styles.Muted.Render(rest)
		} else {
			rendered = m.styles.Muted.Render(line)
		}
		first := Pad(firstPrefix+rendered, width)
		return joinQueryLines(first, indent, height, width)
	}

	wrapped := wrapQueryValue(value, available, m.query.Position())
	start := 0
	if wrapped.cursorLine >= height {
		start = wrapped.cursorLine - height + 1
	}
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		index := start + i
		text := ""
		if index < len(wrapped.lines) {
			text = wrapped.lines[index]
		}
		prefix := indent
		if i == 0 {
			prefix = firstPrefix
		}
		var rendered string
		switch {
		case m.queryRunning:
			rendered = m.styles.Muted.Render(Truncate(text, available))
		case m.focus == FocusQuery && index == wrapped.cursorLine:
			rendered = m.renderQueryCursor(text, wrapped.cursorCol)
		case m.focus == FocusQuery:
			rendered = valueStyle.Render(Truncate(text, available))
		default:
			rendered = valueStyle.Render(Truncate(text, available))
		}
		lines = append(lines, Pad(prefix+rendered, width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewQuerySingle(width int, marker, prompt string, valueStyle lipgloss.Style) string {
	engine := m.profileForStatus().Engine
	if available, hint := m.queryAvailability(); !available && strings.TrimSpace(m.query.Value()) == "" {
		return Pad(marker+prompt+" "+m.styles.Muted.Render(Truncate(hint, width-10)), width)
	}
	var value string
	switch {
	case m.queryRunning:
		value = m.styles.Muted.Render(Truncate(oneLine(m.query.Value()), width-10))
	case m.focus == FocusQuery:
		value = m.query.View()
	case strings.TrimSpace(m.query.Value()) == "":
		value = m.styles.Muted.Render(Truncate(queryPlaceholder(engine, m.currentQueryRef()), width-10))
	default:
		value = valueStyle.Render(Truncate(oneLine(m.query.Value()), width-10))
	}
	return Pad(marker+prompt+" "+value, width)
}

func (m Model) renderQueryCursor(text string, col int) string {
	runes := []rune(text)
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	cursorChar := " "
	after := ""
	if col < len(runes) {
		cursorChar = string(runes[col])
		after = string(runes[col+1:])
	}
	return m.styles.Text.Render(before) +
		m.styles.Cursor.Render(cursorChar) +
		m.styles.Text.Render(after)
}

func joinQueryLines(first, indent string, height, width int) string {
	lines := []string{first}
	for len(lines) < height {
		lines = append(lines, Pad(indent, width))
	}
	return strings.Join(lines, "\n")
}
