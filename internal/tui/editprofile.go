package tui

import (
	"context"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type editField int

const (
	editLabel editField = iota
	editHost
	editPort
	editDatabase
	editUser
	editTLS
)

// openEditModal fills the profile form from the picker selection. Engine and
// source stay fixed; the editable fields are the non-secret connection values.
func (m *Model) openEditModal(p domain.Profile) {
	m.modal = modalEdit
	m.editProfile = p
	m.editOriginalKey = p.Key()
	m.editLabel.SetValue(p.Label)
	m.editHost.SetValue(p.Host)
	m.editPort.SetValue(strconv.Itoa(p.Port))
	m.editDatabase.SetValue(p.Database)
	m.editUser.SetValue(p.User)
	m.editAllowNoTLS = p.AllowNoTLS
	m.editFocus = editLabel
	m.editErr = ""
	m.focusEditField()
}

func (m *Model) closeEditModal() {
	m.modal = modalNone
	m.editProfile = domain.Profile{}
	m.editOriginalKey = ""
	m.editLabel.SetValue("")
	m.editHost.SetValue("")
	m.editPort.SetValue("")
	m.editDatabase.SetValue("")
	m.editUser.SetValue("")
	m.editAllowNoTLS = false
	m.editFocus = editLabel
	m.editErr = ""
}

func (m Model) editFields() []editField {
	fields := []editField{editLabel, editHost, editPort, editDatabase, editUser}
	if m.editProfile.Engine == domain.EnginePostgreSQL {
		fields = append(fields, editTLS)
	}
	return fields
}

func (m *Model) focusEditField() {
	m.editLabel.Blur()
	m.editHost.Blur()
	m.editPort.Blur()
	m.editDatabase.Blur()
	m.editUser.Blur()
	switch m.editFocus {
	case editHost:
		m.editHost.Focus()
		m.editHost.CursorEnd()
	case editPort:
		m.editPort.Focus()
		m.editPort.CursorEnd()
	case editDatabase:
		m.editDatabase.Focus()
		m.editDatabase.CursorEnd()
	case editUser:
		m.editUser.Focus()
		m.editUser.CursorEnd()
	case editTLS:
		// The TLS fallback is a checkbox, not a text field.
	default:
		m.editLabel.Focus()
		m.editLabel.CursorEnd()
	}
}

func (m Model) nextEditField(delta int) editField {
	fields := m.editFields()
	pos := 0
	for i, field := range fields {
		if field == m.editFocus {
			pos = i
			break
		}
	}
	pos = (pos + delta + len(fields)) % len(fields)
	return fields[pos]
}

func (m Model) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.closeEditModal()
			return m, nil
		case keymap.Confirm:
			return m.submitEdit()
		case keymap.Tab:
			m.editFocus = m.nextEditField(1)
			m.focusEditField()
			return m, nil
		case keymap.ShiftTab:
			m.editFocus = m.nextEditField(-1)
			m.focusEditField()
			return m, nil
		case keymap.Space:
			if m.editFocus == editTLS {
				m.editAllowNoTLS = !m.editAllowNoTLS
				return m, nil
			}
		}
	}
	if m.editFocus == editTLS {
		return m, nil
	}
	switch m.editFocus {
	case editHost:
		updated, _ := m.editHost.Update(msg)
		m.editHost = updated
	case editPort:
		updated, _ := m.editPort.Update(msg)
		m.editPort = updated
	case editDatabase:
		updated, _ := m.editDatabase.Update(msg)
		m.editDatabase = updated
	case editUser:
		updated, _ := m.editUser.Update(msg)
		m.editUser = updated
	default:
		updated, _ := m.editLabel.Update(msg)
		m.editLabel = updated
	}
	return m, nil
}

func (m Model) submitEdit() (tea.Model, tea.Cmd) {
	p := m.editProfile
	p.Label = strings.TrimSpace(m.editLabel.Value())
	p.Host = strings.TrimSpace(m.editHost.Value())
	port, err := strconv.Atoi(strings.TrimSpace(m.editPort.Value()))
	if err != nil {
		m.editErr = "port must be a number"
		m.editFocus = editPort
		m.focusEditField()
		return m, nil
	}
	p.Port = port
	p.Database = strings.TrimSpace(m.editDatabase.Value())
	p.User = strings.TrimSpace(m.editUser.Value())
	if p.Engine == domain.EnginePostgreSQL {
		p.AllowNoTLS = m.editAllowNoTLS
	}
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		m.editErr = oneLine(err.Error())
		return m, nil
	}
	originalKey := m.editOriginalKey
	if err := m.svc.UpdateProfile(context.Background(), originalKey, p); err != nil {
		m.editErr = oneLine(err.Error())
		return m, nil
	}
	m.closeEditModal()
	delete(m.rowErrors, originalKey)
	delete(m.rowErrors, p.Key())
	m.reloadProfiles(context.Background(), p.Key())
	m.setMessage(levelSuccess, "saved "+oneLine(p.Label))
	return m, nil
}

func (m Model) openDeleteProfileModal() (tea.Model, tea.Cmd) {
	if m.screen != ScreenPicker || m.connecting {
		return m, nil
	}
	p, ok := m.currentPickerProfile()
	if !ok {
		m.setMessage(levelInfo, "select a profile to delete")
		return m, nil
	}
	m.deleteProfile = p
	m.deleteErr = ""
	m.modal = modalDeleteProfile
	return m, nil
}

func (m *Model) closeDeleteProfileModal() {
	m.modal = modalNone
	m.deleteProfile = domain.Profile{}
	m.deleteErr = ""
}

func (m Model) handleDeleteProfileKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.closeDeleteProfileModal()
	case keymap.Confirm:
		key := m.deleteProfile.Key()
		label := m.deleteProfile.Label
		if err := m.svc.DeleteProfile(context.Background(), key); err != nil {
			m.deleteErr = oneLine(err.Error())
			return m, nil
		}
		m.closeDeleteProfileModal()
		delete(m.rowErrors, key)
		m.reloadProfiles(context.Background(), "")
		m.setMessage(levelSuccess, "profile deleted · "+oneLine(label)+" · credential source kept")
	}
	return m, nil
}

func (m Model) viewDeleteProfile() string {
	width := modalWidth(m.width, 64)
	inner := width - 4
	var body []string
	text := "Delete profile " + oneLine(m.deleteProfile.Label) + "?"
	for _, line := range wrapText(text, inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	source := oneLine(m.deleteProfile.SourceLabel())
	if source != "" {
		for _, line := range wrapText("source "+source+" is kept on disk", inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	}
	body = append(body, m.styles.Muted.Render("Only the profile entry is removed."))
	if m.deleteErr != "" {
		body = append(body, "", m.styles.Error.Render(Truncate(m.deleteErr, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "delete", "Esc", "cancel"))
	return modalBox("delete profile", body, width, m.styles)
}

func (m Model) viewEdit() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	body := []string{
		m.editInfoLine("engine", m.editProfile.Engine.DisplayName(), inner),
		m.editInfoLine("source", m.editProfile.SourceLabel(), inner),
		m.inputLine("label", m.editLabel.View(), m.editFocus == editLabel, inner),
		m.inputLine("host", m.editHost.View(), m.editFocus == editHost, inner),
		m.inputLine("port", m.editPort.View(), m.editFocus == editPort, inner),
		m.inputLine("database", m.editDatabase.View(), m.editFocus == editDatabase, inner),
		m.inputLine("user", m.editUser.View(), m.editFocus == editUser, inner),
	}
	if m.editProfile.Engine == domain.EnginePostgreSQL {
		body = append(body, m.editToggleLine())
	}
	if m.editErr != "" {
		body = append(body, "")
		for _, line := range wrapText(m.editErr, inner) {
			body = append(body, m.styles.Error.Render(line))
		}
	}
	if m.editFocus == editTLS {
		body = append(body, m.styles.Muted.Render("Space toggles the no-TLS fallback"))
	}
	body = append(body, "", m.actionLine("Enter", "save", "Esc", "cancel"))
	return modalBox("edit profile", body, width, m.styles)
}

func (m Model) editInfoLine(label, value string, width int) string {
	const labelWidth = 9
	valueWidth := width - 2 - labelWidth - 1
	if valueWidth < 1 {
		valueWidth = 1
	}
	return m.styles.Muted.Render("  "+Pad(label, labelWidth)) + " " +
		m.styles.Text.Render(Truncate(oneLine(value), valueWidth))
}

func (m Model) editToggleLine() string {
	const labelWidth = 9
	marker := "  "
	style := m.styles.Muted
	if m.editFocus == editTLS {
		marker = "> "
		style = m.styles.Accent
	}
	check := "[ ]"
	if m.editAllowNoTLS {
		check = "[x]"
	}
	return style.Render(marker+Pad("no TLS", labelWidth)) + " " +
		m.styles.Text.Render(check) +
		m.styles.Muted.Render("  TLS first, then plaintext")
}
