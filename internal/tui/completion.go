package tui

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/help"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

const completionPage = 8

type completionState struct {
	active   bool
	modal    bool
	items    []string
	selected int
	offset   int
	start    int
	end      int
}

func (m *Model) openCompletion(modal bool) tea.Cmd {
	if modal && m.modal != modalStatement {
		return nil
	}
	if !modal && m.focus != FocusQuery {
		return nil
	}
	m.completion = completionState{modal: modal}
	m.refreshCompletion()
	if !m.completion.active {
		m.setMessage(levelInfo, "no suggestions")
	}
	return nil
}

func (m *Model) refreshCompletion() {
	value, pos := m.query.Value(), m.query.Position()
	if m.completion.modal {
		value, pos = m.statementInput.Value(), m.statementInput.Position()
	}
	start, end, word := completionWord(value, pos, m.profileForStatus().Engine == domain.EngineMongoDB)
	prefix := word
	if start <= pos && pos < end {
		prefix = string([]rune(value)[start:pos])
		if prefix == "" {
			prefix = word
		}
	}
	seen := map[string]bool{}
	var items []string
	add := func(term string) {
		term = oneLine(term)
		if term == "" || seen[term] {
			return
		}
		if !strings.HasPrefix(strings.ToLower(term), strings.ToLower(prefix)) {
			return
		}
		seen[term] = true
		items = append(items, term)
	}
	for _, term := range help.CompletionTerms(m.profileForStatus().Engine) {
		add(term)
	}
	for _, term := range m.completionIdentifiers() {
		add(term)
	}
	if len(items) == 0 {
		m.completion = completionState{}
		return
	}
	m.completion.active = true
	m.completion.items = items
	m.completion.start = start
	m.completion.end = end
	if m.completion.selected >= len(items) {
		m.completion.selected = len(items) - 1
	}
	if m.completion.selected < 0 {
		m.completion.selected = 0
	}
	m.completion.offset = completionWindowStart(m.completion.offset, m.completion.selected, len(items))
}

func (m Model) completionIdentifiers() []string {
	seen := map[string]bool{}
	var items []string
	add := func(term string) {
		term = oneLine(term)
		if term == "" || seen[term] {
			return
		}
		seen[term] = true
		items = append(items, term)
	}
	databaseNames := make([]string, 0)
	for _, root := range m.tree.roots {
		if node, ok := m.tree.nodes[root]; ok {
			databaseNames = append(databaseNames, node.Node.Name)
			add(node.Node.Name)
		}
	}
	sort.Strings(databaseNames)
	names := make([]string, 0, len(m.tree.nodes))
	for _, node := range m.tree.nodes {
		names = append(names, node.Node.Name)
	}
	sort.Strings(names)
	for _, name := range names {
		add(name)
	}
	for _, node := range m.tree.nodes {
		switch node.Node.Kind {
		case domain.KindTable, domain.KindCollection, domain.KindView:
			if node.Node.Ref.Schema != "" {
				add(node.Node.Ref.Schema + "." + node.Node.Name)
			} else if node.Node.Ref.Database != "" {
				add(node.Node.Ref.Database + "." + node.Node.Name)
			}
		}
	}
	for _, column := range m.data.columns {
		add(column.Name)
	}
	for _, field := range m.details.structure.Fields {
		add(field.Name)
	}
	for _, statement := range m.history {
		for _, word := range identifierWords(statement) {
			add(word)
		}
	}
	return items
}

// identifierWords extracts identifier-like tokens from an executed statement.
func identifierWords(statement string) []string {
	return strings.FieldsFunc(statement, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '$')
	})
}

// completionWord returns the identifier rune span under or beside the cursor.
// It also recognizes a MQL key immediately before ':' so Ctrl+Space after
// "$o:" offers "$or". start and end are the replacement range in runes.
func completionWord(value string, pos int, mongoColon bool) (int, int, string) {
	runes := []rune(value)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	start, end := pos, pos
	switch {
	case pos > 0 && completionRune(runes[pos-1]):
		start = pos - 1
	case pos < len(runes) && completionRune(runes[pos]):
		start = pos
	default:
		if !mongoColon || pos == 0 || runes[pos-1] != ':' {
			return pos, pos, ""
		}
		end = pos - 1
		index := end - 1
		for index >= 0 && unicode.IsSpace(runes[index]) {
			index--
		}
		if index < 0 || !completionRune(runes[index]) {
			return pos, pos, ""
		}
		start = index
		for start > 0 && completionRune(runes[start-1]) {
			start--
		}
		return start, end, string(runes[start:end])
	}
	for start > 0 && completionRune(runes[start-1]) {
		start--
	}
	if end < len(runes) {
		for end < len(runes) && completionRune(runes[end]) {
			end++
		}
	} else {
		end = pos
	}
	return start, end, string(runes[start:end])
}

func completionRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '$'
}

func completionWindowStart(start, selected, total int) int {
	page := completionPage
	if total <= page {
		return 0
	}
	if start < 0 {
		start = 0
	}
	if selected < start {
		start = selected
	}
	if selected >= start+page {
		start = selected - page + 1
	}
	if start > total-page {
		start = total - page
	}
	return start
}

func (m Model) handleCompletionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.completion.active {
		return m, nil, false
	}
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil, false
	}
	switch action {
	case keymap.Up:
		m.completion.selected--
		if m.completion.selected < 0 {
			m.completion.selected = len(m.completion.items) - 1
		}
	case keymap.Down:
		m.completion.selected++
		if m.completion.selected >= len(m.completion.items) {
			m.completion.selected = 0
		}
	case keymap.PageUp:
		m.completion.selected -= completionPage
		if m.completion.selected < 0 {
			m.completion.selected = 0
		}
	case keymap.PageDown:
		m.completion.selected += completionPage
		if m.completion.selected >= len(m.completion.items) {
			m.completion.selected = len(m.completion.items) - 1
		}
	case keymap.Confirm:
		m.acceptCompletion()
		return m, nil, true
	case keymap.Cancel:
		m.completion = completionState{}
		return m, nil, true
	case keymap.Completion:
		m.refreshCompletion()
		return m, nil, true
	default:
		return m, nil, false
	}
	m.completion.offset = completionWindowStart(m.completion.offset, m.completion.selected, len(m.completion.items))
	return m, nil, true
}

func (m *Model) acceptCompletion() {
	state := m.completion
	if !state.active || len(state.items) == 0 {
		return
	}
	item := state.items[state.selected]
	input := &m.query
	if state.modal {
		input = &m.statementInput
	}
	value, cursor := completionText(input.Value(), state.start, state.end, item, m.profileForStatus().Engine)
	input.SetValue(value)
	input.SetCursor(cursor)
	m.completion = completionState{}
}

// completionText replaces the word span with the chosen term and may add the
// surrounding syntax the term expects: ": [  ]" for Mongo array operators and
// "()" for functions, unless the text after the word already has it.
func completionText(value string, start, end int, item string, engine domain.Engine) (string, int) {
	runes := []rune(value)
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	replaceOnly := func() (string, int) {
		next := append([]rune(nil), runes[:start]...)
		next = append(next, []rune(item)...)
		next = append(next, runes[end:]...)
		return string(next), start + len([]rune(item))
	}
	after := runes[end:]
	if engine == domain.EngineMongoDB && help.MQLArrayTerm(item) {
		index := 0
		for index < len(after) && unicode.IsSpace(after[index]) {
			index++
		}
		if index < len(after) && after[index] == ':' {
			rest := index + 1
			for rest < len(after) && unicode.IsSpace(after[rest]) {
				rest++
			}
			if rest < len(after) && after[rest] == '[' {
				return replaceOnly()
			}
			if rest == len(after) || after[rest] == ']' || after[rest] == '}' {
				insertAt := end + rest
				next := append([]rune(nil), runes[:start]...)
				next = append(next, []rune(item)...)
				next = append(next, runes[end:insertAt]...)
				next = append(next, []rune("[  ]")...)
				next = append(next, runes[insertAt:]...)
				cursor := start + len([]rune(item)) + (insertAt - end) + 2
				return string(next), cursor
			}
			return replaceOnly()
		}
		next := append([]rune(nil), runes[:start]...)
		next = append(next, []rune(item)...)
		next = append(next, []rune(": [  ]")...)
		next = append(next, runes[end:]...)
		return string(next), start + len([]rune(item)) + 4
	}
	if help.FunctionTerm(item) {
		index := 0
		for index < len(after) && unicode.IsSpace(after[index]) {
			index++
		}
		if index < len(after) && after[index] == '(' {
			return replaceOnly()
		}
		next := append([]rune(nil), runes[:start]...)
		next = append(next, []rune(item)...)
		next = append(next, '(', ')')
		next = append(next, runes[end:]...)
		return string(next), start + len([]rune(item)) + 1
	}
	return replaceOnly()
}

func (m Model) completionLines() []string {
	if !m.completion.active || len(m.completion.items) == 0 {
		return nil
	}
	start := m.completion.offset
	end := start + completionPage
	if end > len(m.completion.items) {
		end = len(m.completion.items)
	}
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		item := oneLine(m.completion.items[i])
		if i == m.completion.selected {
			lines = append(lines, m.styles.Select.Render(Pad(item, 24)))
			continue
		}
		lines = append(lines, m.styles.Text.Render(item))
	}
	return lines
}

func (m Model) completionTitle() string {
	total := len(m.completion.items)
	if total == 0 {
		return "completion"
	}
	end := m.completion.offset + completionPage
	if end > total {
		end = total
	}
	return "completion " + strconv.Itoa(m.completion.offset+1) + "–" +
		strconv.Itoa(end) + " of " + strconv.Itoa(total)
}

func (m Model) completionPopup() string {
	lines := m.completionLines()
	if len(lines) == 0 {
		return ""
	}
	width := 28
	for _, line := range lines {
		if w := len(StripANSI(line)); w+4 > width {
			width = w + 4
		}
	}
	width = modalWidth(m.width, width)
	body := append([]string{}, lines...)
	body = append(body, m.styles.Muted.Render("Enter accept · Esc close"))
	return modalBox(m.completionTitle(), body, width, m.styles)
}

// completionAnchor returns the popup's top-left cell. The popup sits above the
// input line and its left edge follows the cursor column.
func (m Model) completionAnchor(popupWidth, popupHeight int) (int, int) {
	if m.modal == modalStatement {
		return m.statementCompletionAnchor(popupWidth, popupHeight)
	}
	line, column := m.queryCursor(m.width)
	x := 8 + column
	regions := m.regions()
	queryTop := 1 + regions.BodyHeight
	y := queryTop + line - popupHeight
	return x, y
}

func (m Model) statementCompletionAnchor(popupWidth, popupHeight int) (int, int) {
	modal := m.viewStatement()
	lines := strings.Split(modal, "\n")
	modalWidth := 0
	for _, line := range lines {
		if width := lipgloss.Width(line); width > modalWidth {
			modalWidth = width
		}
	}
	modalX := (m.width - modalWidth) / 2
	modalY := (m.height - len(lines)) / 2
	runes := []rune(m.statementInput.Value())
	pos := m.statementInput.Position()
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	prefixWidth := lipgloss.Width(string(runes[:pos]))
	inputWidth := modalWidth - 16
	if inputWidth < 1 {
		inputWidth = 1
	}
	if prefixWidth >= inputWidth {
		prefixWidth = inputWidth - 1
	}
	x := modalX + 14 + prefixWidth
	y := modalY + 1 - popupHeight
	return x, y
}
