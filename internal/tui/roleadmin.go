package tui

import (
	"context"
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

type roleState struct {
	menu         roleMenuState
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
	case domain.KindSecurity, domain.KindUsers:
		return m.startUserAdd()
	case domain.KindUser:
		m.setMessage(levelInfo, "select the users group to create a user")
	default:
		m.setMessage(levelInfo, "select a role, user, or the matching group")
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
	case domain.KindUser:
		return m.startUserRemove()
	case domain.KindUsers:
		m.setMessage(levelInfo, "select a user first")
	case domain.KindRoles:
		m.setMessage(levelInfo, "select a role first")
	default:
		m.setMessage(levelInfo, "select a role or user")
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
	role := roleRefFromNode(node)
	return m.openMatrix(matrixPrivileges, &role)
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
		role := m.role.menu.role
		switch choice {
		case "Grant privileges", "Revoke privileges":
			return m.openMatrix(matrixPrivileges, &role)
		case "Assign to user/role", "Remove assignment":
			return m.openMatrix(matrixMembership, &role)
		case "Drop role":
			m.role.confirm = role
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
	if m.screen == ScreenMatrix {
		if msg.err != nil {
			m.matrix.busy = false
			text := matrixErrorText(msg.err)
			if m.matrix.details.open {
				m.matrix.details.err = text
				return m, nil
			}
			m.setMessage(levelError, text)
			return m, m.reloadMatrix()
		}
		m.matrix.details = matrixDetails{}
		m.matrixDirty = true
		m.setMessage(levelSuccess, "change applied · "+oneLine(msg.result.Statement))
		return m, m.reloadMatrix()
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
	case roleAssign, roleUnassign:
		commands = append(commands,
			m.refreshPrincipalGroup(domain.KindUsers),
			m.refreshPrincipalGroup(domain.KindRoles),
			m.refreshMatrixScope())
	default:
		commands = append(commands, m.refreshMatrixScope(), m.refreshCurrentPrivileges())
	}
	return m, tea.Batch(commands...)
}

func (m *Model) refreshPrincipalGroup(kind domain.ObjectKind) tea.Cmd {
	current := m.currentDatabase()
	fallbackKey, fallbackNode := "", domain.Node{}
	for key, node := range m.tree.nodes {
		if node.Node.Kind != kind {
			continue
		}
		if current != "" && node.Node.Ref.Database == current {
			m.tree.setLoading(key, true)
			return m.loadChildrenCmd(node.Node)
		}
		if fallbackKey == "" {
			fallbackKey, fallbackNode = key, node.Node
		}
	}
	if fallbackKey == "" {
		return nil
	}
	m.tree.setLoading(fallbackKey, true)
	return m.loadChildrenCmd(fallbackNode)
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

// refreshMatrixScope reloads the loaded object group whose access markers can
// change after a grant or membership edit.
func (m *Model) refreshMatrixScope() tea.Cmd {
	engine := m.engine()
	database := m.currentDatabase()
	schema := m.matrixSchema(engine)
	for key, node := range m.tree.nodes {
		switch {
		case engine == domain.EnginePostgreSQL && node.Node.Kind == domain.KindSchema &&
			node.Node.Ref.Database == database && node.Node.Ref.Schema == schema:
			m.tree.setLoading(key, true)
			return m.loadChildrenCmd(node.Node)
		case engine != domain.EnginePostgreSQL && node.Node.Kind == domain.KindDatabase &&
			node.Node.Ref.Database == database:
			m.tree.setLoading(key, true)
			return m.loadChildrenCmd(node.Node)
		}
	}
	return nil
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
	body = append(body, m.inputLine("name", m.inlineInput(m.role.createName, m.role.createCursor), true, inner))
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

func (m Model) inlineInput(value string, cursor int) string {
	value = filterText(value)
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
