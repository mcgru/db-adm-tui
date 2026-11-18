package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type newField int

const (
	fieldURI newField = iota
	fieldLabel
	fieldPassword
)

func (m *Model) openNewModal() {
	m.modal = modalNew
	m.newFocus = fieldURI
	m.newNeedsPassword = false
	m.newIgnoreSource = false
	m.newBusy = false
	m.newErr = ""
	m.newURI.SetValue("")
	m.newLabel.SetValue("")
	m.newPassword.SetValue("")
	m.newPassword.Blur()
	m.newLabel.Blur()
	m.newURI.Focus()
	m.newURI.CursorEnd()
}

func (m *Model) closeNewModal() {
	m.modal = modalNone
	m.newURI.SetValue("")
	m.newLabel.SetValue("")
	m.newPassword.SetValue("")
	m.newNeedsPassword = false
	m.newIgnoreSource = false
	m.newBusy = false
	m.newErr = ""
	m.newFocus = fieldURI
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
	m.newURI.Blur()
	m.newLabel.Blur()
	m.newPassword.Blur()
	switch m.newFocus {
	case fieldLabel:
		m.newLabel.Focus()
		m.newLabel.CursorEnd()
	case fieldPassword:
		m.newPassword.Focus()
		m.newPassword.CursorEnd()
	default:
		m.newURI.Focus()
		m.newURI.CursorEnd()
	}
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
		case keymap.Tab:
			if m.newBusy {
				return m, nil
			}
			m.newFocus = m.nextNewField(1)
			m.focusNewField()
			return m, nil
		case keymap.ShiftTab:
			if m.newBusy {
				return m, nil
			}
			m.newFocus = m.nextNewField(-1)
			m.focusNewField()
			return m, nil
		}
	}
	if m.newBusy {
		return m, nil
	}
	beforeURI := m.newURI.Value()
	switch m.newFocus {
	case fieldLabel:
		updated, _ := m.newLabel.Update(msg)
		m.newLabel = updated
	case fieldPassword:
		updated, _ := m.newPassword.Update(msg)
		m.newPassword = updated
	default:
		updated, _ := m.newURI.Update(msg)
		m.newURI = updated
	}
	if m.newFocus == fieldURI && m.newURI.Value() != beforeURI && m.newNeedsPassword {
		m.newNeedsPassword = false
		m.newIgnoreSource = false
		m.newPassword.SetValue("")
		m.newErr = ""
		m.focusNewField()
	}
	return m, nil
}

func (m Model) nextNewField(delta int) newField {
	fields := []newField{fieldURI, fieldLabel}
	if m.newNeedsPassword {
		fields = append(fields, fieldPassword)
	}
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

func (m Model) submitNew() (tea.Model, tea.Cmd) {
	if m.newBusy {
		return m, nil
	}
	p, err := m.svc.PrepareNewConnection(context.Background(), m.newURI.Value(), m.newLabel.Value())
	if err != nil {
		m.newErr = app.URIErrorMessage(err)
		m.newFocus = fieldURI
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
	switch m.modal {
	case modalNew:
		if m.newBusy {
			return m, nil
		}
		if m.newFocus == fieldURI && m.newNeedsPassword {
			m.newNeedsPassword = false
			m.newIgnoreSource = false
			m.newPassword.SetValue("")
			m.newErr = ""
			m.focusNewField()
		}
		switch m.newFocus {
		case fieldLabel:
			updated, cmd := m.newLabel.Update(msg)
			m.newLabel = updated
			return m, cmd
		case fieldPassword:
			updated, cmd := m.newPassword.Update(msg)
			m.newPassword = updated
			return m, cmd
		default:
			updated, cmd := m.newURI.Update(msg)
			m.newURI = updated
			return m, cmd
		}
	case modalPassword:
		if m.passwordBusy {
			return m, nil
		}
		updated, cmd := m.passwordInput.Update(msg)
		m.passwordInput = updated
		return m, cmd
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
	m.modal = modalNone
	m.saveProfile = domain.Profile{}
	m.saveSession = nil
	if session == nil {
		return m, nil
	}
	m = m.openWorkspace(app.ConnectResult{Session: session, Profile: profile})
	if !save {
		m.setMessage(levelSuccess, "connected · catalog preview")
		return m, nil
	}
	if err := m.svc.SaveProfile(context.Background(), profile); err != nil {
		m.setMessage(levelError, "Could not save "+oneLine(profile.Label)+". The connection is still open.")
		return m, nil
	}
	m.reloadProfiles(context.Background(), profile.Key())
	m.setMessage(levelSuccess, "saved "+oneLine(profile.Label))
	return m, nil
}

func (m Model) viewNewConn() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	var body []string
	body = append(body, m.inputLine("uri", m.uriInputView(), m.newFocus == fieldURI, inner))
	body = append(body, m.inputLine("label", m.newLabel.View(), m.newFocus == fieldLabel, inner))
	if m.newNeedsPassword {
		body = append(body, m.inputLine("password", m.newPassword.View(), m.newFocus == fieldPassword, inner))
	}
	body = append(body, "")
	for _, line := range wrapText("No password in the URI. It is read from ~/.pgpass or ~/.mongo.*, or prompted once.", inner) {
		body = append(body, m.styles.Muted.Render(line))
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

func (m Model) uriInputView() string {
	value := m.newURI.Value()
	if redacted := domain.Redact(value); redacted != value {
		return m.styles.Text.Render(redacted) + m.styles.Accent.Render(" ")
	}
	return m.newURI.View()
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
