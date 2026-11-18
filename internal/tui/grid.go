package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

const (
	dataPageSize = 100
	// TODO(review): restore dataFetchAllSize to 10000 after scroll debugging.
	dataFetchAllSize   = 1000
	dataMaxWindowRows  = 10000
	dataMaxWindowBytes = 64 << 20
	dataMaxColumnWidth = 32
)

type dataDirection int

const (
	dataInitial dataDirection = iota
	dataNext
	dataPrevious
	dataAll
	dataStart
)

type visibleColumn struct {
	index int
	width int
}

// dataColumn is one column of the rendered grid. A column may be a whole
// engine column or one field of a JSON object, addressed by path.
type dataColumn struct {
	name  string
	index int
	path  []string
	width int
}

// dataPageMsg carries one page result. Stale generations are dropped.
type dataPageMsg struct {
	gen       uint64
	direction dataDirection
	page      domain.DataPage
	err       error
}

// dataState is the Data tab's loaded window. windowStart is the absolute
// zero-based index of rows[0], so row numbers stay stable across eviction.
type dataState struct {
	ref domain.ObjectRef
	has bool
	// loaded reports that at least one page succeeded; loading is a separate
	// per-fetch flag so previous rows stay visible during a fetch.
	loaded         bool
	loading        bool
	cancel         context.CancelFunc
	columns        []domain.Column
	rows           []domain.Row
	view           []dataColumn
	expandedJSON   map[int]bool
	gutter         int
	windowStart    int
	rowOffset      int
	cursorRow      int
	cursorCol      int
	colOffset      int
	hasMore        bool
	estimated      int64
	estimateSource string
	bytes          int
	batch          int
	orderKey       string
	orderKnown     bool
	err            string
	tooLarge       bool
	endDown        int
	endPg          int
	maxRows        int
	maxBytes       int
}

func newDataState() dataState {
	return dataState{
		batch:        dataPageSize,
		estimated:    -1,
		maxRows:      dataMaxWindowRows,
		maxBytes:     dataMaxWindowBytes,
		expandedJSON: map[int]bool{},
	}
}

// resetData clears the window and invalidates any in-flight page.
func (m *Model) resetData() {
	if m.data.cancel != nil {
		m.data.cancel()
	}
	m.dataGen++
	m.data = newDataState()
}

func (m *Model) resetDataFor(ref domain.ObjectRef) {
	m.resetData()
	m.data.ref = ref
	m.data.has = true
}

// toggleJSONColumns expands or collapses only the selected MongoDB column.
// Pressing the key again on the same column cycles it back.
func (m *Model) toggleJSONColumns() {
	d := &m.data
	if !d.loaded || len(d.columns) == 0 || len(d.view) == 0 {
		return
	}
	if d.cursorCol < 0 || d.cursorCol >= len(d.view) {
		return
	}
	index := d.view[d.cursorCol].index
	if d.expandedJSON[index] {
		delete(d.expandedJSON, index)
		d.recount()
		m.focusDataColumn(index)
		m.setMessage(levelInfo, "JSON field collapsed")
		return
	}
	if len(jsonFieldUnion(d.rows, index)) == 0 {
		m.setMessage(levelInfo, "selected column has no JSON object fields")
		return
	}
	if d.expandedJSON == nil {
		d.expandedJSON = map[int]bool{}
	}
	d.expandedJSON[index] = true
	d.recount()
	m.focusDataColumn(index)
	m.setMessage(levelInfo, "JSON field expanded")
}

// focusDataColumn puts the cursor on the first rendered column for one engine
// column: the parent when collapsed, the first subfield when expanded.
func (m *Model) focusDataColumn(index int) {
	d := &m.data
	for i, column := range d.view {
		if column.index == index {
			d.cursorCol = i
			break
		}
	}
	d.clampCursor()
	m.ensureDataColumn()
	m.ensureDataViewport()
}

func (d *dataState) recount() {
	grid := domain.Grid{Columns: d.columns, Rows: d.rows}
	d.bytes = grid.ApproxBytes()
	d.view = buildDataView(d.columns, d.rows, d.expandedJSON)
	d.gutter = gridGutter(d.windowStart + len(d.rows))
	d.clampCursor()
}

func (d *dataState) clampCursor() {
	if len(d.rows) == 0 {
		d.cursorRow = d.windowStart
		d.rowOffset = d.windowStart
		d.cursorCol = 0
		d.colOffset = 0
		return
	}
	last := d.windowStart + len(d.rows) - 1
	if d.cursorRow < d.windowStart {
		d.cursorRow = d.windowStart
	}
	if d.cursorRow > last {
		d.cursorRow = last
	}
	if len(d.view) == 0 {
		d.cursorCol = 0
		d.colOffset = 0
	} else {
		if d.cursorCol < 0 {
			d.cursorCol = 0
		}
		if d.cursorCol >= len(d.view) {
			d.cursorCol = len(d.view) - 1
		}
		if d.colOffset < 0 {
			d.colOffset = 0
		}
		if d.colOffset > d.cursorCol {
			d.colOffset = d.cursorCol
		}
		if d.colOffset >= len(d.view) {
			d.colOffset = len(d.view) - 1
		}
	}
	if d.rowOffset < d.windowStart {
		d.rowOffset = d.windowStart
	}
	if d.rowOffset > last {
		d.rowOffset = last
	}
}

// mergePage appends or prepends one page using the union of column names. New
// columns extend every existing row with empty cells; incoming rows are
// remapped by name so Mongo first-seen order can vary between pages.
func (d *dataState) mergePage(incoming domain.Grid, prepend bool) {
	columns := append([]domain.Column(nil), d.columns...)
	index := make(map[string]int, len(columns))
	for i, column := range columns {
		index[column.Name] = i
	}
	mapping := make([]int, len(incoming.Columns))
	for i, column := range incoming.Columns {
		if existing, ok := index[column.Name]; ok {
			mapping[i] = existing
			continue
		}
		mapping[i] = len(columns)
		index[column.Name] = len(columns)
		columns = append(columns, column)
	}
	if len(columns) > len(d.columns) {
		for i := range d.rows {
			d.rows[i].Cells = padCells(d.rows[i].Cells, len(columns))
		}
	}
	rows := make([]domain.Row, len(incoming.Rows))
	for i, incomingRow := range incoming.Rows {
		cells := make([]domain.Cell, len(columns))
		for j := range cells {
			cells[j] = domain.Cell{Text: "", Kind: domain.CellText}
		}
		for j, cell := range incomingRow.Cells {
			if j < len(mapping) {
				cells[mapping[j]] = cell
			}
		}
		rows[i] = domain.Row{Cells: cells}
	}
	if prepend {
		d.rows = append(rows, d.rows...)
	} else {
		d.rows = append(d.rows, rows...)
	}
	d.columns = columns
	d.recount()
}

func padCells(cells []domain.Cell, width int) []domain.Cell {
	if len(cells) >= width {
		return cells
	}
	padded := make([]domain.Cell, width)
	copy(padded, cells)
	for i := len(cells); i < width; i++ {
		padded[i] = domain.Cell{Text: "", Kind: domain.CellText}
	}
	return padded
}

// capWindow enforces the row and byte caps. Forward paging drops the oldest
// rows at the front; scrolling back drops the farthest rows at the tail so the
// page the user just fetched survives.
func (d *dataState) capWindow(dropFront bool) {
	over := func() bool {
		return len(d.rows) > d.maxRows || d.bytes > d.maxBytes
	}
	if !over() {
		return
	}
	d.bytes = domain.Grid{Columns: d.columns, Rows: d.rows}.ApproxBytes()
	hadRows := len(d.rows) > 0
	for over() && len(d.rows) > 0 {
		if dropFront {
			d.bytes -= d.rows[0].ApproxBytes()
			d.rows = d.rows[1:]
			d.windowStart++
		} else {
			d.bytes -= d.rows[len(d.rows)-1].ApproxBytes()
			d.rows = d.rows[:len(d.rows)-1]
			d.hasMore = true
		}
	}
	if hadRows && len(d.rows) == 0 {
		d.tooLarge = true
	}
	d.recount()
}

func gridColumnWidths(columns []domain.Column, rows []domain.Row) []int {
	widths := make([]int, len(columns))
	for i, column := range columns {
		widths[i] = lipgloss.Width(oneLine(column.Name))
	}
	for _, row := range rows {
		for i := range columns {
			if i >= len(row.Cells) {
				continue
			}
			display := row.Cells[i].Display()
			if len(display) > dataMaxColumnWidth*8 {
				widths[i] = dataMaxColumnWidth
				continue
			}
			width := lipgloss.Width(oneLine(display))
			if width > widths[i] {
				widths[i] = width
			}
		}
	}
	for i := range widths {
		if widths[i] < 4 {
			widths[i] = 4
		}
		if widths[i] > dataMaxColumnWidth {
			widths[i] = dataMaxColumnWidth
		}
	}
	return widths
}

// buildDataView projects engine columns into rendered columns. Only the
// engine columns selected in expanded are replaced by dotted subcolumns in
// first-seen field order.
func buildDataView(columns []domain.Column, rows []domain.Row, expanded map[int]bool) []dataColumn {
	if len(expanded) == 0 {
		widths := gridColumnWidths(columns, rows)
		view := make([]dataColumn, len(columns))
		for i, column := range columns {
			view[i] = dataColumn{name: column.Name, index: i, width: widths[i]}
		}
		return view
	}
	view := make([]dataColumn, 0, len(columns))
	for i, column := range columns {
		fields := jsonFieldUnion(rows, i)
		if !expanded[i] || len(fields) == 0 {
			view = append(view, dataColumn{name: column.Name, index: i})
			continue
		}
		for _, field := range fields {
			view = append(view, dataColumn{
				name:  column.Name + "." + field,
				index: i,
				path:  []string{field},
			})
		}
	}
	for i := range view {
		view[i].width = dataColumnWidth(rows, view[i])
	}
	return view
}

// jsonFieldUnion returns the ordered union of keys in the JSON objects held by
// one engine column. Non-object values contribute nothing.
func jsonFieldUnion(rows []domain.Row, index int) []string {
	seen := map[string]bool{}
	var fields []string
	for _, row := range rows {
		if index < 0 || index >= len(row.Cells) {
			continue
		}
		cell := row.Cells[index]
		if cell.Kind != domain.CellJSON {
			continue
		}
		for _, key := range jsonObjectKeys(cell.Text) {
			if seen[key] {
				continue
			}
			seen[key] = true
			fields = append(fields, key)
		}
	}
	return fields
}

// jsonObjectKeys returns the keys of a JSON object in document order.
func jsonObjectKeys(text string) []string {
	decoder := json.NewDecoder(strings.NewReader(text))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil
	}
	var keys []string
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil
		}
		key, ok := token.(string)
		if !ok {
			return nil
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil
		}
		keys = append(keys, key)
	}
	return keys
}

func dataColumnWidth(rows []domain.Row, column dataColumn) int {
	width := lipgloss.Width(oneLine(column.name))
	for _, row := range rows {
		display := cellForColumn(row, column).Display()
		if len(display) > dataMaxColumnWidth*8 {
			width = dataMaxColumnWidth
			break
		}
		if cellWidth := lipgloss.Width(oneLine(display)); cellWidth > width {
			width = cellWidth
		}
	}
	if width < 4 {
		width = 4
	}
	if width > dataMaxColumnWidth {
		width = dataMaxColumnWidth
	}
	return width
}

func cellForColumn(row domain.Row, column dataColumn) domain.Cell {
	empty := domain.Cell{Text: "", Kind: domain.CellText}
	if column.index < 0 || column.index >= len(row.Cells) {
		return empty
	}
	cell := row.Cells[column.index]
	if len(column.path) == 0 {
		return cell
	}
	if cell.Kind != domain.CellJSON {
		return empty
	}
	return jsonPathCell(cell.Text, column.path)
}

func jsonPathCell(text string, path []string) domain.Cell {
	raw, ok := jsonPathValue(text, path)
	if !ok {
		return domain.Cell{Text: "", Kind: domain.CellText}
	}
	return cellFromJSON(raw)
}

func jsonPathValue(text string, path []string) (json.RawMessage, bool) {
	current := json.RawMessage(text)
	for _, key := range path {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(current, &object); err != nil {
			return nil, false
		}
		next, ok := object[key]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

// cellFromJSON maps one decoded JSON value to a cell. Null stays distinct from
// an empty string; objects and arrays stay compact JSON.
func cellFromJSON(raw json.RawMessage) domain.Cell {
	text := strings.TrimSpace(string(raw))
	switch {
	case text == "" || text == "null":
		return domain.NullCell()
	case text == "true" || text == "false":
		return domain.Cell{Text: text, Kind: domain.CellBool}
	case strings.HasPrefix(text, `"`):
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return domain.Cell{Text: text, Kind: domain.CellText}
		}
		return domain.TextCell(value)
	case strings.HasPrefix(text, "{") || strings.HasPrefix(text, "["):
		return domain.Cell{Text: domain.CompactJSONText(text), Kind: domain.CellJSON}
	default:
		return domain.Cell{Text: text, Kind: domain.CellNumber}
	}
}

func gridGutter(lastAbsolute int) int {
	width := len(strconv.Itoa(lastAbsolute + 1))
	if width < 3 {
		width = 3
	}
	return width
}

// visibleColumnStart keeps the cursor column in view. A single column wider
// than the pane is truncated rather than hidden.
func visibleColumnStart(widths []int, start, cursor, available int) int {
	if len(widths) == 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(widths) {
		cursor = len(widths) - 1
	}
	if start < 0 {
		start = 0
	}
	if start > cursor {
		start = cursor
	}
	if start >= len(widths) {
		start = len(widths) - 1
	}
	if available < 1 {
		available = 1
	}
	fits := func(from int) bool {
		total := 0
		for i := from; i <= cursor; i++ {
			if i > from {
				total++
			}
			total += widths[i]
			if total > available {
				return false
			}
		}
		return true
	}
	for start < cursor && !fits(start) {
		start++
	}
	return start
}

func (m *Model) resetDataGestures(key string) {
	if key != "down" {
		m.data.endDown = 0
	}
	if key != "pgdown" {
		m.data.endPg = 0
	}
}

// nextPage fetches the next batch at the end of the loaded window.
func (m *Model) nextDataPage() tea.Cmd {
	if !m.dataTabActive() {
		m.setMessage(levelInfo, "select a table or collection to browse")
		return nil
	}
	d := &m.data
	if d.loading {
		return nil
	}
	if !d.loaded {
		return m.beginDataFetch(0, d.batch, dataInitial)
	}
	if !d.hasMore {
		m.setMessage(levelInfo, "end of data")
		return nil
	}
	return m.beginDataFetch(d.windowStart+len(d.rows), d.batch, dataNext)
}

// fetchAllData appends one full-fetch batch and grows the retained window by
// the same amount, so repeated Ctrl+A presses accumulate rows instead of
// sliding the window forward. The byte cap still applies.
func (m *Model) fetchAllData() tea.Cmd {
	if !m.dataTabActive() {
		m.setMessage(levelInfo, "select a table or collection to browse")
		return nil
	}
	d := &m.data
	d.batch = dataFetchAllSize
	if d.loading {
		return nil
	}
	if !d.loaded {
		return m.beginDataFetch(0, dataFetchAllSize, dataInitial)
	}
	if !d.hasMore {
		m.setMessage(levelInfo, "end of data")
		return nil
	}
	end := d.windowStart + len(d.rows)
	target := end + dataFetchAllSize
	if target > d.maxRows {
		d.maxRows = target
	}
	return m.beginDataFetch(end, dataFetchAllSize, dataAll)
}

func (m *Model) previousDataPage() tea.Cmd {
	d := &m.data
	if !d.loaded || d.loading || d.windowStart <= 0 {
		return nil
	}
	offset := d.windowStart - d.batch
	if offset < 0 {
		offset = 0
	}
	limit := d.windowStart - offset
	if limit <= 0 {
		return nil
	}
	return m.beginDataFetch(offset, limit, dataPrevious)
}

// dataRangeStart jumps to the first loaded row. If the front of the window was
// evicted, it fetches the whole missing prefix in one request so the range
// becomes contiguous again instead of re-reading rows already in memory.
func (m *Model) dataRangeStart() tea.Cmd {
	if !m.dataFocused() {
		return nil
	}
	d := &m.data
	d.endDown = 0
	d.endPg = 0
	if !d.loaded {
		return m.ensureData()
	}
	if len(d.rows) == 0 {
		return nil
	}
	if d.windowStart > 0 {
		return m.beginDataFetch(0, d.windowStart, dataStart)
	}
	d.cursorRow = d.windowStart
	d.rowOffset = d.windowStart
	m.ensureDataViewport()
	return nil
}

// dataRangeEnd jumps to the last loaded row without fetching more pages.
func (m *Model) dataRangeEnd() tea.Cmd {
	if !m.dataFocused() {
		return nil
	}
	d := &m.data
	d.endDown = 0
	d.endPg = 0
	if !d.loaded {
		return m.ensureData()
	}
	if len(d.rows) == 0 {
		return nil
	}
	d.cursorRow = d.windowStart + len(d.rows) - 1
	m.ensureDataViewport()
	return nil
}

// beginDataFetch starts one serialized page request. A generation identifies
// the result; Esc and object changes invalidate older work.
func (m *Model) beginDataFetch(offset, limit int, direction dataDirection) tea.Cmd {
	if m.session == nil || !m.data.has {
		return nil
	}
	if m.data.cancel != nil {
		m.data.cancel()
	}
	m.dataGen++
	gen := m.dataGen
	base := m.sessionCtx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithCancel(base)
	m.data.cancel = cancel
	m.data.loading = true
	m.data.err = ""
	m.setMessage(levelInfo, "loading rows…")

	session := m.session
	ref := m.data.ref
	return func() tea.Msg {
		page, err := session.Page(ctx, domain.DataQuery{Ref: ref, Offset: offset, Limit: limit})
		return dataPageMsg{
			gen:       gen,
			direction: direction,
			page:      page,
			err:       err,
		}
	}
}

func (m Model) handleDataPage(msg dataPageMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.dataGen {
		return m, nil
	}
	m.data.loading = false
	if m.data.cancel != nil {
		m.data.cancel()
		m.data.cancel = nil
	}
	if msg.err != nil {
		if m.session != nil && !m.session.Alive() {
			return m.leaveWorkspace(msg.err)
		}
		if domain.KindOf(msg.err) == domain.KindCancelled || errors.Is(msg.err, context.Canceled) {
			m.setMessage(levelInfo, "cancelled")
			return m, nil
		}
		m.data.err = oneLine(msg.err.Error())
		m.setMessage(levelError, m.data.err)
		return m, nil
	}
	m.data.err = ""
	m.data.tooLarge = false
	if msg.page.OrderKeyKnown {
		m.data.orderKnown = true
		m.data.orderKey = msg.page.OrderKey
	} else if !m.data.orderKnown {
		m.data.orderKnown = true
		m.data.orderKey = ""
	}
	switch msg.direction {
	case dataInitial:
		m.data.columns = append([]domain.Column(nil), msg.page.Grid.Columns...)
		m.data.rows = append([]domain.Row(nil), msg.page.Grid.Rows...)
		m.data.windowStart = 0
		m.data.cursorRow = 0
		m.data.cursorCol = 0
		m.data.colOffset = 0
		m.data.rowOffset = 0
		m.data.hasMore = msg.page.HasMore
		m.data.loaded = true
		m.data.tooLarge = false
		m.data.recount()
		m.data.capWindow(true)
	case dataPrevious:
		oldStart := m.data.windowStart
		m.data.mergePage(msg.page.Grid, true)
		if msg.page.Offset >= 0 {
			m.data.windowStart = msg.page.Offset
		} else {
			m.data.windowStart -= len(msg.page.Grid.Rows)
		}
		if len(msg.page.Grid.Rows) > 0 {
			m.data.cursorRow = msg.page.Offset + len(msg.page.Grid.Rows) - 1
		} else {
			m.data.cursorRow = oldStart - 1
		}
		m.data.capWindow(false)
	case dataStart:
		m.data.mergePage(msg.page.Grid, true)
		if msg.page.Offset >= 0 {
			m.data.windowStart = msg.page.Offset
		} else {
			m.data.windowStart -= len(msg.page.Grid.Rows)
		}
		m.data.cursorRow = m.data.windowStart
		m.data.rowOffset = m.data.windowStart
		m.data.capWindow(false)
	case dataNext, dataAll:
		m.data.mergePage(msg.page.Grid, false)
		m.data.hasMore = msg.page.HasMore
		m.data.capWindow(true)
	}
	m.data.estimated = msg.page.EstimatedTotal
	m.data.estimateSource = msg.page.EstimateSource
	m.data.loaded = true
	m.data.recount()
	m.ensureDataViewport()
	if msg.direction == dataInitial && m.data.orderKnown && m.data.orderKey == "" {
		m.setMessage(levelWarning, "no stable order; rows may repeat or skip")
	} else {
		m.message = ""
		m.messageLevel = levelNone
	}
	return m, nil
}

// cancelDataFetch keeps the loaded rows but invalidates the in-flight page.
func (m *Model) cancelDataFetch() {
	if m.data.cancel != nil {
		m.data.cancel()
		m.data.cancel = nil
	}
	m.dataGen++
	m.data.loading = false
}

func (m Model) dataTabActive() bool {
	return m.details.has && m.details.currentTab() == "Data"
}

func (m Model) dataFocused() bool {
	return m.screen == ScreenWorkspace && m.focus == FocusMain && m.dataTabActive()
}

// ensureData starts the initial page when the Data tab becomes active for a
// table or collection. It never triggers a database call from View.
func (m *Model) ensureData() tea.Cmd {
	if m.session == nil || !m.details.has || m.details.currentTab() != "Data" {
		return nil
	}
	node := m.details.node
	if node.Kind != domain.KindTable && node.Kind != domain.KindCollection {
		return nil
	}
	if !m.data.has || m.data.ref.Key() != node.Ref.Key() {
		m.resetDataFor(node.Ref)
	}
	if m.data.loading || m.data.loaded || m.data.err != "" {
		return nil
	}
	return m.beginDataFetch(0, dataPageSize, dataInitial)
}

func (m *Model) moveDataRow(delta int) tea.Cmd {
	if !m.dataFocused() || !m.data.loaded || len(m.data.rows) == 0 {
		return nil
	}
	d := &m.data
	last := d.windowStart + len(d.rows) - 1
	target := d.cursorRow + delta
	if target < d.windowStart {
		if d.windowStart > 0 {
			return m.previousDataPage()
		}
		target = d.windowStart
	}
	if target > last {
		if d.hasMore {
			if delta == 1 {
				return m.dataDownGesture()
			}
			return m.nextDataPage()
		}
		target = last
		m.setMessage(levelInfo, "end of data")
	}
	d.cursorRow = target
	d.endDown = 0
	m.ensureDataViewport()
	return nil
}

func (m *Model) moveDataPage(delta int) tea.Cmd {
	if !m.dataFocused() || !m.data.loaded || len(m.data.rows) == 0 {
		return nil
	}
	d := &m.data
	last := d.windowStart + len(d.rows) - 1
	if delta < 0 {
		if d.cursorRow == d.windowStart && d.windowStart > 0 {
			return m.previousDataPage()
		}
		d.cursorRow -= m.dataPageRows()
		if d.cursorRow < d.windowStart {
			d.cursorRow = d.windowStart
		}
		d.endPg = 0
		m.ensureDataViewport()
		return nil
	}
	if d.cursorRow >= last {
		if d.hasMore {
			return m.dataPgDownGesture()
		}
		m.setMessage(levelInfo, "end of data")
		return nil
	}
	d.cursorRow += m.dataPageRows()
	if d.cursorRow > last {
		d.cursorRow = last
	}
	d.endPg = 0
	m.ensureDataViewport()
	return nil
}

func (m *Model) dataDownGesture() tea.Cmd {
	d := &m.data
	d.endPg = 0
	if !d.hasMore {
		m.setMessage(levelInfo, "end of data")
		return nil
	}
	switch d.endDown {
	case 0:
		d.endDown = 1
		m.setMessage(levelInfo, "press Down twice more")
	case 1:
		d.endDown = 2
		m.setMessage(levelInfo, "press Down again")
	default:
		d.endDown = 0
		return m.nextDataPage()
	}
	return nil
}

func (m *Model) dataPgDownGesture() tea.Cmd {
	d := &m.data
	d.endDown = 0
	if !d.hasMore {
		m.setMessage(levelInfo, "end of data")
		return nil
	}
	if d.endPg == 0 {
		d.endPg = 1
		m.setMessage(levelInfo, "press PgDown again")
		return nil
	}
	d.endPg = 0
	return m.nextDataPage()
}

func (m *Model) moveDataColumn(delta int) {
	if !m.dataFocused() || len(m.data.view) == 0 {
		return
	}
	d := &m.data
	d.cursorCol += delta
	if d.cursorCol < 0 {
		d.cursorCol = 0
	}
	if d.cursorCol >= len(d.view) {
		d.cursorCol = len(d.view) - 1
	}
	m.ensureDataColumn()
}

func (m *Model) moveDataColumnEdge(last bool) {
	if !m.dataFocused() || len(m.data.view) == 0 {
		return
	}
	if last {
		m.data.cursorCol = len(m.data.view) - 1
	} else {
		m.data.cursorCol = 0
	}
	m.ensureDataColumn()
}

func (m *Model) ensureDataColumn() {
	d := &m.data
	if len(d.view) == 0 {
		return
	}
	widths := make([]int, len(d.view))
	for i := range d.view {
		widths[i] = d.view[i].width
	}
	available := m.dataInnerWidth() - d.gutter - 2
	if available < 1 {
		available = 1
	}
	d.colOffset = visibleColumnStart(widths, d.colOffset, d.cursorCol, available)
}

func (m *Model) ensureDataViewport() {
	d := &m.data
	if len(d.rows) == 0 {
		d.rowOffset = d.windowStart
		return
	}
	pageRows := m.dataPageRows()
	if pageRows < 1 {
		pageRows = 1
	}
	last := d.windowStart + len(d.rows) - 1
	if d.cursorRow < d.rowOffset {
		d.rowOffset = d.cursorRow
	}
	if d.cursorRow >= d.rowOffset+pageRows {
		d.rowOffset = d.cursorRow - pageRows + 1
	}
	if d.rowOffset < d.windowStart {
		d.rowOffset = d.windowStart
	}
	if d.rowOffset > last {
		d.rowOffset = last
	}
}

func (m Model) dataBodyHeight() int {
	regions := layout.Compute(m.width, m.height)
	height := regions.BodyHeight - 4
	if height < 1 {
		height = 1
	}
	return height
}

func (m Model) dataPageRows() int {
	height := m.dataBodyHeight() - 2 // column header + header rule
	if m.data.orderKnown && m.data.orderKey == "" {
		height--
	}
	if m.data.err != "" {
		height--
	}
	if height < 1 {
		height = 1
	}
	return height
}

func (m Model) dataInnerWidth() int {
	regions := layout.Compute(m.width, m.height)
	width := regions.MainWidth - 4
	if width < 1 {
		width = 1
	}
	return width
}

func (m Model) dataSummary() string {
	d := m.data
	if !d.has || !m.dataTabActive() {
		return ""
	}
	if d.loading {
		return "loading rows…"
	}
	if !d.loaded {
		return "not loaded"
	}
	if d.tooLarge {
		return "row exceeds the 64 MB window"
	}
	if len(d.rows) == 0 {
		return "0 rows"
	}
	end := d.windowStart + len(d.rows)
	var text string
	if d.estimateSource != "" && d.estimated >= 0 {
		if d.windowStart == 0 {
			text = fmt.Sprintf("loaded %s of ~%s", formatCount(int64(len(d.rows))), formatCount(d.estimated))
		} else {
			text = fmt.Sprintf("rows %s–%s of ~%s",
				formatCount(int64(d.windowStart+1)), formatCount(int64(end)), formatCount(d.estimated))
		}
	} else {
		if d.windowStart == 0 {
			text = fmt.Sprintf("loaded %s · total unavailable", formatCount(int64(len(d.rows))))
		} else {
			text = fmt.Sprintf("rows %s–%s · total unavailable",
				formatCount(int64(d.windowStart+1)), formatCount(int64(end)))
		}
	}
	if d.hasMore {
		text += " · more available"
	}
	return text
}

func formatCount(value int64) string {
	if value < 0 {
		return strconv.FormatInt(value, 10)
	}
	digits := strconv.FormatInt(value, 10)
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(digits[:first])
	for i := first; i < len(digits); i += 3 {
		b.WriteString(",")
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

func dataObjectName(ref domain.ObjectRef) string {
	switch {
	case ref.Schema != "":
		return ref.Schema + "." + ref.Name
	case ref.Database != "":
		return ref.Database + "." + ref.Name
	default:
		return ref.Name
	}
}

// viewData renders the Data tab body. Lines are limited to height so the box
// frame never grows.
func (m Model) viewData(width, height int) []string {
	var lines []string
	d := m.data
	if !d.has {
		return []string{m.styles.Muted.Render("Select a table or collection to browse.")}
	}
	if d.orderKnown && d.orderKey == "" {
		lines = append(lines, m.styles.Warning.Render(Truncate("no stable order; rows may repeat or skip", width)))
	}
	if d.err != "" {
		lines = append(lines, m.styles.Error.Render(Truncate("error: "+oneLine(d.err), width)))
	}
	if !d.loaded {
		if len(lines) < height {
			switch {
			case d.err != "":
				lines = append(lines, m.styles.Muted.Render(Truncate("press m to retry", width)))
			case d.loading:
				lines = append(lines, m.styles.Muted.Render(Truncate("loading rows…", width)))
			default:
				lines = append(lines, m.styles.Muted.Render(Truncate("press Enter to load rows", width)))
			}
		}
		return clipLines(lines, height)
	}
	remaining := height - len(lines)
	if remaining < 1 {
		return clipLines(lines, height)
	}
	lines = append(lines, m.renderGrid(width, remaining)...)
	return clipLines(lines, height)
}

func clipLines(lines []string, height int) []string {
	if height < 0 {
		height = 0
	}
	if len(lines) > height {
		return lines[:height]
	}
	return lines
}

func (m Model) renderGrid(width, height int) []string {
	d := m.data
	if len(d.columns) == 0 {
		if len(d.rows) == 0 {
			if d.tooLarge {
				return []string{m.styles.Warning.Render(Truncate("Row exceeds the 64 MB window. Press m to retry.", width))}
			}
			name := oneLine(dataObjectName(d.ref))
			return []string{m.styles.Muted.Render(Truncate("No rows. "+name+" is empty.", width))}
		}
		return nil
	}
	view := d.view
	if len(view) == 0 {
		view = buildDataView(d.columns, d.rows, d.expandedJSON)
	}
	widths := make([]int, len(view))
	for i := range view {
		widths[i] = view[i].width
	}
	gutter := d.gutter
	if gutter < 1 {
		gutter = 3
	}
	available := width - gutter - 2
	if available < 1 {
		available = 1
	}
	startCol := visibleColumnStart(widths, d.colOffset, d.cursorCol, available)

	var visible []visibleColumn
	used := 0
	for i := startCol; i < len(widths); i++ {
		addition := widths[i]
		if len(visible) > 0 {
			addition++
		}
		if used+addition > available {
			if len(visible) == 0 {
				visible = append(visible, visibleColumn{index: i, width: available})
			}
			break
		}
		used += addition
		visible = append(visible, visibleColumn{index: i, width: widths[i]})
	}

	headerStyle := m.styles.Label.Background(lipgloss.Color(theme.Nightshift.Raised))
	cursorHeaderStyle := m.styles.Accent.Bold(true).Underline(true).Background(lipgloss.Color(theme.Nightshift.Raised))
	var header strings.Builder
	header.WriteString(headerStyle.Render(strings.Repeat(" ", gutter) + "  "))
	headerUsed := gutter + 2
	for j, column := range visible {
		sep := ""
		if j > 0 {
			sep = " "
		}
		name := Truncate(oneLine(view[column.index].name), column.width)
		text := sep + Pad(name, column.width)
		headerUsed += lipgloss.Width(text)
		if column.index == d.cursorCol {
			header.WriteString(cursorHeaderStyle.Render(text))
		} else {
			header.WriteString(headerStyle.Render(text))
		}
	}
	if headerUsed < width {
		header.WriteString(headerStyle.Render(strings.Repeat(" ", width-headerUsed)))
	}
	lines := []string{header.String()}
	if height <= 1 {
		return lines
	}
	lines = append(lines, m.styles.Border.Render(strings.Repeat("─", width)))
	if height <= 2 {
		return lines
	}

	viewRows := height - 2
	if len(d.rows) == 0 {
		if d.tooLarge {
			lines = append(lines, m.styles.Warning.Render(Truncate("Row exceeds the 64 MB window. Press m to retry.", width)))
			return lines
		}
		name := oneLine(dataObjectName(d.ref))
		lines = append(lines, m.styles.Muted.Render(Truncate("No rows. "+name+" is empty.", width)))
		return lines
	}
	startRow := d.rowOffset
	if d.cursorRow < startRow {
		startRow = d.cursorRow
	}
	if d.cursorRow >= startRow+viewRows {
		startRow = d.cursorRow - viewRows + 1
	}
	if startRow < d.windowStart {
		startRow = d.windowStart
	}
	last := d.windowStart + len(d.rows) - 1
	if startRow > last {
		startRow = last
	}
	end := startRow + viewRows
	if end > last+1 {
		end = last + 1
	}
	for absolute := startRow; absolute < end; absolute++ {
		row := d.rows[absolute-d.windowStart]
		lines = append(lines, m.renderGridRow(row, view, visible, gutter, absolute, width))
	}
	return lines
}

func (m Model) renderGridRow(row domain.Row, view []dataColumn, visible []visibleColumn, gutter, absolute, width int) string {
	d := m.data
	if absolute == d.cursorRow {
		var b strings.Builder
		b.WriteString(m.styles.Select.Render(fmt.Sprintf("%*d  ", gutter, absolute+1)))
		used := gutter + 2
		for j, column := range visible {
			sep := ""
			if j > 0 {
				sep = " "
			}
			text := cellForColumn(row, view[column.index]).Display()
			cell := sep + Pad(Truncate(oneLine(text), column.width), column.width)
			used += lipgloss.Width(cell)
			if column.index == d.cursorCol {
				b.WriteString(m.styles.Cursor.Render(cell))
			} else {
				b.WriteString(m.styles.Select.Render(cell))
			}
		}
		if used < width {
			b.WriteString(m.styles.Select.Render(strings.Repeat(" ", width-used)))
		}
		return b.String()
	}
	var b strings.Builder
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("%*d", gutter, absolute+1)))
	b.WriteString("  ")
	for j, column := range visible {
		if j > 0 {
			b.WriteString(" ")
		}
		cell := cellForColumn(row, view[column.index])
		style := m.styles.Text
		if cell.Null {
			style = m.styles.Muted
		}
		text := Truncate(oneLine(cell.Display()), column.width)
		b.WriteString(style.Render(Pad(text, column.width)))
	}
	return b.String()
}
