package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type rowAction int

const (
	rowInsert rowAction = iota
	rowUpdate
	rowDelete
)

type rowValueMode int

const (
	rowValue rowValueMode = iota
	rowNull
	rowDefault
)

type rowInsertState struct {
	ref      domain.ObjectRef
	columns  []domain.Column
	values   []string
	modes    []rowValueMode
	cursor   int
	offset   int
	valuePos int
	mongo    bool
	document string
	docPos   int
	err      string
}

type rowEditState struct {
	ref      domain.ObjectRef
	column   string
	label    string
	value    string
	mode     rowValueMode
	valuePos int
	rows     [][]domain.RowValue
	kind     domain.CellKind
	mongo    bool
	err      string
}

type rowDeleteState struct {
	ref  domain.ObjectRef
	rows [][]domain.RowValue
}

type rowSelection struct {
	id    string
	keys  []domain.RowValue
	label string
}

type rowDoneMsg struct {
	gen    uint64
	action rowAction
	result domain.MutationResult
	ref    domain.ObjectRef
	count  int
	err    error
}

func (m *Model) rowWritable() bool {
	if m.session == nil {
		m.setMessage(levelInfo, "no active connection")
		return false
	}
	if m.session.ReadOnly() {
		m.setMessage(levelError, "Blocked: row changes require read-write mode. Ctrl+X Ctrl+W to switch.")
		return false
	}
	return true
}

func (m Model) rowDataTab() bool {
	return m.screen == ScreenWorkspace && m.focus == FocusMain &&
		m.details.has && m.details.currentTab() == "Data" && !m.resultsActive
}

func (m Model) startRowAdd() (tea.Model, tea.Cmd) {
	if !m.rowDataTab() {
		if m.resultsActive {
			m.setMessage(levelInfo, "row changes are available on the Data tab")
		} else {
			m.setMessage(levelInfo, "open the Data tab first")
		}
		return m, nil
	}
	if !m.rowWritable() {
		return m, nil
	}
	if !m.data.loaded || len(m.data.columns) == 0 {
		m.setMessage(levelInfo, "load Data before inserting a row")
		return m, nil
	}
	m.rowInsert = rowInsertState{
		ref:     m.data.ref,
		columns: append([]domain.Column(nil), m.data.columns...),
		values:  make([]string, len(m.data.columns)),
		modes:   make([]rowValueMode, len(m.data.columns)),
	}
	for i := range m.rowInsert.modes {
		m.rowInsert.modes[i] = rowDefault
	}
	if m.engine() == domain.EngineMongoDB {
		m.rowInsert.mongo = true
		m.rowInsert.document = "{}"
		m.rowInsert.docPos = len([]rune("{}"))
	}
	m.modal = modalRowInsert
	return m, nil
}

func (m Model) startRowEdit() (tea.Model, tea.Cmd) {
	if !m.rowDataTab() {
		if m.resultsActive {
			m.setMessage(levelInfo, "row changes are available on the Data tab")
		} else {
			m.setMessage(levelInfo, "open the Data tab first")
		}
		return m, nil
	}
	if !m.rowWritable() {
		return m, nil
	}
	if !m.data.loaded || len(m.data.view) == 0 || len(m.data.columns) == 0 {
		m.setMessage(levelInfo, "load Data before editing a cell")
		return m, nil
	}
	if !m.data.orderKnown || strings.TrimSpace(m.data.orderKey) == "" {
		m.setMessage(levelError, "cannot identify a row without a primary key or _id")
		return m, nil
	}
	if m.data.cursorCol < 0 || m.data.cursorCol >= len(m.data.view) {
		m.setMessage(levelInfo, "select a cell first")
		return m, nil
	}
	column := m.data.view[m.data.cursorCol]
	if column.index < 0 || column.index >= len(m.data.columns) {
		m.setMessage(levelInfo, "select a cell first")
		return m, nil
	}
	mongo := m.engine() == domain.EngineMongoDB
	columnName := column.name
	if mongo {
		if columnName == "_id" || strings.HasPrefix(columnName, "_id.") {
			m.setMessage(levelError, "the _id field is read-only")
			return m, nil
		}
		if !domain.ValidMongoPath(columnName) {
			m.setMessage(levelError, "the update field is not valid")
			return m, nil
		}
	} else {
		columnName = m.data.columns[column.index].Name
		if len(column.path) > 0 {
			m.setMessage(levelInfo, "collapse the JSON column to edit its parent value")
			return m, nil
		}
		if m.isOrderKeyColumn(columnName) {
			m.setMessage(levelError, "primary key columns are read-only in this editor")
			return m, nil
		}
	}
	current, ok := m.currentDataRow()
	if !ok {
		m.setMessage(levelInfo, "select a loaded row first")
		return m, nil
	}
	cell := cellForColumn(current, column)
	if cell.Kind == domain.CellBinary {
		m.setMessage(levelError, "binary values cannot be edited")
		return m, nil
	}
	targets, notice, level := m.writeTargets()
	if notice != "" {
		m.setMessage(level, notice)
		return m, nil
	}
	rows := make([][]domain.RowValue, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, target.keys)
	}
	mode := rowValue
	value := cell.Text
	if cell.Null {
		mode = rowNull
		value = ""
	}
	m.rowEdit = rowEditState{
		ref:      m.data.ref,
		column:   columnName,
		label:    column.name,
		value:    value,
		mode:     mode,
		valuePos: len([]rune(value)),
		rows:     rows,
		kind:     cell.Kind,
		mongo:    mongo,
	}
	m.modal = modalRowEdit
	return m, nil
}

func (m Model) startRowRemove() (tea.Model, tea.Cmd) {
	if !m.rowDataTab() {
		if m.resultsActive {
			m.setMessage(levelInfo, "row changes are available on the Data tab")
		} else {
			m.setMessage(levelInfo, "open the Data tab first")
		}
		return m, nil
	}
	if !m.rowWritable() {
		return m, nil
	}
	if !m.data.loaded || !m.data.orderKnown || strings.TrimSpace(m.data.orderKey) == "" {
		m.setMessage(levelError, "cannot identify a row without a primary key or _id")
		return m, nil
	}
	targets, notice, level := m.writeTargets()
	if notice != "" {
		m.setMessage(level, notice)
		return m, nil
	}
	rows := make([][]domain.RowValue, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, target.keys)
	}
	m.rowDelete = rowDeleteState{ref: m.data.ref, rows: rows}
	m.modal = modalRowDelete
	return m, nil
}

// writeTargets returns the explicit selection, or the current row when nothing
// is selected. The message and level explain why no target is available.
func (m Model) writeTargets() ([]rowSelection, string, messageLevel) {
	if selected := m.data.selectedRows(); len(selected) > 0 {
		return selected, "", levelNone
	}
	row, ok := m.currentDataRow()
	if !ok {
		return nil, "select a loaded row first", levelInfo
	}
	keys, ok := m.rowKeys(row)
	if !ok {
		for _, key := range m.rowKeyCells(row) {
			if key.Kind == domain.CellBinary {
				return nil, "binary key values cannot be used for row changes", levelError
			}
		}
		return nil, "the current row has no complete key", levelError
	}
	return []rowSelection{rowSelectionFor(keys)}, "", levelNone
}

func (m Model) currentDataRow() (domain.Row, bool) {
	position := m.data.dataRowPosition(m.data.cursorRow)
	if position < 0 {
		return domain.Row{}, false
	}
	absolute := m.data.dataRowAt(position)
	index := absolute - m.data.windowStart
	if index < 0 || index >= len(m.data.rows) {
		return domain.Row{}, false
	}
	return m.data.rows[index], true
}

func (m Model) rowKeys(row domain.Row) ([]domain.RowValue, bool) {
	keys := m.rowKeyCells(row)
	if len(keys) == 0 {
		return nil, false
	}
	for _, key := range keys {
		if key.Kind == domain.CellBinary {
			return nil, false
		}
	}
	return keys, true
}

func (m Model) rowKeyCells(row domain.Row) []domain.RowValue {
	names := splitRowKey(m.data.orderKey)
	if len(names) == 0 {
		return nil
	}
	index := make(map[string]int, len(m.data.columns))
	for i, column := range m.data.columns {
		index[column.Name] = i
	}
	keys := make([]domain.RowValue, 0, len(names))
	for _, name := range names {
		column, ok := index[name]
		if !ok || column >= len(row.Cells) {
			return nil
		}
		cell := row.Cells[column]
		if cell.Null {
			return nil
		}
		keys = append(keys, domain.RowValue{
			Column: name, Value: cell.Text, Kind: cell.Kind,
		})
	}
	return keys
}

func (m Model) isOrderKeyColumn(name string) bool {
	for _, key := range splitRowKey(m.data.orderKey) {
		if key == name {
			return true
		}
	}
	return false
}

func splitRowKey(key string) []string {
	parts := strings.Split(key, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func rowSelectionFor(keys []domain.RowValue) rowSelection {
	return rowSelection{
		id:    rowKeyID(keys),
		keys:  append([]domain.RowValue(nil), keys...),
		label: domain.RowKeyLabel(keys),
	}
}

func rowKeyID(keys []domain.RowValue) string {
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%d:%s:%t:%d:%s\x1e", len(key.Column), key.Column, key.Null, len(key.Value), key.Value)
	}
	return b.String()
}

func (d dataState) selectedCount() int { return len(d.selected) }

func (d dataState) isSelected(id string) bool {
	_, ok := d.selected[id]
	return ok
}

func (d dataState) selectedRows() []rowSelection {
	if len(d.selected) == 0 {
		return nil
	}
	rows := make([]rowSelection, 0, len(d.selected))
	for _, selection := range d.selected {
		rows = append(rows, selection)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].label == rows[j].label {
			return rows[i].id < rows[j].id
		}
		return rows[i].label < rows[j].label
	})
	return rows
}

func (m *Model) clearDataSelection() {
	m.data.selected = nil
}

func (m *Model) toggleDataSelection() tea.Cmd {
	if !m.dataFocused() || !m.data.loaded {
		return nil
	}
	if !m.data.orderKnown || strings.TrimSpace(m.data.orderKey) == "" {
		m.setMessage(levelError, "cannot select a row without a primary key or _id")
		return nil
	}
	row, ok := m.currentDataRow()
	if !ok {
		m.setMessage(levelInfo, "select a loaded row first")
		return nil
	}
	keys, ok := m.rowKeys(row)
	if !ok {
		m.setMessage(levelError, "this row has no stable key")
		return nil
	}
	selection := rowSelectionFor(keys)
	selected := m.data.selected
	if _, ok := selected[selection.id]; ok {
		delete(selected, selection.id)
	} else {
		if selected == nil {
			selected = map[string]rowSelection{}
			m.data.selected = selected
		}
		selected[selection.id] = selection
	}
	count := len(selected)
	if count == 0 {
		m.setMessage(levelInfo, "selection cleared")
	} else {
		m.setMessage(levelInfo, fmt.Sprintf("%d rows selected", count))
	}
	return nil
}

func (m Model) rowSelected(row domain.Row) bool {
	keys, ok := m.rowKeys(row)
	if !ok {
		return false
	}
	return m.data.isSelected(rowKeyID(keys))
}

func (m *Model) moveRowField(delta int) {
	ins := &m.rowInsert
	if len(ins.columns) == 0 {
		return
	}
	ins.cursor = (ins.cursor + delta + len(ins.columns)) % len(ins.columns)
	ins.valuePos = len([]rune(ins.values[ins.cursor]))
	ins.offset = completionWindowStart(ins.offset, ins.cursor, 10)
}

func (m Model) rowInsertInputValue() string {
	ins := m.rowInsert
	if ins.cursor < 0 || ins.cursor >= len(ins.values) {
		return ""
	}
	return ins.values[ins.cursor]
}

func (m *Model) setRowInsertInputValue(value string) {
	ins := &m.rowInsert
	if ins.cursor < 0 || ins.cursor >= len(ins.values) {
		return
	}
	ins.values[ins.cursor] = value
	ins.modes[ins.cursor] = rowValue
}

func (m *Model) cycleRowMode(mode rowValueMode) {
	ins := &m.rowInsert
	if ins.cursor < 0 || ins.cursor >= len(ins.modes) {
		return
	}
	ins.modes[ins.cursor] = mode
}

func (m Model) handleRowInsertKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	ins := &m.rowInsert
	key := msg.String()
	if ins.mongo {
		if action, ok := keymap.Lookup(key); ok {
			switch action {
			case keymap.Cancel:
				m.modal = modalNone
				return m, nil
			case keymap.Confirm:
				return m.submitRowInsert()
			}
		}
		value, pos := editFilterText(ins.document, ins.docPos, msg)
		ins.document = value
		ins.docPos = pos
		return m, nil
	}
	if key == "ctrl+n" {
		m.cycleRowMode(rowNull)
		return m, nil
	}
	if key == "ctrl+d" {
		m.cycleRowMode(rowDefault)
		return m, nil
	}
	if action, ok := keymap.Lookup(key); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			return m, nil
		case keymap.Tab, keymap.Down:
			m.moveRowField(1)
			return m, nil
		case keymap.ShiftTab, keymap.Up:
			m.moveRowField(-1)
			return m, nil
		case keymap.Confirm:
			return m.submitRowInsert()
		}
	}
	if ins.cursor < 0 || ins.cursor >= len(ins.values) {
		return m, nil
	}
	if msg.Text != "" && ins.modes[ins.cursor] != rowValue {
		m.setRowInsertInputValue("")
		ins.valuePos = 0
	}
	value, pos := editFilterText(m.rowInsertInputValue(), ins.valuePos, msg)
	m.setRowInsertInputValue(value)
	ins.valuePos = pos
	return m, nil
}

func (m Model) submitRowInsert() (tea.Model, tea.Cmd) {
	ins := m.rowInsert
	if ins.mongo {
		document := strings.TrimSpace(ins.document)
		if document == "" {
			m.rowInsert.err = "enter an extended JSON document"
			return m, nil
		}
		spec := domain.RowInsert{Ref: ins.ref, Document: document}
		return m.startRowMutation(rowInsert, ins.ref, 1, func(session *app.Session) (domain.MutationResult, error) {
			return session.InsertRow(context.Background(), spec)
		})
	}
	if len(ins.columns) == 0 {
		m.rowInsert.err = "the loaded grid has no columns"
		return m, nil
	}
	values := make([]domain.RowValue, 0, len(ins.columns))
	for i, column := range ins.columns {
		mode := rowValue
		if i < len(ins.modes) {
			mode = ins.modes[i]
		}
		value := ""
		if i < len(ins.values) {
			value = ins.values[i]
		}
		values = append(values, domain.RowValue{
			Column: column.Name, Value: value,
			Null: mode == rowNull, Default: mode == rowDefault,
		})
	}
	return m.startRowMutation(rowInsert, ins.ref, 1, func(session *app.Session) (domain.MutationResult, error) {
		return session.InsertRow(context.Background(), domain.RowInsert{Ref: ins.ref, Values: values})
	})
}

func (m Model) handleRowEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+n" {
		m.rowEdit.mode = rowNull
		m.rowEdit.err = ""
		return m, nil
	}
	if key == "ctrl+d" {
		if m.rowEdit.mongo {
			m.rowEdit.err = "MongoDB has no DEFAULT value"
			return m, nil
		}
		m.rowEdit.mode = rowDefault
		m.rowEdit.err = ""
		return m, nil
	}
	if action, ok := keymap.Lookup(key); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			return m, nil
		case keymap.Confirm:
			return m.submitRowEdit()
		}
	}
	if msg.Text != "" && m.rowEdit.mode != rowValue {
		m.rowEdit.value = ""
		m.rowEdit.valuePos = 0
		m.rowEdit.mode = rowValue
		m.rowEdit.err = ""
	}
	value, pos := editFilterText(m.rowEdit.value, m.rowEdit.valuePos, msg)
	m.rowEdit.value = value
	m.rowEdit.valuePos = pos
	return m, nil
}

func (m Model) submitRowEdit() (tea.Model, tea.Cmd) {
	edit := m.rowEdit
	if edit.mongo && edit.mode == rowDefault {
		m.rowEdit.err = "MongoDB has no DEFAULT value"
		return m, nil
	}
	if len(edit.rows) == 0 {
		m.rowEdit.err = "no rows selected"
		return m, nil
	}
	value := domain.RowValue{
		Column:  edit.column,
		Value:   edit.value,
		Null:    edit.mode == rowNull,
		Default: edit.mode == rowDefault,
		Kind:    edit.kind,
	}
	spec := domain.RowUpdate{Ref: edit.ref, Column: edit.column, Value: value, Rows: edit.rows}
	return m.startRowMutation(rowUpdate, edit.ref, len(edit.rows), func(session *app.Session) (domain.MutationResult, error) {
		return session.UpdateRows(context.Background(), spec)
	})
}

func (m Model) handleRowDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
		case keymap.Confirm:
			spec := domain.RowDelete{Ref: m.rowDelete.ref, Rows: m.rowDelete.rows}
			return m.startRowMutation(rowDelete, m.rowDelete.ref, len(m.rowDelete.rows), func(session *app.Session) (domain.MutationResult, error) {
				return session.DeleteRows(context.Background(), spec)
			})
		}
	}
	return m, nil
}

func (m Model) startRowMutation(action rowAction, ref domain.ObjectRef, count int, run func(*app.Session) (domain.MutationResult, error)) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	m.modal = modalNone
	m.rowGen++
	gen := m.rowGen
	session := m.session
	switch action {
	case rowInsert:
		m.setMessage(levelInfo, "inserting row…")
	case rowUpdate:
		if count > 1 {
			m.setMessage(levelInfo, fmt.Sprintf("updating %d rows…", count))
		} else {
			m.setMessage(levelInfo, "updating row…")
		}
	default:
		if count > 1 {
			m.setMessage(levelInfo, fmt.Sprintf("deleting %d rows…", count))
		} else {
			m.setMessage(levelInfo, "deleting row…")
		}
	}
	return m, func() tea.Msg {
		result, err := run(session)
		return rowDoneMsg{gen: gen, action: action, result: result, ref: ref, count: count, err: err}
	}
}

func (m Model) handleRowDone(msg rowDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.rowGen {
		return m, nil
	}
	if msg.err != nil {
		m.setMessage(levelError, oneLine(msg.err.Error()))
		return m, nil
	}
	m.resetDataFor(msg.ref)
	switch msg.action {
	case rowInsert:
		m.dataNotice = "row inserted · " + oneLine(msg.result.Statement)
	case rowUpdate:
		if msg.count > 1 {
			m.dataNotice = fmt.Sprintf("%d rows updated · %s", msg.count, oneLine(msg.result.Statement))
		} else {
			m.dataNotice = "row updated · " + oneLine(msg.result.Statement)
		}
	default:
		if msg.count > 1 {
			m.dataNotice = fmt.Sprintf("%d rows removed · %s", msg.count, oneLine(msg.result.Statement))
		} else {
			m.dataNotice = "row removed · " + oneLine(msg.result.Statement)
		}
	}
	m.dataNoticeLvl = levelSuccess
	return m, m.ensureData()
}

func (m Model) viewRowInsert() string {
	width := modalWidth(m.width, 72)
	inner := width - 4
	var body []string
	body = append(body, m.styles.Muted.Render("profile ")+m.styles.Text.Render(oneLine(m.profileForStatus().Label)))
	body = append(body, m.styles.Muted.Render("target ")+m.styles.Text.Render(oneLine(dataObjectName(m.rowInsert.ref))))
	if m.rowInsert.mongo {
		body = append(body, m.styles.Header.Render("extended JSON document"))
		body = append(body, m.inputLine("json", m.inlineInput(m.rowInsert.document, m.rowInsert.docPos), true, inner))
		body = append(body, "", m.styles.Muted.Render("Enter insert · Esc cancel"))
		if m.rowInsert.err != "" {
			body = append(body, m.styles.Error.Render(Truncate(m.rowInsert.err, inner)))
		}
		return modalBox("insert document", body, width, m.styles)
	}
	body = append(body, m.styles.Header.Render("columns · Ctrl+N null · Ctrl+D default"))
	start := m.rowInsert.offset
	end := start + 10
	if end > len(m.rowInsert.columns) {
		end = len(m.rowInsert.columns)
	}
	for i := start; i < end; i++ {
		column := m.rowInsert.columns[i]
		mode := rowValue
		if i < len(m.rowInsert.modes) {
			mode = m.rowInsert.modes[i]
		}
		label := "[" + rowModeLabel(mode) + "]"
		value := ""
		if i < len(m.rowInsert.values) {
			value = m.rowInsert.values[i]
		}
		if mode == rowNull {
			value = "NULL"
		} else if mode == rowDefault {
			value = "DEFAULT"
		} else if i == m.rowInsert.cursor {
			value = m.inlineInput(value, m.rowInsert.valuePos)
		}
		line := fmt.Sprintf("%-18s %-10s %s", Truncate(column.Name, 18), label, value)
		if i == m.rowInsert.cursor {
			line = "> " + line
			body = append(body, m.styles.Select.Render(Pad(Truncate(line, inner), inner)))
		} else {
			body = append(body, m.styles.Text.Render(Truncate(line, inner)))
		}
	}
	if m.rowInsert.err != "" {
		body = append(body, m.styles.Error.Render(Truncate(m.rowInsert.err, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "insert", "Tab", "next field"))
	body = append(body, m.styles.Muted.Render("Esc cancel"))
	return modalBox("insert row", body, width, m.styles)
}

func (m Model) viewRowEdit() string {
	width := modalWidth(m.width, 72)
	inner := width - 4
	var body []string
	body = append(body, m.styles.Muted.Render("profile ")+m.styles.Text.Render(oneLine(m.profileForStatus().Label)))
	body = append(body, m.styles.Muted.Render("target ")+m.styles.Text.Render(oneLine(dataObjectName(m.rowEdit.ref))))
	target := "row · " + oneLine(m.rowEdit.label)
	if len(m.rowEdit.rows) > 1 {
		target = fmt.Sprintf("%d rows · %s", len(m.rowEdit.rows), oneLine(m.rowEdit.label))
	}
	body = append(body, m.styles.Muted.Render("update ")+m.styles.Text.Render(Truncate(target, inner)))
	mode := "[" + rowModeLabel(m.rowEdit.mode) + "]"
	value := m.rowEdit.value
	switch m.rowEdit.mode {
	case rowNull:
		value = "NULL"
	case rowDefault:
		value = "DEFAULT"
	default:
		value = m.inlineInput(value, m.rowEdit.valuePos)
	}
	line := fmt.Sprintf("%-22s %-8s %s", Truncate(oneLine(m.rowEdit.label), 22), mode, value)
	body = append(body, m.styles.Select.Render(Pad(Truncate(line, inner), inner)))
	if m.rowEdit.err != "" {
		body = append(body, m.styles.Error.Render(Truncate(m.rowEdit.err, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "update", "Ctrl+N", "null"))
	body = append(body, m.styles.Muted.Render("Ctrl+D default (SQL) · Esc cancel"))
	return modalBox("update cell", body, width, m.styles)
}

func rowModeLabel(mode rowValueMode) string {
	switch mode {
	case rowNull:
		return "NULL"
	case rowDefault:
		return "DEF"
	default:
		return "val"
	}
}

func (m Model) viewRowDelete() string {
	width := modalWidth(m.width, 68)
	inner := width - 4
	var body []string
	body = append(body, m.styles.Muted.Render("profile ")+m.styles.Text.Render(oneLine(m.profileForStatus().Label)))
	count := len(m.rowDelete.rows)
	text := "Delete row from " + oneLine(dataObjectName(m.rowDelete.ref)) + "?"
	if count > 1 {
		text = fmt.Sprintf("Delete %d rows from %s?", count, oneLine(dataObjectName(m.rowDelete.ref)))
	}
	for _, line := range wrapText(text, inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	var labels []string
	for i, row := range m.rowDelete.rows {
		if i == 3 {
			labels = append(labels, "…")
			break
		}
		labels = append(labels, oneLine(domain.RowKeyLabel(row)))
	}
	if label := strings.Join(labels, ", "); label != "" {
		for _, line := range wrapText("keys "+label, inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	}
	body = append(body, "", m.actionLine("Enter", "remove", "Esc", "cancel"))
	return modalBox("delete row", body, width, m.styles)
}
