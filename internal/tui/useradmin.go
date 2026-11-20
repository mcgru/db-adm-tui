package tui

import (
	"context"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/profile"
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
	hosts    textinput.Model
	password textinput.Model
	focus    int
	formErr  string

	confirm     domain.RoleRef
	saveProfile domain.Profile
}

type userHomeState struct {
	user    domain.RoleRef
	profile domain.Profile
	choices []string
	index   int
	err     string
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
		hosts:    newInput("host1:27017,host2:27018", false),
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
	if !ok || (node.Kind != domain.KindUsers && node.Kind != domain.KindSecurity) {
		m.setMessage(levelInfo, "select Security or the users group to create a user")
		return m, nil
	}
	if !m.userWritable() {
		return m, nil
	}
	m.user = newUserState()
	m.user.host.SetValue("%")
	if m.engine() == domain.EngineMongoDB {
		m.user.database.SetValue("admin")
		m.user.hosts.SetValue(m.mongoProfileHosts())
	}
	m.user.focus = 0
	m.focusUserField()
	m.modal = modalUserForm
	return m, nil
}

func (m Model) startUserHome() (tea.Model, tea.Cmd) {
	if m.engine() != domain.EngineMongoDB {
		m.setMessage(levelInfo, "home.db is available for MongoDB profiles")
		return m, nil
	}
	node, ok := m.tree.selectedNode()
	if !ok || node.Kind != domain.KindUser {
		m.setMessage(levelInfo, "select a MongoDB user to set home.db")
		return m, nil
	}
	name := node.Name
	authDatabase := node.Ref.Database
	if index := strings.LastIndex(name, "@"); index >= 0 {
		if authDatabase == "" {
			authDatabase = name[index+1:]
		}
		name = name[:index]
	}
	var match domain.Profile
	found := false
	for _, view := range m.profiles {
		p := view.Profile
		if p.Engine != domain.EngineMongoDB || p.User != name {
			continue
		}
		if p.EffectiveAuthDatabase() == authDatabase {
			match = p
			found = true
			break
		}
		if !found {
			match = p
			found = true
		}
	}
	if !found {
		m.setMessage(levelInfo, "no saved profile for "+name+"; save a profile first")
		return m, nil
	}
	choices := m.mongoUserDatabases()
	if match.Database != "" {
		present := false
		for _, choice := range choices {
			if choice == match.Database {
				present = true
				break
			}
		}
		if !present {
			choices = append(choices, match.Database)
		}
	}
	index := 0
	for i, choice := range choices {
		if choice == match.Database {
			index = i
			break
		}
	}
	m.userHome = userHomeState{
		user:    domain.RoleRef{Engine: domain.EngineMongoDB, Database: authDatabase, Name: name, Kind: domain.KindUser},
		profile: match,
		choices: choices,
		index:   index,
	}
	m.modal = modalUserHome
	return m, nil
}

func (m *Model) closeUserHome() {
	m.modal = modalNone
	m.userHome = userHomeState{}
}

func (m Model) handleUserHomeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.closeUserHome()
	case keymap.Space:
		if len(m.userHome.choices) > 0 {
			m.userHome.index = (m.userHome.index + 1) % len(m.userHome.choices)
		}
	case keymap.Confirm:
		if len(m.userHome.choices) == 0 {
			m.userHome.err = "no databases available"
			return m, nil
		}
		updated := m.userHome.profile
		authDatabase := updated.EffectiveAuthDatabase()
		updated.Database = m.userHome.choices[m.userHome.index]
		updated.AuthDatabase = authDatabase
		if err := m.svc.UpdateProfile(context.Background(), updated.Key(), updated); err != nil {
			m.userHome.err = oneLine(err.Error())
			return m, nil
		}
		label := updated.Label
		m.closeUserHome()
		m.reloadProfiles(context.Background(), "")
		m.setMessage(levelSuccess, "home.db saved · "+oneLine(label)+" · "+updated.Database)
	}
	return m, nil
}

func (m Model) viewUserHome() string {
	width := modalWidth(m.width, 62)
	inner := width - 4
	var body []string
	body = append(body, m.styles.Muted.Render("user ")+m.styles.Text.Render(oneLine(m.userHome.user.Name+"@"+m.userHome.user.Database)))
	body = append(body, m.styles.Muted.Render("profile ")+m.styles.Text.Render(oneLine(m.userHome.profile.Label)))
	body = append(body, m.styles.Muted.Render("auth.db ")+m.styles.Text.Render(oneLine(m.userHome.profile.EffectiveAuthDatabase())))
	if len(m.userHome.choices) > 0 {
		body = append(body, m.styles.Header.Render("home.db · Space cycles"))
		body = append(body, m.styles.Select.Render(Pad(Truncate("> "+m.userHome.choices[m.userHome.index], inner), inner)))
		for _, line := range wrapText(strings.Join(m.userHome.choices, ", "), inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	}
	if m.userHome.err != "" {
		body = append(body, m.styles.Error.Render(Truncate(m.userHome.err, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "save", "Space", "cycle"))
	body = append(body, m.styles.Muted.Render("Esc cancel"))
	return modalBox("home database", body, width, m.styles)
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
		fields = append(fields, 2, 3, 4)
	default:
		fields = append(fields, 2)
	}
	return fields
}

func (m *Model) focusUserField() {
	m.user.name.Blur()
	m.user.host.Blur()
	m.user.database.Blur()
	m.user.hosts.Blur()
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
	case 4:
		m.user.hosts.Focus()
		m.user.hosts.CursorEnd()
	}
}

// mongoUserDatabases returns the auth-database choices: admin first, then the
// databases loaded in the tree. Complex custom targets can still be typed.
func (m Model) mongoUserDatabases() []string {
	seen := map[string]bool{"admin": true}
	choices := []string{"admin"}
	for _, key := range m.tree.roots {
		entry, ok := m.tree.node(key)
		if !ok || entry.Node.Kind != domain.KindDatabase {
			continue
		}
		name := strings.TrimSpace(entry.Node.Ref.Database)
		if name == "" {
			name = strings.TrimSpace(entry.Node.Name)
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		choices = append(choices, name)
	}
	sort.Strings(choices[1:])
	return choices
}

func (m Model) mongoProfileHosts() string {
	p := m.profileForStatus()
	if p.Source != nil {
		if hosts := profile.MongoSelectorHosts(p.Source.Selector); len(hosts) > 0 {
			return strings.Join(hosts, ",")
		}
	}
	return p.Addr()
}

func (m *Model) cycleUserDatabase() {
	choices := m.mongoUserDatabases()
	if len(choices) == 0 {
		return
	}
	current := strings.TrimSpace(m.user.database.Value())
	next := choices[0]
	for i, choice := range choices {
		if choice == current {
			next = choices[(i+1)%len(choices)]
			break
		}
	}
	m.user.database.SetValue(next)
	m.user.database.CursorEnd()
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
		if action == keymap.Space && m.user.focus == 3 && m.engine() == domain.EngineMongoDB {
			m.cycleUserDatabase()
			return m, nil
		}
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
	case 4:
		updated, _ := m.user.hosts.Update(msg)
		m.user.hosts = updated
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
	mongoHosts := ""
	if m.engine() == domain.EngineMongoDB {
		database = strings.TrimSpace(m.user.database.Value())
		if database == "" {
			m.user.formErr = "enter the MongoDB database for the user"
			m.user.focus = 3
			m.focusUserField()
			return m, nil
		}
		hosts, err := profile.ParseMongoHosts(m.user.hosts.Value())
		if err != nil {
			m.user.formErr = err.Error()
			m.user.focus = 4
			m.focusUserField()
			return m, nil
		}
		mongoHosts = strings.Join(hosts, ",")
	}
	spec := domain.UserSpec{
		Engine: m.engine(), Database: database, Name: name, Host: host,
		MongoHosts: mongoHosts, Password: password,
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
	if p.Engine == domain.EngineMongoDB {
		profile.AuthDatabase = database
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
		body = append(body, m.inputLine("auth.db", m.user.database.View(), m.user.focus == 3, inner))
		for _, line := range wrapText("auth database: stores the MongoDB user; it does not grant access", inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
		for _, line := range wrapText("Space cycles: "+strings.Join(m.mongoUserDatabases(), ", "), inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
		body = append(body, m.inputLine("hosts", m.user.hosts.View(), m.user.focus == 4, inner))
		for _, line := range wrapText("comma-separated host:port list for the credential URI", inner) {
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
