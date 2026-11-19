package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

func (m Model) modeBadge() string {
	if m.session == nil || m.session.ReadOnly() {
		return m.styles.ModeRO.Render("r/o")
	}
	return m.styles.ModeRW.Render("r/w")
}

func (m *Model) openModeConfirm() {
	m.completion = completionState{}
	m.modal = modalMode
}

func (m Model) handleModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.modal = modalNone
		return m, nil
	case keymap.Confirm:
		m.modal = modalNone
		if m.session == nil || !m.session.ReadOnly() {
			return m, nil
		}
		return m, m.startModeSwitch(false)
	}
	return m, nil
}

func (m *Model) startModeSwitch(readOnly bool) tea.Cmd {
	if m.session == nil || m.modeBusy {
		return nil
	}
	m.modeBusy = true
	m.modeGen++
	gen := m.modeGen
	if readOnly {
		m.setMessage(levelInfo, "switching to read-only…")
	} else {
		m.setMessage(levelInfo, "switching to read-write…")
	}
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		err := session.SetReadOnly(ctx, readOnly)
		return modeDoneMsg{gen: gen, readOnly: readOnly, err: err}
	}
}

func (m Model) handleModeDone(msg modeDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.modeGen {
		return m, nil
	}
	m.modeBusy = false
	if msg.err != nil {
		m.setMessage(levelError, "Could not switch mode: "+oneLine(msg.err.Error()))
		return m, nil
	}
	if msg.readOnly {
		m.setMessage(levelSuccess, "read-only mode on. Write statements are blocked.")
		return m, nil
	}
	m.setMessage(levelSuccess, "read-write mode on. Read-only guard is off.")
	return m, nil
}

func (m Model) viewModeConfirm() string {
	width := modalWidth(m.width, 64)
	inner := width - 4
	var body []string
	text := "Enable read-write mode? Writes will run on " + oneLine(m.profileForStatus().Label) + "."
	for _, line := range wrapText(text, inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	if m.elevated || m.rootLaunch {
		warning := "This account has elevated rights. Read-only mode is a product guard, not a security boundary."
		if !m.elevated && m.rootLaunch {
			warning = "The tool is running as root. Read-only mode is a product guard, not a security boundary."
		}
		body = append(body, "")
		for _, line := range wrapText(warning, inner) {
			body = append(body, m.styles.Warning.Render(line))
		}
	}
	body = append(body, "", m.actionLine("Enter", "enable", "Esc", "cancel"))
	return modalBox("read-write mode", body, width, m.styles)
}
