package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type userAction int

const (
	userCreate userAction = iota
	userDrop
)

type userState struct {
	name     textinput.Model
	host     textinput.Model
	database textinput.Model
	password textinput.Model
	focus    int
	formErr  string

	confirm     domain.RoleRef
	saveProfile domain.Profile
}

type userDoneMsg struct {
	gen      uint64
	action   userAction
	result   domain.UserResult
	name     string
	host     string
	database string
	err      error
}

func newUserState() userState {
	return userState{
		name:     newInput("user", false),
		host:     newInput("%", false),
		database: newInput("database", false),
		password: newInput("password", true),
	}
}

func (m *Model) userWritable() bool {
	if m.session == nil {
		m.setMessage(levelInfo, "no active connection")
		return false
	}
	if m.session.ReadOnly() {
		m.setMessage(levelError, "Blocked: user changes require read-write mode. Ctrl+X Ctrl+W to switch.")
		return false
	}
	return true
}

func (m Model) startUserAdd() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok || node.Kind != domain.KindUsers {
		m.setMessage(levelInfo, "select the users group to create a user")
		return m, nil
	}
	if !m.userWritable() {
		return m, nil
	}
	m.user = newUserState()
	m.user.host.SetValue("%")
	if m.engine() == domain.EngineMongoDB {
		m.user.database.SetValue(m.currentDatabase())
	}
	m.user.focus = 0
	m.focusUserField()
	m.modal = modalUserForm
	return m, nil
}

func (m Model) startUserRemove() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok || node.Kind != domain.KindUser {
		m.setMessage(levelInfo, "select a user to drop")
		return m, nil
	}
	if !m.userWritable() {
		return m, nil
	}
	m.user.confirm = roleRefFromNode(node)
	m.modal = modalUserConfirm
	return m, nil
}

func (m Model) userFields() []int {
	fields := []int{0}
	switch m.engine() {
	case domain.EngineMySQL:
		fields = append(fields, 1, 2)
	case domain.EngineMongoDB:
		fields = append(fields, 2, 3)
	default:
		fields = append(fields, 2)
	}
	return fields
}

func (m *Model) focusUserField() {
	m.user.name.Blur()
	m.user.host.Blur()
	m.user.database.Blur()
	m.user.password.Blur()
	switch m.user.focus {
	case 0:
		m.user.name.Focus()
		m.user.name.CursorEnd()
	case 1:
		m.user.host.Focus()
		m.user.host.CursorEnd()
	case 2:
		m.user.password.Focus()
		m.user.password.CursorEnd()
	case 3:
		m.user.database.Focus()
		m.user.database.CursorEnd()
	}
}

func (m *Model) nextUserField(delta int) {
	fields := m.userFields()
	pos := 0
	for i, field := range fields {
		if field == m.user.focus {
			pos = i
			break
		}
	}
	pos = (pos + delta + len(fields)) % len(fields)
	m.user.focus = fields[pos]
	m.focusUserField()
}

func (m Model) handleUserFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			m.user = newUserState()
			return m, nil
		case keymap.Tab:
			m.nextUserField(1)
			return m, nil
		case keymap.ShiftTab:
			m.nextUserField(-1)
			return m, nil
		case keymap.Confirm:
			return m.submitUserForm()
		}
	}
	switch m.user.focus {
	case 0:
		updated, _ := m.user.name.Update(msg)
		m.user.name = updated
	case 1:
		updated, _ := m.user.host.Update(msg)
		m.user.host = updated
	case 2:
		updated, _ := m.user.password.Update(msg)
		m.user.password = updated
	case 3:
		updated, _ := m.user.database.Update(msg)
		m.user.database = updated
	}
	return m, nil
}

func (m Model) submitUserForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.user.name.Value())
	host := strings.TrimSpace(m.user.host.Value())
	password := m.user.password.Value()
	if name == "" {
		m.user.formErr = "enter a user name"
		m.user.focus = 0
		m.focusUserField()
		return m, nil
	}
	if password == "" {
		m.user.formErr = "enter a password"
		m.user.focus = 2
		m.focusUserField()
		return m, nil
	}
	if m.engine() == domain.EngineMySQL && host == "" {
		host = "%"
	}
	database := m.currentDatabase()
	if m.engine() == domain.EngineMongoDB {
		database = strings.TrimSpace(m.user.database.Value())
		if database == "" {
			m.user.formErr = "enter the MongoDB database for the user"
			m.user.focus = 3
			m.focusUserField()
			return m, nil
		}
	}
	spec := domain.UserSpec{
		Engine: m.engine(), Database: database, Name: name, Host: host, Password: password,
	}
	m.user.password.SetValue("")
	return m.startUserMutation(userCreate, name, host, database, func(session *app.Session) (domain.UserResult, error) {
		return session.CreateUser(context.Background(), spec)
	})
}

func (m Model) handleUserConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
		case keymap.Confirm:
			user := m.user.confirm
			database := user.Database
			return m.startUserMutation(userDrop, user.Name, "", database, func(session *app.Session) (domain.UserResult, error) {
				result, err := session.DropUser(context.Background(), user)
				return domain.UserResult{Mutation: result}, err
			})
		}
	}
	return m, nil
}

func (m Model) startUserMutation(action userAction, name, host, database string, run func(*app.Session) (domain.UserResult, error)) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	m.modal = modalNone
	m.userGen++
	gen := m.userGen
	session := m.session
	if action == userCreate {
		m.setMessage(levelInfo, "creating user…")
	} else {
		m.setMessage(levelInfo, "dropping user…")
	}
	return m, func() tea.Msg {
		result, err := run(session)
		return userDoneMsg{
			gen: gen, action: action, result: result,
			name: name, host: host, database: database, err: err,
		}
	}
}

func (m Model) handleUserDone(msg userDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.userGen {
		return m, nil
	}
	if msg.err != nil {
		commands := []tea.Cmd{}
		if msg.action == userCreate {
			commands = append(commands, m.refreshPrincipalGroup(domain.KindUsers))
		}
		m.setMessage(levelError, oneLine(msg.err.Error()))
		return m, tea.Batch(commands...)
	}
	commands := []tea.Cmd{m.refreshPrincipalGroup(domain.KindUsers)}
	if msg.action == userCreate {
		m.setMessage(levelSuccess, "user created · "+oneLine(msg.result.Mutation.Statement))
		if msg.result.Source != nil {
			m.user.saveProfile = m.userProfile(msg.name, msg.database, msg.result.Source)
			m.modal = modalUserSave
		}
	} else {
		m.setMessage(levelSuccess, "user removed · "+oneLine(msg.result.Mutation.Statement))
	}
	return m, tea.Batch(commands...)
}

func (m Model) userProfile(name, database string, source *domain.Source) domain.Profile {
	p := m.profileForStatus()
	user := name
	if p.Engine == domain.EngineMySQL {
		if index := strings.LastIndex(user, "@"); index >= 0 {
			user = user[:index]
		}
	}
	label := user + "@" + database
	if database == "" {
		label = user
	}
	profile := domain.Profile{
		Label:      label,
		Engine:     p.Engine,
		Host:       p.Host,
		Port:       p.Port,
		Database:   database,
		User:       user,
		AllowNoTLS: p.AllowNoTLS,
		Source:     source,
	}
	return profile
}

func (m Model) handleUserSaveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			m.user.saveProfile = domain.Profile{}
		case keymap.Confirm:
			profile := m.user.saveProfile
			m.modal = modalNone
			m.user.saveProfile = domain.Profile{}
			if err := m.svc.SaveProfile(context.Background(), profile); err != nil {
				m.setMessage(levelError, oneLine(err.Error()))
				return m, nil
			}
			m.reloadProfiles(context.Background(), profile.Key())
			m.setMessage(levelSuccess, "saved "+oneLine(profile.Label))
		}
	}
	return m, nil
}

func (m Model) viewUserForm() string {
	width := modalWidth(m.width, 64)
	inner := width - 4
	var body []string
	body = append(body, m.styles.Muted.Render("engine ")+m.styles.Text.Render(m.engine().DisplayName()))
	if m.engine() != domain.EngineMongoDB {
		body = append(body, m.styles.Muted.Render("database ")+m.styles.Text.Render(oneLine(m.currentDatabase())))
	}
	body = append(body, m.inputLine("name", m.user.name.View(), m.user.focus == 0, inner))
	if m.engine() == domain.EngineMySQL {
		body = append(body, m.inputLine("host", m.user.host.View(), m.user.focus == 1, inner))
	}
	body = append(body, m.inputLine("password", m.user.password.View(), m.user.focus == 2, inner))
	if m.engine() == domain.EngineMongoDB {
		body = append(body, m.inputLine("database", m.user.database.View(), m.user.focus == 3, inner))
		for _, line := range wrapText("auth database: stores the MongoDB user; it does not grant access", inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	}
	for _, line := range wrapText("roles: after create use Ctrl+X Ctrl+R → 1 Role × Principal", inner) {
		body = append(body, m.styles.Muted.Render(line))
	}
	if m.user.formErr != "" {
		body = append(body, "", m.styles.Error.Render(Truncate(m.user.formErr, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "create", "Tab", "next field"))
	body = append(body, m.styles.Muted.Render("Esc cancel"))
	return modalBox("new user", body, width, m.styles)
}

func (m Model) viewUserConfirm() string {
	width := modalWidth(m.width, 64)
	inner := width - 4
	var body []string
	text := "Drop user " + oneLine(m.user.confirm.Name) + "? Native dependencies or ownership may block it."
	for _, line := range wrapText(text, inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	body = append(body, "", m.actionLine("Enter", "remove", "Esc", "cancel"))
	return modalBox("drop user", body, width, m.styles)
}

func (m Model) viewUserSave() string {
	width := modalWidth(m.width, 64)
	inner := width - 4
	var body []string
	for _, line := range wrapText("Save "+oneLine(m.user.saveProfile.Label)+" as a profile?", inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	body = append(body, "", m.actionLine("Enter", "save", "Esc", "skip"))
	return modalBox("save profile", body, width, m.styles)
}
