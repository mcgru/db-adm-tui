package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type rowAction int

const (
	rowInsert rowAction = iota
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

type rowDeleteState struct {
	ref  domain.ObjectRef
	keys []domain.RowValue
}

type rowDoneMsg struct {
	gen    uint64
	action rowAction
	result domain.MutationResult
	ref    domain.ObjectRef
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
	row, ok := m.currentDataRow()
	if !ok {
		m.setMessage(levelInfo, "select a loaded row first")
		return m, nil
	}
	keys, ok := m.rowKeys(row)
	if !ok {
		m.setMessage(levelError, "the current row has no complete key")
		return m, nil
	}
	for _, key := range keys {
		if key.Kind == domain.CellBinary {
			m.setMessage(levelError, "binary key values cannot be used for deletes")
			return m, nil
		}
	}
	m.rowDelete = rowDeleteState{ref: m.data.ref, keys: keys}
	m.modal = modalRowDelete
	return m, nil
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
	names := splitRowKey(m.data.orderKey)
	if len(names) == 0 {
		return nil, false
	}
	index := make(map[string]int, len(m.data.columns))
	for i, column := range m.data.columns {
		index[column.Name] = i
	}
	keys := make([]domain.RowValue, 0, len(names))
	for _, name := range names {
		column, ok := index[name]
		if !ok || column >= len(row.Cells) {
			return nil, false
		}
		cell := row.Cells[column]
		if cell.Null {
			return nil, false
		}
		keys = append(keys, domain.RowValue{
			Column: name, Value: cell.Text, Kind: cell.Kind,
		})
	}
	return keys, true
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
		return m.startRowMutation(rowInsert, ins.ref, func(session *app.Session) (domain.MutationResult, error) {
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
	return m.startRowMutation(rowInsert, ins.ref, func(session *app.Session) (domain.MutationResult, error) {
		return session.InsertRow(context.Background(), domain.RowInsert{Ref: ins.ref, Values: values})
	})
}

func (m Model) handleRowDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
		case keymap.Confirm:
			spec := domain.RowDelete{Ref: m.rowDelete.ref, Keys: m.rowDelete.keys}
			return m.startRowMutation(rowDelete, m.rowDelete.ref, func(session *app.Session) (domain.MutationResult, error) {
				return session.DeleteRow(context.Background(), spec)
			})
		}
	}
	return m, nil
}

func (m Model) startRowMutation(action rowAction, ref domain.ObjectRef, run func(*app.Session) (domain.MutationResult, error)) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	m.modal = modalNone
	m.rowGen++
	gen := m.rowGen
	session := m.session
	if action == rowInsert {
		m.setMessage(levelInfo, "inserting row…")
	} else {
		m.setMessage(levelInfo, "deleting row…")
	}
	return m, func() tea.Msg {
		result, err := run(session)
		return rowDoneMsg{gen: gen, action: action, result: result, ref: ref, err: err}
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
	if msg.action == rowInsert {
		m.dataNotice = "row inserted · " + oneLine(msg.result.Statement)
	} else {
		m.dataNotice = "row removed · " + oneLine(msg.result.Statement)
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
	text := "Delete row from " + oneLine(dataObjectName(m.rowDelete.ref)) + "?"
	for _, line := range wrapText(text, inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	if label := oneLine(domain.RowKeyLabel(m.rowDelete.keys)); label != "" {
		for _, line := range wrapText("key "+label, inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	}
	body = append(body, "", m.actionLine("Enter", "remove", "Esc", "cancel"))
	return modalBox("delete row", body, width, m.styles)
}
