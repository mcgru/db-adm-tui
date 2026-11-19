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

type roleAction int

const (
	roleCreate roleAction = iota
	roleGrant
	roleRevoke
	roleAssign
	roleUnassign
	roleDrop
)

type roleMenuState struct {
	role     domain.RoleRef
	add      bool
	selected int
}

type roleFormState struct {
	action      roleAction
	role        domain.RoleRef
	focus       int
	fieldCursor int
	targetKind  domain.ObjectKind
	memberKind  domain.ObjectKind
	database    string
	schema      string
	table       string
	member      string
	choices     []string
	chosen      map[string]bool
	cursor      int
	offset      int
	err         string
}

type roleState struct {
	menu         roleMenuState
	form         roleFormState
	confirm      domain.RoleRef
	createName   string
	createCursor int
}

type roleDoneMsg struct {
	gen    uint64
	action roleAction
	result domain.MutationResult
	err    error
}

func newRoleState() roleState {
	return roleState{}
}

func (m *Model) roleWritable() bool {
	if m.session == nil {
		m.setMessage(levelInfo, "no active connection")
		return false
	}
	if m.session.ReadOnly() {
		m.setMessage(levelError, "Blocked: role changes require read-write mode. Ctrl+X Ctrl+W to switch.")
		return false
	}
	return true
}

func (m Model) startRoleAdd() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok {
		return m, nil
	}
	switch node.Kind {
	case domain.KindRoles:
		if !m.roleWritable() {
			return m, nil
		}
		m.createRoleModal()
	case domain.KindRole:
		if !m.roleWritable() {
			return m, nil
		}
		m.openRoleMenu(node, true)
	case domain.KindUsers, domain.KindUser:
		m.setMessage(levelInfo, "user changes are story 9c")
	default:
		m.setMessage(levelInfo, "select a role or the roles group")
	}
	return m, nil
}

func (m Model) startRoleRemove() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok {
		return m, nil
	}
	switch node.Kind {
	case domain.KindRole:
		if !m.roleWritable() {
			return m, nil
		}
		m.openRoleMenu(node, false)
	case domain.KindUsers, domain.KindUser:
		m.setMessage(levelInfo, "user changes are story 9c")
	case domain.KindRoles:
		m.setMessage(levelInfo, "select a role first")
	default:
		m.setMessage(levelInfo, "select a role")
	}
	return m, nil
}

func (m Model) startRoleEdit() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok || node.Kind != domain.KindRole {
		m.setMessage(levelInfo, "select a role to edit")
		return m, nil
	}
	if !m.roleWritable() {
		return m, nil
	}
	m.openRoleForm(roleGrant, roleRefFromNode(node))
	return m, nil
}

func roleRefFromNode(node domain.Node) domain.RoleRef {
	return domain.RoleRef{
		Engine:   node.Ref.Engine,
		Database: node.Ref.Database,
		Name:     node.Name,
		Kind:     node.Kind,
	}
}

func (m *Model) createRoleModal() {
	m.role.createName = ""
	m.role.createCursor = 0
	m.modal = modalRoleCreate
}

func (m *Model) openRoleMenu(node domain.Node, add bool) {
	m.role.menu = roleMenuState{role: roleRefFromNode(node), add: add, selected: 0}
	m.modal = modalRoleMenu
}

func (m Model) roleMenuItems() []string {
	if m.role.menu.add {
		return []string{"Grant privileges", "Assign to user/role"}
	}
	return []string{"Revoke privileges", "Remove assignment", "Drop role"}
}

func (m *Model) openRoleForm(action roleAction, role domain.RoleRef) {
	database := role.Database
	if database == "" {
		database = m.currentDatabase()
	}
	form := roleFormState{
		action:     action,
		role:       role,
		targetKind: domain.KindDatabase,
		memberKind: domain.KindUser,
		database:   database,
		chosen:     map[string]bool{},
	}
	if action == roleAssign || action == roleUnassign {
		form.focus = 0
	} else {
		kinds := domain.RoleTargetKinds(m.profileForStatus().Engine)
		if len(kinds) > 0 {
			form.targetKind = kinds[0]
		}
		form.choices = domain.RolePrivilegeChoices(m.profileForStatus().Engine, form.targetKind)
		form.focus = 0
	}
	m.role.form = form
	m.modal = modalRoleForm
}

func (m *Model) cycleRoleTargetKind(delta int) {
	form := &m.role.form
	kinds := domain.RoleTargetKinds(m.profileForStatus().Engine)
	if len(kinds) == 0 {
		return
	}
	index := 0
	for i, kind := range kinds {
		if kind == form.targetKind {
			index = i
			break
		}
	}
	index = (index + delta + len(kinds)) % len(kinds)
	form.targetKind = kinds[index]
	form.choices = domain.RolePrivilegeChoices(m.profileForStatus().Engine, form.targetKind)
	form.chosen = map[string]bool{}
	form.cursor = 0
	form.offset = 0
}

func (m *Model) toggleRolePrivilege() {
	form := &m.role.form
	if form.cursor < 0 || form.cursor >= len(form.choices) {
		return
	}
	value := form.choices[form.cursor]
	form.chosen[value] = !form.chosen[value]
}

func (m Model) roleFormFields() int {
	if m.role.form.action == roleAssign || m.role.form.action == roleUnassign {
		return 2
	}
	return 5
}

func (m *Model) moveRoleFocus(delta int) {
	form := &m.role.form
	form.focus = (form.focus + delta + m.roleFormFields()) % m.roleFormFields()
	form.fieldCursor = len([]rune(m.roleFieldValue()))
}

func (m Model) roleFieldValue() string {
	form := m.role.form
	if form.action == roleAssign || form.action == roleUnassign {
		if form.focus == 1 {
			return form.member
		}
		return ""
	}
	switch form.focus {
	case 1:
		return form.database
	case 2:
		return form.schema
	case 3:
		return form.table
	}
	return ""
}

func (m *Model) setRoleFieldValue(value string) {
	form := &m.role.form
	switch form.focus {
	case 1:
		form.database = value
	case 2:
		form.schema = value
	case 3:
		form.table = value
	}
	if form.action == roleAssign || form.action == roleUnassign {
		if form.focus == 1 {
			form.member = value
		}
	}
}

func (m Model) handleRoleMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.modal = modalNone
	case keymap.Up:
		m.role.menu.selected--
		if m.role.menu.selected < 0 {
			m.role.menu.selected = len(m.roleMenuItems()) - 1
		}
	case keymap.Down:
		m.role.menu.selected++
		if m.role.menu.selected >= len(m.roleMenuItems()) {
			m.role.menu.selected = 0
		}
	case keymap.Confirm:
		m.modal = modalNone
		items := m.roleMenuItems()
		if len(items) == 0 {
			return m, nil
		}
		choice := items[m.role.menu.selected]
		switch choice {
		case "Grant privileges":
			m.openRoleForm(roleGrant, m.role.menu.role)
		case "Assign to user/role":
			m.openRoleForm(roleAssign, m.role.menu.role)
		case "Revoke privileges":
			m.openRoleForm(roleRevoke, m.role.menu.role)
		case "Remove assignment":
			m.openRoleForm(roleUnassign, m.role.menu.role)
		case "Drop role":
			m.role.confirm = m.role.menu.role
			m.modal = modalRoleConfirm
		}
	}
	return m, nil
}

func (m Model) handleRoleCreateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			return m, nil
		case keymap.Confirm:
			name := strings.TrimSpace(m.role.createName)
			if name == "" {
				m.setMessage(levelError, "enter a role name")
				return m, nil
			}
			return m.startRoleMutation(roleCreate, func(session *app.Session) (domain.MutationResult, error) {
				return session.CreateRole(context.Background(), domain.RoleRef{
					Engine: m.profileForStatus().Engine, Database: m.currentDatabase(), Name: name, Kind: domain.KindRole,
				})
			})
		}
	}
	m.role.createName, m.role.createCursor = editFilterText(m.role.createName, m.role.createCursor, msg)
	return m, nil
}

func (m Model) handleRoleFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if action, ok := keymap.Lookup(key); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
			return m, nil
		case keymap.Tab:
			m.moveRoleFocus(1)
			return m, nil
		case keymap.ShiftTab:
			m.moveRoleFocus(-1)
			return m, nil
		case keymap.Confirm:
			return m.submitRoleForm()
		case keymap.Space:
			if m.role.form.action != roleAssign && m.role.form.action != roleUnassign && m.role.form.focus == 0 {
				m.cycleRoleTargetKind(1)
				return m, nil
			}
			if m.role.form.action != roleAssign && m.role.form.action != roleUnassign && m.role.form.focus == 4 {
				m.toggleRolePrivilege()
				return m, nil
			}
			if (m.role.form.action == roleAssign || m.role.form.action == roleUnassign) && m.role.form.focus == 0 {
				if m.role.form.memberKind == domain.KindUser {
					m.role.form.memberKind = domain.KindRole
				} else {
					m.role.form.memberKind = domain.KindUser
				}
				return m, nil
			}
		case keymap.Up:
			if m.role.form.action != roleAssign && m.role.form.action != roleUnassign && m.role.form.focus == 4 {
				m.role.form.cursor--
				if m.role.form.cursor < 0 {
					m.role.form.cursor = len(m.role.form.choices) - 1
				}
				m.role.form.offset = completionWindowStart(m.role.form.offset, m.role.form.cursor, 8)
				return m, nil
			}
		case keymap.Down:
			if m.role.form.action != roleAssign && m.role.form.action != roleUnassign && m.role.form.focus == 4 {
				m.role.form.cursor++
				if m.role.form.cursor >= len(m.role.form.choices) {
					m.role.form.cursor = 0
				}
				m.role.form.offset = completionWindowStart(m.role.form.offset, m.role.form.cursor, 8)
				return m, nil
			}
		}
	}
	if m.role.form.action != roleAssign && m.role.form.action != roleUnassign &&
		(m.role.form.focus == 0 || m.role.form.focus == 4) {
		return m, nil
	}
	value, cursor := editFilterText(m.roleFieldValue(), m.role.form.fieldCursor, msg)
	m.role.form.fieldCursor = cursor
	m.setRoleFieldValue(value)
	return m, nil
}

func (m Model) submitRoleForm() (tea.Model, tea.Cmd) {
	form := m.role.form
	if form.action == roleAssign || form.action == roleUnassign {
		member := strings.TrimSpace(form.member)
		if member == "" {
			m.role.form.err = "enter a user or role name"
			return m, nil
		}
		membership := domain.RoleMembership{
			Role:   form.role,
			Member: domain.RoleRef{Engine: form.role.Engine, Database: form.role.Database, Name: member, Kind: form.memberKind},
		}
		return m.startRoleMutation(form.action, func(session *app.Session) (domain.MutationResult, error) {
			if form.action == roleAssign {
				return session.GrantRole(context.Background(), membership)
			}
			return session.RevokeRole(context.Background(), membership)
		})
	}
	target, err := m.roleTarget(form)
	if err != nil {
		m.role.form.err = err.Error()
		return m, nil
	}
	privileges := make([]string, 0, len(form.choices))
	for _, choice := range form.choices {
		if form.chosen[choice] {
			privileges = append(privileges, choice)
		}
	}
	if len(privileges) == 0 {
		m.role.form.err = "choose at least one privilege"
		return m, nil
	}
	grant := domain.PrivilegeGrant{Role: form.role, Target: target, Privileges: privileges}
	return m.startRoleMutation(form.action, func(session *app.Session) (domain.MutationResult, error) {
		if form.action == roleGrant {
			return session.GrantPrivileges(context.Background(), grant)
		}
		return session.RevokePrivileges(context.Background(), grant)
	})
}

func (m Model) roleTarget(form roleFormState) (domain.ObjectRef, error) {
	database := strings.TrimSpace(form.database)
	schema := strings.TrimSpace(form.schema)
	table := strings.TrimSpace(form.table)
	switch form.targetKind {
	case domain.KindDatabase:
		if database == "" {
			return domain.ObjectRef{}, fmt.Errorf("enter a database")
		}
		return domain.ObjectRef{Engine: form.role.Engine, Kind: domain.KindDatabase, Database: database, Name: database}, nil
	case domain.KindSchema:
		if schema == "" {
			return domain.ObjectRef{}, fmt.Errorf("enter a schema")
		}
		return domain.ObjectRef{Engine: form.role.Engine, Kind: domain.KindSchema, Database: database, Schema: schema, Name: schema}, nil
	case domain.KindTable:
		if database == "" || table == "" {
			return domain.ObjectRef{}, fmt.Errorf("enter a database and table")
		}
		return domain.ObjectRef{Engine: form.role.Engine, Kind: domain.KindTable, Database: database, Schema: schema, Name: table}, nil
	case domain.KindCollection:
		if database == "" || table == "" {
			return domain.ObjectRef{}, fmt.Errorf("enter a database and collection")
		}
		return domain.ObjectRef{Engine: form.role.Engine, Kind: domain.KindCollection, Database: database, Name: table}, nil
	default:
		return domain.ObjectRef{}, fmt.Errorf("choose a target")
	}
}

func (m Model) startRoleMutation(action roleAction, run func(*app.Session) (domain.MutationResult, error)) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	m.modal = modalNone
	m.roleGen++
	gen := m.roleGen
	session := m.session
	m.setMessage(levelInfo, "applying role change…")
	return m, func() tea.Msg {
		result, err := run(session)
		return roleDoneMsg{gen: gen, action: action, result: result, err: err}
	}
}

func (m Model) handleRoleDone(msg roleDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.roleGen {
		return m, nil
	}
	if msg.err != nil {
		m.setMessage(levelError, oneLine(msg.err.Error()))
		return m, nil
	}
	m.setMessage(levelSuccess, "role change applied · "+oneLine(msg.result.Statement))
	commands := []tea.Cmd{}
	switch msg.action {
	case roleCreate, roleDrop:
		commands = append(commands, m.refreshPrincipalGroup(domain.KindRoles))
	case roleGrant, roleRevoke:
		commands = append(commands, m.refreshCurrentPrivileges())
	case roleAssign, roleUnassign:
		commands = append(commands, m.refreshPrincipalGroup(domain.KindUsers), m.refreshPrincipalGroup(domain.KindRoles))
	default:
		commands = append(commands, m.refreshCurrentPrivileges())
	}
	return m, tea.Batch(commands...)
}

func (m *Model) refreshPrincipalGroup(kind domain.ObjectKind) tea.Cmd {
	for key, node := range m.tree.nodes {
		if node.Node.Kind == kind {
			m.tree.setLoading(key, true)
			return m.loadChildrenCmd(node.Node)
		}
	}
	return nil
}

func (m *Model) refreshCurrentPrivileges() tea.Cmd {
	if !m.details.has {
		return nil
	}
	m.details.privilegesLoaded = false
	m.details.privilegesLoading = false
	m.details.privilegesErr = ""
	return m.ensurePrivileges()
}

func (m Model) handleRoleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.Cancel:
			m.modal = modalNone
		case keymap.Confirm:
			role := m.role.confirm
			return m.startRoleMutation(roleDrop, func(session *app.Session) (domain.MutationResult, error) {
				return session.DropRole(context.Background(), role)
			})
		}
	}
	return m, nil
}

func (m Model) viewRoleMenu() string {
	width := modalWidth(m.width, 56)
	inner := width - 4
	var body []string
	for i, item := range m.roleMenuItems() {
		line := "  " + item
		if i == m.role.menu.selected {
			line = "> " + item
			body = append(body, m.styles.Select.Render(Pad(line, inner)))
		} else {
			body = append(body, m.styles.Text.Render(line))
		}
	}
	body = append(body, "", m.actionLine("Enter", "choose", "Esc", "cancel"))
	return modalBox("role "+oneLine(m.role.menu.role.Name), body, width, m.styles)
}

func (m Model) viewRoleCreate() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	var body []string
	body = append(body, m.inputLine("name", m.roleInputValue(m.role.createName, true), true, inner))
	body = append(body, "", m.actionLine("Enter", "create", "Esc", "cancel"))
	return modalBox("new role", body, width, m.styles)
}

func (m Model) viewRoleConfirm() string {
	width := modalWidth(m.width, 60)
	inner := width - 4
	var body []string
	for _, line := range wrapText("Drop role "+oneLine(m.role.confirm.Name)+"? Native dependencies or ownership may block it.", inner) {
		body = append(body, m.styles.Text.Render(line))
	}
	body = append(body, "", m.actionLine("Enter", "remove", "Esc", "cancel"))
	return modalBox("drop role", body, width, m.styles)
}

func (m Model) viewRoleForm() string {
	width := modalWidth(m.width, 70)
	inner := width - 4
	form := m.role.form
	var body []string
	body = append(body, m.styles.Muted.Render("role ")+m.styles.Text.Render(oneLine(form.role.Name)))
	if form.action == roleAssign || form.action == roleUnassign {
		kind := "user"
		if form.memberKind == domain.KindRole {
			kind = "role"
		}
		body = append(body, m.inputLine("member", m.roleInputValue(kind, form.focus == 0), form.focus == 0, inner))
		body = append(body, m.inputLine("name", m.roleInputValue(form.member, form.focus == 1), form.focus == 1, inner))
	} else {
		kind := roleTargetKindLabel(form.targetKind)
		body = append(body, m.inputLine("target", m.roleInputValue(kind+" (Space)", form.focus == 0), form.focus == 0, inner))
		body = append(body, m.inputLine("database", m.roleInputValue(form.database, form.focus == 1), form.focus == 1, inner))
		body = append(body, m.inputLine("schema", m.roleInputValue(form.schema, form.focus == 2), form.focus == 2, inner))
		tableLabel := "table"
		if form.targetKind == domain.KindCollection {
			tableLabel = "collection"
		}
		body = append(body, m.inputLine(tableLabel, m.roleInputValue(form.table, form.focus == 3), form.focus == 3, inner))
		body = append(body, "", m.styles.Header.Render("privileges · Space toggle · Up/Down move"))
		body = append(body, m.viewRoleChoices(inner)...)
	}
	if form.err != "" {
		body = append(body, m.styles.Error.Render(Truncate(form.err, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "apply", "Tab", "next field"))
	body = append(body, m.styles.Muted.Render("Esc cancel"))
	return modalBox(roleActionTitle(form.action), body, width, m.styles)
}

func (m Model) viewRoleChoices(width int) []string {
	form := m.role.form
	if len(form.choices) == 0 {
		return []string{m.styles.Muted.Render("no privileges for this target")}
	}
	start := form.offset
	end := start + 8
	if end > len(form.choices) {
		end = len(form.choices)
	}
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		mark := "[ ]"
		if form.chosen[form.choices[i]] {
			mark = "[x]"
		}
		line := "  " + mark + " " + form.choices[i]
		if i == form.cursor {
			line = "> " + mark + " " + form.choices[i]
			lines = append(lines, m.styles.Select.Render(Pad(line, width)))
		} else {
			lines = append(lines, m.styles.Text.Render(line))
		}
	}
	return lines
}

func (m Model) roleInputValue(value string, focused bool) string {
	value = filterText(value)
	if !focused {
		return m.styles.Text.Render(value)
	}
	cursor := m.role.form.fieldCursor
	if m.modal == modalRoleCreate {
		cursor = m.role.createCursor
	}
	runes := []rune(value)
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

func roleTargetKindLabel(kind domain.ObjectKind) string {
	switch kind {
	case domain.KindDatabase:
		return "database"
	case domain.KindSchema:
		return "schema"
	case domain.KindTable:
		return "table"
	case domain.KindCollection:
		return "collection"
	default:
		return string(kind)
	}
}

func roleActionTitle(action roleAction) string {
	switch action {
	case roleGrant:
		return "grant privileges"
	case roleRevoke:
		return "revoke privileges"
	case roleAssign:
		return "assign role"
	case roleUnassign:
		return "remove assignment"
	default:
		return "role"
	}
}
