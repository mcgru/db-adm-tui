package tui

import (
	"context"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type newField int

const (
	fieldEngine newField = iota
	fieldLabel
	fieldHost
	fieldPort
	fieldDatabase
	fieldUser
	fieldAuth
	fieldTLS
	fieldPassword
)

func (m *Model) openNewModal() {
	m.modal = modalNew
	m.newEngine = domain.EnginePostgreSQL
	m.newFocus = fieldEngine
	m.newNeedsPassword = false
	m.newIgnoreSource = false
	m.newBusy = false
	m.newErr = ""
	m.newAllowNoTLS = false
	m.newLabel.SetValue("")
	m.newHost.SetValue("")
	m.newPort.SetValue(strconv.Itoa(m.newEngine.DefaultPort()))
	m.newDatabase.SetValue("")
	m.newUser.SetValue("")
	m.newAuth.SetValue("")
	m.newPassword.SetValue("")
	m.newPassword.Blur()
	m.focusNewField()
}

func (m *Model) closeNewModal() {
	m.modal = modalNone
	m.newEngine = domain.EnginePostgreSQL
	m.newLabel.SetValue("")
	m.newHost.SetValue("")
	m.newPort.SetValue("")
	m.newDatabase.SetValue("")
	m.newUser.SetValue("")
	m.newAuth.SetValue("")
	m.newPassword.SetValue("")
	m.newAllowNoTLS = false
	m.newNeedsPassword = false
	m.newIgnoreSource = false
	m.newBusy = false
	m.newErr = ""
	m.newFocus = fieldEngine
}

func (m *Model) openPasswordModal(p domain.Profile) {
	m.modal = modalPassword
	m.passwordProfile = p
	m.passwordInput.SetValue("")
	m.passwordInput.Focus()
	m.passwordInput.CursorEnd()
	m.passwordBusy = false
	m.message = ""
	m.messageLevel = levelNone
}

func (m *Model) closePasswordModal() {
	m.modal = modalNone
	m.passwordInput.SetValue("")
	m.passwordProfile = domain.Profile{}
	m.passwordBusy = false
}

func (m *Model) focusNewField() {
	m.newLabel.Blur()
	m.newHost.Blur()
	m.newPort.Blur()
	m.newDatabase.Blur()
	m.newUser.Blur()
	m.newAuth.Blur()
	m.newPassword.Blur()
	switch m.newFocus {
	case fieldLabel:
		m.newLabel.Focus()
		m.newLabel.CursorEnd()
	case fieldHost:
		m.newHost.Focus()
		m.newHost.CursorEnd()
	case fieldPort:
		m.newPort.Focus()
		m.newPort.CursorEnd()
	case fieldDatabase:
		m.newDatabase.Focus()
		m.newDatabase.CursorEnd()
	case fieldUser:
		m.newUser.Focus()
		m.newUser.CursorEnd()
	case fieldAuth:
		m.newAuth.Focus()
		m.newAuth.CursorEnd()
	case fieldPassword:
		m.newPassword.Focus()
		m.newPassword.CursorEnd()
	case fieldEngine, fieldTLS:
		// Selectors and checkboxes are not text fields.
	}
}

func (m Model) newFields() []newField {
	fields := []newField{
		fieldEngine, fieldLabel, fieldHost, fieldPort, fieldDatabase, fieldUser,
	}
	if m.newEngine == domain.EngineMongoDB {
		fields = append(fields, fieldAuth)
	}
	if m.newEngine == domain.EnginePostgreSQL {
		fields = append(fields, fieldTLS)
	}
	if m.newNeedsPassword {
		fields = append(fields, fieldPassword)
	}
	return fields
}

func (m Model) nextNewField(delta int) newField {
	fields := m.newFields()
	pos := 0
	for i, f := range fields {
		if f == m.newFocus {
			pos = i
			break
		}
	}
	pos = (pos + delta + len(fields)) % len(fields)
	return fields[pos]
}

// cycleNewEngine keeps the port at the engine default unless the user typed a
// custom value, and resets fields that only apply to the previous engine.
func (m *Model) cycleNewEngine() {
	previous := m.newEngine
	previousPort := strconv.Itoa(previous.DefaultPort())
	switch previous {
	case domain.EnginePostgreSQL:
		m.newEngine = domain.EngineMySQL
	case domain.EngineMySQL:
		m.newEngine = domain.EngineMongoDB
	default:
		m.newEngine = domain.EnginePostgreSQL
	}
	if port := strings.TrimSpace(m.newPort.Value()); port == "" || port == previousPort {
		m.newPort.SetValue(strconv.Itoa(m.newEngine.DefaultPort()))
		m.newPort.CursorEnd()
	}
	if m.newEngine == domain.EngineMongoDB {
		if strings.TrimSpace(m.newAuth.Value()) == "" {
			m.newAuth.SetValue(strings.TrimSpace(m.newDatabase.Value()))
			m.newAuth.CursorEnd()
		}
	} else {
		m.newAuth.SetValue("")
	}
	m.newErr = ""
	m.clearNewPassword()
	m.focusNewField()
}

func (m *Model) clearNewPassword() {
	if !m.newNeedsPassword && !m.newIgnoreSource && m.newPassword.Value() == "" {
		return
	}
	m.newNeedsPassword = false
	m.newIgnoreSource = false
	m.newPassword.SetValue("")
}

func (m Model) newFieldDefinesConnection(field newField) bool {
	switch field {
	case fieldEngine, fieldHost, fieldPort, fieldDatabase, fieldUser, fieldAuth:
		return true
	default:
		return false
	}
}

func (m *Model) updateNewField(msg tea.Msg) bool {
	switch m.newFocus {
	case fieldLabel:
		updated, _ := m.newLabel.Update(msg)
		m.newLabel = updated
	case fieldHost:
		updated, _ := m.newHost.Update(msg)
		m.newHost = updated
	case fieldPort:
		updated, _ := m.newPort.Update(msg)
		m.newPort = updated
	case fieldDatabase:
		updated, _ := m.newDatabase.Update(msg)
		m.newDatabase = updated
	case fieldUser:
		updated, _ := m.newUser.Update(msg)
		m.newUser = updated
	case fieldAuth:
		updated, _ := m.newAuth.Update(msg)
		m.newAuth = updated
	case fieldPassword:
		updated, _ := m.newPassword.Update(msg)
		m.newPassword = updated
	default:
		return false
	}
	return true
}

func (m Model) handleNewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			if m.newBusy {
				m.cancelConnect()
			}
			m.closeNewModal()
			return m, nil
		case keymap.Confirm:
			return m.submitNew()
		case keymap.Tab, keymap.Down:
			if m.newBusy {
				return m, nil
			}
			m.newFocus = m.nextNewField(1)
			m.focusNewField()
			return m, nil
		case keymap.ShiftTab, keymap.Up:
			if m.newBusy {
				return m, nil
			}
			m.newFocus = m.nextNewField(-1)
			m.focusNewField()
			return m, nil
		case keymap.Space:
			if m.newBusy {
				return m, nil
			}
			switch m.newFocus {
			case fieldEngine:
				m.cycleNewEngine()
				return m, nil
			case fieldTLS:
				m.newAllowNoTLS = !m.newAllowNoTLS
				m.newErr = ""
				return m, nil
			}
		}
	}
	if m.newBusy {
		return m, nil
	}
	beforeFocus := m.newFocus
	if !m.updateNewField(msg) {
		return m, nil
	}
	m.newErr = ""
	if m.newFocus != fieldPassword && m.newFieldDefinesConnection(beforeFocus) {
		m.clearNewPassword()
	}
	return m, nil
}

func (m Model) newProfileFromForm() (domain.Profile, newField, error) {
	host := strings.TrimSpace(m.newHost.Value())
	if host == "" {
		return domain.Profile{}, fieldHost, domain.NewError(domain.KindConfig, "new-profile", "enter a host")
	}
	port, err := strconv.Atoi(strings.TrimSpace(m.newPort.Value()))
	if err != nil || port < 1 || port > 65535 {
		return domain.Profile{}, fieldPort, domain.NewError(domain.KindConfig, "new-profile", "port must be a number between 1 and 65535")
	}
	database := strings.TrimSpace(m.newDatabase.Value())
	if database == "" {
		return domain.Profile{}, fieldDatabase, domain.NewError(domain.KindConfig, "new-profile", "enter a database")
	}
	user := strings.TrimSpace(m.newUser.Value())
	if user == "" {
		return domain.Profile{}, fieldUser, domain.NewError(domain.KindConfig, "new-profile", "enter a user")
	}
	auth := strings.TrimSpace(m.newAuth.Value())
	if m.newEngine == domain.EngineMongoDB && auth == "" {
		auth = database
	}
	p := domain.Profile{
		Label:        strings.TrimSpace(m.newLabel.Value()),
		Engine:       m.newEngine,
		Host:         host,
		Port:         port,
		Database:     database,
		AuthDatabase: auth,
		User:         user,
		AllowNoTLS:   m.newEngine == domain.EnginePostgreSQL && m.newAllowNoTLS,
	}.Normalize()
	if err := p.Validate(); err != nil {
		return domain.Profile{}, fieldEngine, err
	}
	return p, fieldEngine, nil
}

func (m Model) submitNew() (tea.Model, tea.Cmd) {
	if m.newBusy {
		return m, nil
	}
	p, focus, err := m.newProfileFromForm()
	if err != nil {
		m.newErr = oneLine(err.Error())
		m.newFocus = focus
		m.focusNewField()
		return m, nil
	}
	p, err = m.svc.PrepareNewProfile(context.Background(), p)
	if err != nil {
		m.newErr = app.URIErrorMessage(err)
		m.newFocus = fieldHost
		m.focusNewField()
		return m, nil
	}
	if m.newIgnoreSource {
		p.Source = nil
	}
	cred := app.Credential{}
	if m.newNeedsPassword {
		cred = app.Credential{Password: m.newPassword.Value(), Provided: true}
	}
	m.newErr = ""
	return m.beginConnect(originNew, p, cred)
}

func (m Model) handlePasswordKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			if m.passwordBusy {
				m.cancelConnect()
			}
			m.closePasswordModal()
			return m, nil
		case keymap.Confirm:
			if m.passwordBusy {
				return m, nil
			}
			cred := app.Credential{Password: m.passwordInput.Value(), Provided: true}
			return m.beginConnect(originPassword, m.passwordProfile, cred)
		}
	}
	if m.passwordBusy {
		return m, nil
	}
	updated, _ := m.passwordInput.Update(msg)
	m.passwordInput = updated
	return m, nil
}

func (m Model) handlePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if m.modal == modalNone && m.screen == ScreenWorkspace && m.focus == FocusTree && m.tree.searchPrompt {
		m.tree.searchPattern, m.tree.searchCursor = insertFilterText(m.tree.searchPattern, m.tree.searchCursor, msg.Content)
		return m, nil
	}
	if m.modal == modalNone && m.screen == ScreenWorkspace && m.focus == FocusQuery && !m.queryRunning {
		updated, cmd := m.query.Update(msg)
		m.query = updated
		if m.completion.active {
			m.refreshCompletion()
		}
		return m, cmd
	}
	if m.modal == modalStatement {
		updated, cmd := m.statementInput.Update(msg)
		m.statementInput = updated
		if m.completion.active {
			m.refreshCompletion()
		}
		return m, cmd
	}
	if m.modal == modalNone {
		switch m.data.prompt {
		case promptBuffer:
			m.data.bufferPattern, m.data.bufferCursor = insertFilterText(m.data.bufferPattern, m.data.bufferCursor, msg.Content)
			m.updateBufferFilter()
			return m, nil
		case promptServer:
			m.data.serverPattern, m.data.serverCursor = insertFilterText(m.data.serverPattern, m.data.serverCursor, msg.Content)
			return m, nil
		}
	}
	if m.modal == modalRowInsert {
		if m.rowInsert.mongo {
			m.rowInsert.document, m.rowInsert.docPos = insertFilterText(
				m.rowInsert.document, m.rowInsert.docPos, msg.Content)
		} else if m.rowInsert.cursor >= 0 && m.rowInsert.cursor < len(m.rowInsert.values) {
			value, pos := insertFilterText(m.rowInsertInputValue(), m.rowInsert.valuePos, msg.Content)
			m.setRowInsertInputValue(value)
			m.rowInsert.valuePos = pos
		}
		return m, nil
	}
	if m.modal == modalRowEdit {
		if m.rowEdit.mode != rowValue {
			m.rowEdit.value = ""
			m.rowEdit.valuePos = 0
			m.rowEdit.mode = rowValue
		}
		m.rowEdit.value, m.rowEdit.valuePos = insertFilterText(
			m.rowEdit.value, m.rowEdit.valuePos, msg.Content)
		m.rowEdit.err = ""
		return m, nil
	}
	switch m.modal {
	case modalNew:
		if m.newBusy {
			return m, nil
		}
		beforeFocus := m.newFocus
		if !m.updateNewField(msg) {
			return m, nil
		}
		m.newErr = ""
		if m.newFocus != fieldPassword && m.newFieldDefinesConnection(beforeFocus) {
			m.clearNewPassword()
		}
		return m, nil
	case modalPassword:
		if m.passwordBusy {
			return m, nil
		}
		updated, cmd := m.passwordInput.Update(msg)
		m.passwordInput = updated
		return m, cmd
	case modalEdit:
		if m.editFocus == editTLS {
			return m, nil
		}
		switch m.editFocus {
		case editHost:
			updated, cmd := m.editHost.Update(msg)
			m.editHost = updated
			return m, cmd
		case editPort:
			updated, cmd := m.editPort.Update(msg)
			m.editPort = updated
			return m, cmd
		case editDatabase:
			updated, cmd := m.editDatabase.Update(msg)
			m.editDatabase = updated
			return m, cmd
		case editUser:
			updated, cmd := m.editUser.Update(msg)
			m.editUser = updated
			return m, cmd
		default:
			updated, cmd := m.editLabel.Update(msg)
			m.editLabel = updated
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleSaveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			return m.finishSave(false)
		case keymap.Confirm:
			return m.finishSave(true)
		}
	}
	return m, nil
}

func (m Model) finishSave(save bool) (tea.Model, tea.Cmd) {
	profile := m.saveProfile
	session := m.saveSession
	result := m.saveResult
	m.modal = modalNone
	m.saveProfile = domain.Profile{}
	m.saveSession = nil
	m.saveResult = app.ConnectResult{}
	if session == nil {
		return m, nil
	}
	if result.Session == nil {
		result = app.ConnectResult{Session: session, Profile: profile}
	}
	m = m.openWorkspace(result)
	if !save {
		m.setMessage(levelSuccess, "connected")
		return m.loadRoot()
	}
	if err := m.svc.SaveProfile(context.Background(), profile); err != nil {
		m.setMessage(levelError, "Could not save "+oneLine(profile.Label)+". The connection is still open.")
		return m.loadRoot()
	}
	m.reloadProfiles(context.Background(), profile.Key())
	m.setMessage(levelSuccess, "saved "+oneLine(profile.Label))
	return m.loadRoot()
}

func (m Model) viewNewConn() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	databaseLabel := "database"
	if m.newEngine == domain.EngineMongoDB {
		databaseLabel = "home.db"
	}
	engineValue := m.newEngine.DisplayName()
	if m.newFocus == fieldEngine {
		engineValue += "  Space cycles"
	}
	body := []string{
		m.inputLine("engine", engineValue, m.newFocus == fieldEngine, inner),
		m.editInfoLine("source", "auto (file when matched, else prompt)", inner),
		m.inputLine("label", m.newLabel.View(), m.newFocus == fieldLabel, inner),
		m.inputLine("host", m.newHost.View(), m.newFocus == fieldHost, inner),
		m.inputLine("port", m.newPort.View(), m.newFocus == fieldPort, inner),
		m.inputLine(databaseLabel, m.newDatabase.View(), m.newFocus == fieldDatabase, inner),
		m.inputLine("user", m.newUser.View(), m.newFocus == fieldUser, inner),
	}
	if m.newEngine == domain.EngineMongoDB {
		body = append(body, m.mutedInputLine("auth.db", m.newAuth.View(), m.newFocus == fieldAuth, inner))
	}
	if m.newEngine == domain.EnginePostgreSQL {
		body = append(body, m.toggleLine("no TLS", "TLS first, then plaintext", m.newAllowNoTLS, m.newFocus == fieldTLS))
	}
	if m.newNeedsPassword {
		body = append(body, m.inputLine("password", m.newPassword.View(), m.newFocus == fieldPassword, inner))
	}
	body = append(body, "")
	for _, line := range wrapText("No password is stored. A matching ~/.pgpass, ~/.mypass, or ~/.mongo.* source is used; otherwise a one-time password is prompted.", inner) {
		body = append(body, m.styles.Muted.Render(line))
	}
	if m.newFocus == fieldEngine {
		body = append(body, m.styles.Muted.Render("Space cycles engine · port follows the default unless edited"))
	}
	if m.newFocus == fieldTLS {
		body = append(body, m.styles.Muted.Render("Space toggles the no-TLS fallback"))
	}
	if m.newErr != "" {
		body = append(body, "")
		for _, line := range wrapText(m.newErr, inner) {
			body = append(body, m.styles.Error.Render(line))
		}
	}
	if m.newBusy {
		body = append(body, "")
		body = append(body, m.styles.Info.Render("connecting…"))
	}
	body = append(body, "", m.actionLine("Enter", "connect", "Esc", "cancel"))
	body = append(body, m.styles.Muted.Render("Tab/↑↓ move between fields"))
	return modalBox("new connection", body, width, m.styles)
}

func (m Model) viewPassword() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	label := oneLine(m.passwordProfile.Label)
	body := []string{m.inputLine("password", m.passwordInput.View(), true, inner)}
	if m.passwordBusy {
		body = append(body, "")
		body = append(body, m.styles.Info.Render("connecting…"))
	}
	body = append(body, "", m.actionLine("Enter", "connect", "Esc", "cancel"))
	return modalBox("Password for "+label, body, width, m.styles)
}

func (m Model) viewSave() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	var body []string
	for _, line := range wrapText("Save "+oneLine(m.saveProfile.Label)+" as a profile?", inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	body = append(body, "", m.actionLine("Enter", "save", "Esc", "skip"))
	return modalBox("save profile", body, width, m.styles)
}

func (m Model) inputLine(label, view string, focused bool, width int) string {
	marker := "  "
	style := m.styles.Muted
	if focused {
		marker = "> "
		style = m.styles.Accent
	}
	labelWidth := 9
	inputWidth := width - len(marker) - labelWidth - 1
	if inputWidth < 1 {
		inputWidth = 1
	}
	view = Truncate(view, inputWidth)
	return style.Render(marker+Pad(label, labelWidth)) + " " + view
}

func (m Model) actionLine(key1, action1, key2, action2 string) string {
	return m.styles.Accent.Render(key1) + " " + m.styles.Muted.Render(action1) + " · " +
		m.styles.Accent.Render(key2) + " " + m.styles.Muted.Render(action2)
}
