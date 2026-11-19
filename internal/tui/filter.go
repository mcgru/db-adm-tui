package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

// dataPrompt identifies the one inline filter bar. Buffer and server prompts
// are mutually exclusive.
type dataPrompt int

const (
	promptNone dataPrompt = iota
	promptBuffer
	promptServer
)

// filterBarVisible reports whether the Data tab spends a line on the filter
// bar. A buffer filter keeps its bar visible after the prompt closes.
func (d dataState) filterBarVisible() bool {
	return d.prompt != promptNone || d.bufferPattern != ""
}

// filterNoMatchVisible reports the separate no-match line under the bar.
func (d dataState) filterNoMatchVisible() bool {
	return d.bufferPattern != "" && len(d.matchCells) == 0 && !d.tooLarge
}

// recomputeMatches refreshes the raw-row offsets that survive the literal
// buffer pattern. It runs inside recount so every load, merge, and eviction
// keeps the projection honest.
// filterMatch is one matching cell: a raw-row offset and a rendered-column
// index. A row with two matching cells yields two entries.
type filterMatch struct {
	row int
	col int
}

func (d *dataState) recomputeMatches() {
	d.matches = d.matches[:0]
	d.matchCells = d.matchCells[:0]
	if d.bufferPattern == "" {
		return
	}
	for i, row := range d.rows {
		matchedRow := false
		for col, viewColumn := range d.view {
			text := Truncate(oneLine(cellForColumn(row, viewColumn).Display()), viewColumn.width)
			if !containsFold(text, d.bufferPattern, d.bufferCaseInsensitive) {
				continue
			}
			d.matchCells = append(d.matchCells, filterMatch{row: i, col: col})
			matchedRow = true
		}
		if matchedRow {
			d.matches = append(d.matches, i)
		}
	}
}

// focusFirstMatch keeps the cursor on a highlighted cell after a recompute.
func (d *dataState) focusFirstMatch() {
	if d.bufferPattern == "" || len(d.matchCells) == 0 {
		return
	}
	row := d.cursorRow - d.windowStart
	if d.matchCellIndex(row, d.cursorCol) >= 0 {
		return
	}
	d.cursorRow = d.windowStart + d.matchCells[0].row
	d.cursorCol = d.matchCells[0].col
}

func (d dataState) matchCellIndex(row, col int) int {
	for i, match := range d.matchCells {
		if match.row == row && match.col == col {
			return i
		}
	}
	return -1
}

// matchCellAtOrAfter returns the first match at or after (row, col) in
// row-major order, or len(matchCells) when none remains.
func (d dataState) matchCellAtOrAfter(row, col int) int {
	return sort.Search(len(d.matchCells), func(i int) bool {
		match := d.matchCells[i]
		return match.row > row || (match.row == row && match.col >= col)
	})
}

// matchCellAtOrBefore returns the last match at or before (row, col), or -1.
func (d dataState) matchCellAtOrBefore(row, col int) int {
	return sort.Search(len(d.matchCells), func(i int) bool {
		match := d.matchCells[i]
		return match.row > row || (match.row == row && match.col > col)
	}) - 1
}

// matchNumber is the one-based position of the cursor's match, or the next one
// when the cursor sits between matches.
func (d dataState) matchNumber() (int, int) {
	count := len(d.matchCells)
	if count == 0 {
		return 0, 0
	}
	index := d.matchCellIndex(d.cursorRow-d.windowStart, d.cursorCol)
	if index < 0 {
		index = d.matchCellAtOrAfter(d.cursorRow-d.windowStart, d.cursorCol)
		if index >= count {
			index = 0
		}
	}
	return index + 1, count
}

func containsFold(text, pattern string, insensitive bool) bool {
	return len(matchRanges(text, pattern, insensitive)) > 0
}

// matchRanges returns non-overlapping rune ranges of pattern inside text.
func matchRanges(text, pattern string, insensitive bool) [][2]int {
	if pattern == "" {
		return nil
	}
	runes := []rune(text)
	needle := []rune(pattern)
	if len(needle) > len(runes) {
		return nil
	}
	var ranges [][2]int
	for i := 0; i+len(needle) <= len(runes); {
		if runesEqualFold(runes[i:i+len(needle)], needle, insensitive) {
			ranges = append(ranges, [2]int{i, i + len(needle)})
			i += len(needle)
			continue
		}
		i++
	}
	return ranges
}

func runesEqualFold(haystack, needle []rune, insensitive bool) bool {
	if len(haystack) != len(needle) {
		return false
	}
	if insensitive {
		return strings.EqualFold(string(haystack), string(needle))
	}
	return string(haystack) == string(needle)
}

// highlightMatches paints every literal match with the Match style while
// keeping the cell's base style for the rest of the text.
func highlightMatches(text, pattern string, insensitive bool, base, match lipgloss.Style) string {
	ranges := matchRanges(text, pattern, insensitive)
	if len(ranges) == 0 {
		return base.Render(text)
	}
	runes := []rune(text)
	var b strings.Builder
	last := 0
	for _, span := range ranges {
		if span[0] > last {
			b.WriteString(base.Render(string(runes[last:span[0]])))
		}
		b.WriteString(match.Render(string(runes[span[0]:span[1]])))
		last = span[1]
	}
	if last < len(runes) {
		b.WriteString(base.Render(string(runes[last:])))
	}
	return b.String()
}

// openBufferFilter starts `/` over the loaded rows. It never queries.
func (m *Model) openBufferFilter() tea.Cmd {
	if m.screen != ScreenWorkspace || !m.dataTabActive() {
		m.setMessage(levelInfo, "/ filters rows in the Data tab")
		return nil
	}
	if !m.dataFocused() {
		m.setMessage(levelInfo, "focus the data view to filter")
		return nil
	}
	if !m.data.loaded {
		m.setMessage(levelInfo, "load rows before filtering")
		return nil
	}
	cmd := m.clearServerFilter()
	d := &m.data
	d.err = ""
	d.prompt = promptBuffer
	d.bufferCaseInsensitive = m.caseInsensitive
	d.bufferCursor = len([]rune(d.bufferPattern))
	return cmd
}

// openServerFilter starts Ctrl+F on the focused column or field. SQL targets
// the base column even when a JSON subfield holds the cursor; MongoDB keeps the
// dotted path.
func (m *Model) openServerFilter() tea.Cmd {
	if m.screen != ScreenWorkspace || !m.dataTabActive() {
		m.setMessage(levelInfo, "Ctrl+F filters a data column or field")
		return nil
	}
	if !m.dataFocused() {
		m.setMessage(levelInfo, "focus the data view to filter")
		return nil
	}
	if !m.data.loaded {
		m.setMessage(levelInfo, "load rows before filtering")
		return nil
	}
	column, ok := m.focusedDataColumn()
	if !ok {
		m.setMessage(levelInfo, "no focused column to filter")
		return nil
	}
	field, ok := m.filterField(column)
	if !ok {
		m.data.err = "the focused field cannot be filtered"
		m.setMessage(levelError, m.data.err)
		return nil
	}
	m.clearLocalFilter()
	m.data.err = ""
	d := &m.data
	d.prompt = promptServer
	d.serverField = field
	if d.serverFilter != nil && d.serverFilter.Field == field {
		d.serverPattern = d.serverFilter.Pattern
	} else {
		d.serverPattern = ""
	}
	d.serverCursor = len([]rune(d.serverPattern))
	return nil
}

func (m Model) focusedDataColumn() (dataColumn, bool) {
	d := m.data
	if d.cursorCol < 0 || d.cursorCol >= len(d.view) {
		return dataColumn{}, false
	}
	return d.view[d.cursorCol], true
}

// filterField maps a rendered column to the engine-facing field.
func (m Model) filterField(column dataColumn) (string, bool) {
	if column.index < 0 || column.index >= len(m.data.columns) {
		return "", false
	}
	field := m.data.columns[column.index].Name
	if m.engine() == domain.EngineMongoDB {
		field = column.name
		if !domain.ValidMongoPath(field) {
			return "", false
		}
	}
	if strings.TrimSpace(field) == "" {
		return "", false
	}
	return field, true
}

// handleFilterKey owns every key while the filter bar is open. Text editing
// wins, so paging gestures cannot start a fetch from an open prompt.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	d := &m.data
	if d.prompt == promptNone {
		return m, nil, false
	}
	m.message = ""
	m.messageLevel = levelNone
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Quit, keymap.ForceQuit:
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.ChordX:
			return m, nil, false
		case keymap.Tab, keymap.ShiftTab, keymap.Disconnect:
			m.closePromptForNavigation()
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.Help:
			if msg.String() != "f1" {
				break
			}
			m.closePromptForNavigation()
			next, cmd := m.apply(action)
			return next, cmd, true
		case keymap.Cancel:
			if d.prompt == promptBuffer {
				if d.loading {
					m.cancelDataFetch()
				}
				m.clearLocalFilter()
				return m, nil, true
			}
			if d.loading {
				m.cancelDataFetch()
				m.setMessage(levelInfo, "cancelled")
				return m, nil, true
			}
			d.prompt = promptNone
			return m, nil, true
		case keymap.Confirm:
			if d.prompt == promptBuffer {
				m.applyBufferPrompt()
				return m, nil, true
			}
			return m, m.applyServerFilter(), true
		}
	}
	if d.prompt == promptBuffer {
		d.bufferPattern, d.bufferCursor = editFilterText(d.bufferPattern, d.bufferCursor, msg)
		m.updateBufferFilter()
		return m, nil, true
	}
	d.serverPattern, d.serverCursor = editFilterText(d.serverPattern, d.serverCursor, msg)
	return m, nil, true
}

// closePromptForNavigation releases the bar so global focus and disconnect keys
// can run. A buffer pattern is kept; a server draft is cancelled because its
// fetch cannot share the key's command path.
func (m *Model) closePromptForNavigation() {
	if m.data.prompt == promptBuffer {
		m.applyBufferPrompt()
		return
	}
	m.data.prompt = promptNone
}

// editFilterText applies one input key to a single-line pattern.
func editFilterText(value string, cursor int, msg tea.KeyPressMsg) (string, int) {
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	switch msg.String() {
	case "left":
		if cursor > 0 {
			cursor--
		}
		return string(runes), cursor
	case "right":
		if cursor < len(runes) {
			cursor++
		}
		return string(runes), cursor
	case "home":
		return string(runes), 0
	case "end":
		return string(runes), len(runes)
	case "backspace":
		if cursor > 0 {
			runes = append(runes[:cursor-1], runes[cursor:]...)
			cursor--
		}
		return string(runes), cursor
	case "delete":
		if cursor < len(runes) {
			runes = append(runes[:cursor], runes[cursor+1:]...)
		}
		return string(runes), cursor
	}
	if msg.Text == "" {
		return string(runes), cursor
	}
	return insertFilterText(string(runes), cursor, msg.Text)
}

func insertFilterText(value string, cursor int, text string) (string, int) {
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	for _, r := range Sanitize(text) {
		if unicode.IsControl(r) {
			continue
		}
		runes = append(runes, 0)
		copy(runes[cursor+1:], runes[cursor:])
		runes[cursor] = r
		cursor++
	}
	return string(runes), cursor
}

// updateBufferFilter recomputes the literal projection as the pattern changes.
func (m *Model) updateBufferFilter() {
	d := &m.data
	d.bufferCaseInsensitive = m.caseInsensitive
	d.recount()
	m.ensureDataViewport()
	m.ensureDataColumn()
}

// applyBufferPrompt closes the bar; Enter keeps the filter, Esc clears it. An
// empty pattern clears the projection while leaving every loaded row.
func (m *Model) applyBufferPrompt() {
	d := &m.data
	d.prompt = promptNone
	if d.bufferPattern == "" {
		m.clearLocalFilter()
		return
	}
	m.updateBufferFilter()
}

// clearLocalFilter drops the buffer projection and restores all loaded rows.
func (m *Model) clearLocalFilter() {
	d := &m.data
	if d.prompt == promptBuffer {
		d.prompt = promptNone
	}
	d.bufferPattern = ""
	d.bufferCursor = 0
	d.matches = nil
	d.recount()
	m.ensureDataViewport()
}

// applyServerFilter starts the parameterized regexp fetch. The unfiltered
// window is cached by value so Esc can restore it without a query.
func (m *Model) applyServerFilter() tea.Cmd {
	d := &m.data
	if strings.TrimSpace(d.serverField) == "" {
		return nil
	}
	pattern := d.serverPattern
	if pattern == "" {
		d.prompt = promptNone
		if d.serverFilter != nil {
			return m.clearServerFilter()
		}
		return nil
	}
	if d.loading {
		m.setMessage(levelInfo, "wait for the current fetch to finish")
		return nil
	}
	m.clearLocalFilter()
	if d.serverFilter == nil {
		cached := *d
		cached.serverCached = nil
		cached.serverFilter = nil
		cached.prompt = promptNone
		cached.filterApplying = false
		cached.filterRestore = nil
		cached.filterClearing = false
		cached.caseApplying = false
		cached.matches = nil
		cached.matchCells = nil
		cached.bufferPattern = ""
		cached.bufferCursor = 0
		cached.loading = false
		cached.cancel = nil
		cached.err = ""
		d.serverCached = &cached
		d.filterRestore = nil
	} else {
		d.filterRestore = d.serverFilter
	}
	d.serverFilter = &domain.DataFilter{
		Field:           d.serverField,
		Pattern:         pattern,
		CaseInsensitive: m.caseInsensitive,
	}
	d.filterApplying = true
	d.prompt = promptNone
	d.batch = dataPageSize
	d.err = ""
	return m.beginDataFetch(0, dataPageSize, dataInitial)
}

// clearServerFilter returns to the cached unfiltered view. When no cache
// exists it reloads the first page; a failed reload keeps the filtered rows.
func (m *Model) clearServerFilter() tea.Cmd {
	d := &m.data
	if d.serverFilter == nil {
		return nil
	}
	if d.serverCached != nil {
		if d.cancel != nil {
			d.cancel()
		}
		m.dataGen++
		cached := *d.serverCached
		cached.serverCached = nil
		cached.serverFilter = nil
		cached.prompt = promptNone
		cached.filterApplying = false
		cached.filterRestore = nil
		cached.filterClearing = false
		cached.caseApplying = false
		cached.caseRestore = false
		cached.matches = nil
		cached.matchCells = nil
		cached.bufferPattern = ""
		cached.bufferCursor = 0
		cached.loading = false
		cached.cancel = nil
		cached.err = ""
		m.data = cached
		m.ensureDataViewport()
		return nil
	}
	d.filterClearing = true
	d.filterRestore = d.serverFilter
	d.serverFilter = nil
	d.prompt = promptNone
	return m.beginDataFetch(0, d.batch, dataInitial)
}

// revertFilterOnError restores the pre-apply filter after a failed or
// cancelled apply/clear attempt while keeping the previous rows.
func (m *Model) revertFilterOnError() {
	d := &m.data
	if d.filterApplying {
		d.serverFilter = d.filterRestore
		d.filterRestore = nil
		d.filterApplying = false
		d.prompt = promptServer
	}
	if d.filterClearing {
		d.serverFilter = d.filterRestore
		d.filterRestore = nil
		d.filterClearing = false
	}
	if d.caseApplying {
		m.caseInsensitive = d.caseRestore
		if d.serverFilter != nil {
			d.serverFilter.CaseInsensitive = d.caseRestore
		}
		if d.bufferPattern != "" {
			d.bufferCaseInsensitive = d.caseRestore
			d.recount()
		}
		d.caseApplying = false
	}
}

// resetDataFilters clears both filter surfaces. Object changes discard the
// whole data state; tab changes call this to restore the unfiltered window.
func (m *Model) resetDataFilters() tea.Cmd {
	d := &m.data
	d.prompt = promptNone
	d.bufferPattern = ""
	d.bufferCursor = 0
	d.matches = nil
	d.matchCells = nil
	d.serverPattern = ""
	d.serverCursor = 0
	d.serverField = ""
	d.filterApplying = false
	d.filterClearing = false
	d.filterRestore = nil
	d.caseApplying = false
	d.caseRestore = false
	if d.serverFilter == nil {
		d.serverCached = nil
		return nil
	}
	return m.clearServerFilter()
}

// nextMatch walks matching cells for a buffer filter and loaded rows for a
// server filter, wrapping in both cases.
func (m *Model) nextMatch(delta int) tea.Cmd {
	if !m.dataFocused() {
		return nil
	}
	d := &m.data
	if d.bufferPattern == "" {
		if d.serverFilter == nil {
			return nil
		}
		count := d.dataRowCount()
		if count == 0 {
			return nil
		}
		position := d.dataRowPosition(d.cursorRow)
		if position < 0 {
			position = 0
		} else {
			position = (position + delta + count) % count
		}
		d.cursorRow = d.dataRowAt(position)
		m.ensureDataViewport()
		return nil
	}
	count := len(d.matchCells)
	if count == 0 {
		return nil
	}
	row := d.cursorRow - d.windowStart
	index := d.matchCellIndex(row, d.cursorCol)
	if index < 0 {
		if delta > 0 {
			index = d.matchCellAtOrAfter(row, d.cursorCol)
			if index >= count {
				index = 0
			}
		} else {
			index = d.matchCellAtOrBefore(row, d.cursorCol)
			if index < 0 {
				index = count - 1
			}
		}
	} else {
		index = (index + delta + count) % count
	}
	match := d.matchCells[index]
	d.cursorRow = d.windowStart + match.row
	d.cursorCol = match.col
	m.ensureDataViewport()
	m.ensureDataColumn()
	return nil
}

// toggleCaseMode flips the session-wide search case. A buffer projection
// recomputes immediately; an active server filter refetches from offset 0.
func (m *Model) toggleCaseMode() tea.Cmd {
	m.caseInsensitive = !m.caseInsensitive
	if m.screen != ScreenWorkspace || !m.dataTabActive() {
		return nil
	}
	d := &m.data
	if d.bufferPattern != "" {
		d.bufferCaseInsensitive = m.caseInsensitive
		d.recount()
		m.ensureDataViewport()
		m.ensureDataColumn()
	}
	if d.serverFilter != nil {
		d.caseApplying = true
		d.caseRestore = !m.caseInsensitive
		d.serverFilter.CaseInsensitive = m.caseInsensitive
		d.err = ""
		return m.beginDataFetch(0, d.batch, dataInitial)
	}
	return nil
}

// filterKeepsNewKey reports whether `n` belongs to match navigation instead of
// the picker's new-connection action.
func (m Model) filterKeepsNewKey() bool {
	if m.screen != ScreenWorkspace || !m.dataTabActive() {
		return false
	}
	return m.data.bufferPattern != "" || m.data.serverFilter != nil || m.data.prompt != promptNone
}

// renderFilterBar draws the one-line inline prompt or active buffer filter.
func (m Model) renderFilterBar(width int) string {
	d := m.data
	if d.prompt == promptServer {
		label := m.styles.Accent.Render("regexp on " + oneLine(d.serverField))
		value := m.renderFilterValue(d.serverPattern, d.serverCursor)
		hint := m.styles.Muted.Render("Enter apply · Esc cancel")
		return Truncate(label+" "+value+"  "+hint, width)
	}
	label := m.styles.Accent.Render("/")
	value := m.renderFilterValue(d.bufferPattern, d.bufferCursor)
	var count string
	switch {
	case d.bufferPattern == "":
		count = m.styles.Muted.Render("type to filter · Esc clear")
	case len(d.matchCells) == 0:
		count = m.styles.Error.Render("0 matches · Esc clear")
	default:
		position, total := d.matchNumber()
		count = m.styles.Muted.Render(fmt.Sprintf("match %d of %d · Esc clear", position, total))
	}
	if d.prompt == promptBuffer && d.bufferPattern != "" && len(d.matches) > 0 {
		count += m.styles.Muted.Render(" · Enter keep")
	}
	return Truncate(label+" "+value+"  "+count, width)
}

// renderNoMatchLine carries the full no-match copy on its own row so it fits
// at the 80x24 floor instead of truncating inside the bar.
func (m Model) renderNoMatchLine(width int) string {
	text := fmt.Sprintf("No matches for %q. Esc clears the filter.", filterText(m.data.bufferPattern))
	return m.styles.Error.Render(Truncate(text, width))
}

// filteredEmptyLine names the filtered empty state instead of pretending the
// whole object is empty.
func (m Model) filteredEmptyLine(width int) string {
	field := ""
	if m.data.serverFilter != nil {
		field = oneLine(m.data.serverFilter.Field)
	}
	text := "No rows match the regexp on " + field + "."
	return m.styles.Muted.Render(Truncate(text, width))
}

// filterText sanitizes a pattern without trimming meaningful leading or
// trailing spaces.
func filterText(value string) string {
	value = Sanitize(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return value
}

func (m Model) renderFilterValue(value string, cursor int) string {
	clean := filterText(value)
	if m.data.prompt == promptNone {
		return m.styles.Text.Render(clean)
	}
	runes := []rune(clean)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	return m.styles.Text.Render(string(runes[:cursor])) +
		m.styles.Accent.Render("▏") +
		m.styles.Text.Render(string(runes[cursor:]))
}

// filterBannerLines wraps the filtered banner at the pane width. At 80x24 the
// full statement is wider than the 47-cell main pane, so it takes two rows.
func (m Model) filterBannerLines(width int) []string {
	filter := m.data.serverFilter
	if filter == nil || m.data.filterApplying || m.data.filterClearing {
		return nil
	}
	text := fmt.Sprintf("Filtered by regexp on %s; %s matching rows loaded",
		oneLine(filter.Field), formatCount(int64(len(m.data.rows))))
	if m.data.estimateSource != "" && m.data.estimated >= 0 {
		text += fmt.Sprintf(" of ~%s total rows (table estimate)", formatCount(m.data.estimated))
	} else {
		text += " total unavailable"
	}
	lines := wrapWords(text, width)
	for i := range lines {
		lines[i] = m.styles.Info.Render(lines[i])
	}
	return lines
}

// wrapWords breaks text at spaces and truncates any single overlong word.
func wrapWords(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if lipgloss.Width(current)+1+lipgloss.Width(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, Truncate(current, width))
		current = word
	}
	if current != "" {
		lines = append(lines, Truncate(current, width))
	}
	return lines
}
