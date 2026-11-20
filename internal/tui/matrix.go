package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

type matrixKind int

const (
	matrixMembership matrixKind = iota
	matrixPrivileges
)

type matrixCursor struct {
	row, col             int
	rowOffset, colOffset int
	filter               string
	filterPrompt         bool
	filterCursor         int
}

type matrixDetails struct {
	open       bool
	membership bool

	role    domain.RoleRef
	member  domain.RoleRef
	object  domain.ObjectRef
	granted bool

	choices []string
	chosen  map[string]bool
	current []string
	cursor  int
	offset  int
	err     string
}

type matrixState struct {
	active matrixKind

	membership        domain.RoleMatrix
	membershipLoaded  bool
	membershipLoading bool
	membershipErr     string

	privileges        domain.PrivilegeMatrix
	privilegesLoaded  bool
	privilegesLoading bool
	privilegesErr     string

	objects  []domain.ObjectRef
	scope    string
	ensureDB string

	cursors [2]matrixCursor
	details matrixDetails
	busy    bool

	pendingRole   domain.RoleRef
	pendingMatrix matrixKind
	hasPending    bool
}

type matrixMembershipMsg struct {
	gen  uint64
	data domain.RoleMatrix
	err  error
}

type matrixPrivilegesMsg struct {
	gen  uint64
	data domain.PrivilegeMatrix
	err  error
}

func newMatrixState() matrixState {
	return matrixState{}
}

func (m Model) openMatrix(active matrixKind, focus *domain.RoleRef) (tea.Model, tea.Cmd) {
	return m.openMatrixWithDB(active, focus, "")
}

func (m Model) openMatrixWithDB(active matrixKind, focus *domain.RoleRef, ensureDB string) (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	if m.session.ReadOnly() {
		m.setMessage(levelError, "Blocked: the access matrix requires read-write mode. Ctrl+X Ctrl+W to switch.")
		return m, nil
	}
	m.screen = ScreenMatrix
	m.modal = modalNone
	m.message = ""
	m.messageLevel = levelNone
	m.completion = completionState{}
	m.pendingChord = false
	m.matrix = newMatrixState()
	m.matrix.active = active
	m.matrix.ensureDB = ensureDB
	if focus == nil && ensureDB != "" && m.engine() == domain.EngineMongoDB && active == matrixMembership {
		role := domain.RoleRef{
			Engine: domain.EngineMongoDB, Database: ensureDB,
			Name: "read@" + ensureDB, Kind: domain.KindRole, Builtin: true,
		}
		focus = &role
	}
	if focus != nil {
		m.matrix.pendingRole = *focus
		m.matrix.pendingMatrix = active
		m.matrix.hasPending = true
	}
	m.matrix.objects, m.matrix.scope = m.matrixObjects()
	return m, m.loadMatrix()
}

func (m *Model) ensureMatrixDatabaseRoles() {
	database := strings.TrimSpace(m.matrix.ensureDB)
	if database == "" || m.engine() != domain.EngineMongoDB {
		return
	}
	for _, name := range []string{"read", "readWrite"} {
		ref := domain.RoleRef{
			Engine: domain.EngineMongoDB, Database: database,
			Name: name + "@" + database, Kind: domain.KindRole, Builtin: true,
		}
		if roleIndex(m.matrix.membership.Roles, ref.Key()) < 0 {
			m.matrix.membership.Roles = append(m.matrix.membership.Roles, ref)
		}
	}
	sort.SliceStable(m.matrix.membership.Roles, func(i, j int) bool {
		if m.matrix.membership.Roles[i].Database != m.matrix.membership.Roles[j].Database {
			return m.matrix.membership.Roles[i].Database < m.matrix.membership.Roles[j].Database
		}
		return m.matrix.membership.Roles[i].Name < m.matrix.membership.Roles[j].Name
	})
}

func (m Model) matrixEnsureDatabase() string {
	if m.engine() != domain.EngineMongoDB {
		return ""
	}
	node, ok := m.tree.selectedNode()
	if !ok {
		return ""
	}
	switch node.Kind {
	case domain.KindDatabase:
		if node.Ref.Database != "" {
			return node.Ref.Database
		}
		return node.Name
	case domain.KindTable, domain.KindCollection:
		return node.Ref.Database
	default:
		return ""
	}
}

func (m *Model) loadMatrix() tea.Cmd {
	if m.session == nil {
		return nil
	}
	m.matrixGen++
	gen := m.matrixGen
	// Keep already-loaded cells visible while a mutation refreshes the matrix.
	// The initial open starts from an empty state, so it still shows a loading
	// notice until the first page arrives.
	m.matrix.membershipLoading = true
	m.matrix.membershipErr = ""
	m.matrix.privilegesLoading = true
	m.matrix.privilegesErr = ""
	session := m.session
	objects := append([]domain.ObjectRef(nil), m.matrix.objects...)
	membershipCmd := func() tea.Msg {
		data, err := session.RoleMatrix(context.Background())
		return matrixMembershipMsg{gen: gen, data: data, err: err}
	}
	privilegesCmd := func() tea.Msg {
		data, err := session.PrivilegeMatrix(context.Background(), objects)
		return matrixPrivilegesMsg{gen: gen, data: data, err: err}
	}
	return tea.Batch(membershipCmd, privilegesCmd)
}

func (m Model) reloadMatrix() tea.Cmd {
	return m.loadMatrix()
}

func (m Model) handleMatrixMembership(msg matrixMembershipMsg) (tea.Model, tea.Cmd) {
	if m.screen != ScreenMatrix || msg.gen != m.matrixGen {
		return m, nil
	}
	m.matrix.membershipLoading = false
	if msg.err != nil {
		m.matrix.membershipErr = matrixErrorText(msg.err)
		if !m.matrix.privilegesLoading {
			m.matrix.busy = false
		}
		return m, nil
	}
	m.matrix.membership = msg.data
	m.matrix.membershipLoaded = true
	if !m.matrix.privilegesLoading {
		m.matrix.busy = false
	}
	m.ensureMatrixDatabaseRoles()
	m.applyMatrixPendingFocus()
	m.clampMatrix()
	return m, nil
}

func (m Model) handleMatrixPrivileges(msg matrixPrivilegesMsg) (tea.Model, tea.Cmd) {
	if m.screen != ScreenMatrix || msg.gen != m.matrixGen {
		return m, nil
	}
	m.matrix.privilegesLoading = false
	if msg.err != nil {
		m.matrix.privilegesErr = matrixErrorText(msg.err)
		if !m.matrix.membershipLoading {
			m.matrix.busy = false
		}
		return m, nil
	}
	m.matrix.privileges = msg.data
	m.matrix.privilegesLoaded = true
	if !m.matrix.membershipLoading {
		m.matrix.busy = false
	}
	m.applyMatrixPendingFocus()
	m.clampMatrix()
	return m, nil
}

func (m *Model) applyMatrixPendingFocus() {
	if !m.matrix.hasPending {
		return
	}
	role := m.matrix.pendingRole
	switch m.matrix.pendingMatrix {
	case matrixMembership:
		if !m.matrix.membershipLoaded {
			return
		}
		m.matrix.active = matrixMembership
		if index := roleIndex(m.matrix.membership.Roles, role.Key()); index >= 0 {
			m.matrix.cursors[matrixMembership].row = index
		}
	case matrixPrivileges:
		if !m.matrix.privilegesLoaded {
			return
		}
		m.matrix.active = matrixPrivileges
		if index := roleIndex(m.matrix.privileges.Principals, role.Key()); index >= 0 {
			m.matrix.cursors[matrixPrivileges].row = index
		}
	}
	m.matrix.hasPending = false
}

func roleIndex(refs []domain.RoleRef, key string) int {
	for i, ref := range refs {
		if ref.Key() == key {
			return i
		}
	}
	return -1
}

// matrixRoleRows returns the left-hand rows: roles in the membership matrix,
// principals/roles in the privilege matrix.
func (m Model) matrixRoleRows() []domain.RoleRef {
	filter := m.matrix.cursors[m.matrix.active].filter
	switch m.matrix.active {
	case matrixMembership:
		return filterMatrixRoles(m.matrix.membership.Roles, filter)
	default:
		return filterMatrixRoles(m.matrix.privileges.Principals, filter)
	}
}

func (m Model) matrixObjectCols() []domain.ObjectRef {
	if m.matrix.active != matrixPrivileges {
		return nil
	}
	return filterMatrixObjects(m.matrix.privileges.Objects, m.matrix.cursors[matrixPrivileges].filter)
}

func (m Model) matrixRoleCols() []domain.RoleRef {
	if m.matrix.active != matrixMembership {
		return nil
	}
	return filterMatrixRoles(m.matrix.membership.Principals, m.matrix.cursors[matrixMembership].filter)
}

func (m Model) matrixRowCount() int {
	return len(m.matrixRoleRows())
}

func (m Model) matrixColCount() int {
	if m.matrix.active == matrixMembership {
		return len(m.matrixRoleCols())
	}
	return len(m.matrixObjectCols())
}

func (m Model) matrixRowLabel(index int) string {
	rows := m.matrixRoleRows()
	if index < 0 || index >= len(rows) {
		return ""
	}
	return matrixRoleLabel(rows[index])
}

func (m Model) matrixColLabel(index int) string {
	if m.matrix.active == matrixMembership {
		cols := m.matrixRoleCols()
		if index < 0 || index >= len(cols) {
			return ""
		}
		return matrixPrincipalLabel(cols[index])
	}
	cols := m.matrixObjectCols()
	if index < 0 || index >= len(cols) {
		return ""
	}
	return matrixObjectLabel(cols[index])
}

// matrixRoleLabel keeps the database suffix on the left, where there is room
// for long names such as readWriteAnyDatabase@admin.
func matrixRoleLabel(ref domain.RoleRef) string {
	return rolePrefix(ref) + oneLine(ref.Name)
}

// matrixPrincipalLabel is the compact column label. MongoDB users in admin are
// shown without @admin; users in another database keep @database. MySQL users
// hide the default % host.
func matrixPrincipalLabel(ref domain.RoleRef) string {
	if ref.Kind == domain.KindUser && ref.Engine == domain.EngineMongoDB {
		name := oneLine(ref.Name)
		database := ref.Database
		if index := strings.LastIndex(name, "@"); index >= 0 {
			if database == "" {
				database = name[index+1:]
			}
			name = name[:index]
		}
		if database == "" || database == "admin" {
			return "u:" + name
		}
		return "u:" + name + "@" + database
	}
	if ref.Kind == domain.KindUser && ref.Engine == domain.EngineMySQL {
		name := oneLine(ref.Name)
		name = strings.TrimSuffix(name, "@%")
		return "u:" + name
	}
	return rolePrefix(ref) + oneLine(ref.Name)
}

func rolePrefix(ref domain.RoleRef) string {
	switch {
	case ref.Builtin:
		return "R:"
	case ref.Kind == domain.KindUser:
		return "u:"
	case ref.Kind == domain.KindRole:
		return "r:"
	default:
		return "·:"
	}
}

func matrixObjectLabel(ref domain.ObjectRef) string {
	name := oneLine(ref.Name)
	switch ref.Kind {
	case domain.KindDatabase:
		return "[db] " + name
	case domain.KindSchema:
		return "[schema] " + name
	case domain.KindView:
		return name + " [view]"
	case domain.KindCollection:
		return name
	default:
		return name
	}
}

func (m Model) matrixCellText(row, col int) string {
	if m.matrix.active == matrixMembership {
		rows := m.matrixRoleRows()
		cols := m.matrixRoleCols()
		if row < 0 || row >= len(rows) || col < 0 || col >= len(cols) {
			return ""
		}
		if m.matrix.membership.Members[cols[col].Key()][rows[row].Key()] {
			return "[x]"
		}
		return "[ ]"
	}
	rows := m.matrixRoleRows()
	cols := m.matrixObjectCols()
	if row < 0 || row >= len(rows) || col < 0 || col >= len(cols) {
		return ""
	}
	values := m.matrix.privileges.Cells[rows[row].Key()][cols[col].Key()]
	if len(values) == 0 {
		return "—"
	}
	if len(values) == 1 {
		return values[0]
	}
	return values[0] + fmt.Sprintf(" +%d", len(values)-1)
}

func (m Model) matrixFilterVisible() bool {
	cursor := m.matrix.cursors[m.matrix.active]
	return cursor.filter != "" || cursor.filterPrompt
}

func (m Model) matrixCellWidth() int {
	if m.matrix.active == matrixMembership {
		width := 12
		for _, ref := range m.matrixRoleCols() {
			if w := lipgloss.Width(matrixPrincipalLabel(ref)); w > width {
				width = w
			}
		}
		if width > 24 {
			width = 24
		}
		return width
	}
	return 16
}

// matrixTruncatedColumnLabel returns the full current column label when it
// does not fit into a cell. The matrix tabs line shows it after "Tab switch"
// (or on the next line when the combined line would be too wide).
func (m Model) matrixTruncatedColumnLabel() (string, bool) {
	cursor := m.matrix.cursors[m.matrix.active]
	if cursor.col < 0 || cursor.col >= m.matrixColCount() {
		return "", false
	}
	label := oneLine(m.matrixColLabel(cursor.col))
	if label == "" || lipgloss.Width(label) <= m.matrixCellWidth() {
		return "", false
	}
	return label, true
}

func (m Model) matrixPrincipalWidth() int {
	width := m.width / 4
	if width < 18 {
		width = 18
	}
	if width > 32 {
		width = 32
	}
	return width
}

func (m Model) matrixViewport() (rows, cols int) {
	innerHeight := m.height - 4
	used := len(m.matrixHeaderLines(m.width-4)) + 1 // tabs/hint + column header
	if m.matrixFilterVisible() {
		used++
	}
	rows = innerHeight - used
	if rows < 1 {
		rows = 1
	}
	innerWidth := m.width - 4
	cols = (innerWidth - m.matrixPrincipalWidth() - 1) / (m.matrixCellWidth() + 1)
	if cols < 1 {
		cols = 1
	}
	return rows, cols
}

func (m *Model) clampMatrix() {
	cursor := &m.matrix.cursors[m.matrix.active]
	rows := m.matrixRowCount()
	cols := m.matrixColCount()
	if rows == 0 {
		cursor.row = 0
	} else if cursor.row >= rows {
		cursor.row = rows - 1
	}
	if cols == 0 {
		cursor.col = 0
	} else if cursor.col >= cols {
		cursor.col = cols - 1
	}
	if cursor.row < 0 {
		cursor.row = 0
	}
	if cursor.col < 0 {
		cursor.col = 0
	}
	m.ensureMatrixVisible()
}

func (m *Model) ensureMatrixVisible() {
	rows, cols := m.matrixViewport()
	cursor := &m.matrix.cursors[m.matrix.active]
	rowCount := m.matrixRowCount()
	colCount := m.matrixColCount()
	if cursor.row < cursor.rowOffset {
		cursor.rowOffset = cursor.row
	}
	if cursor.row >= cursor.rowOffset+rows {
		cursor.rowOffset = cursor.row - rows + 1
	}
	maxRowOffset := rowCount - rows
	if maxRowOffset < 0 {
		maxRowOffset = 0
	}
	if cursor.rowOffset > maxRowOffset {
		cursor.rowOffset = maxRowOffset
	}
	if cursor.rowOffset < 0 {
		cursor.rowOffset = 0
	}
	if cursor.col < cursor.colOffset {
		cursor.colOffset = cursor.col
	}
	if cursor.col >= cursor.colOffset+cols {
		cursor.colOffset = cursor.col - cols + 1
	}
	maxColOffset := colCount - cols
	if maxColOffset < 0 {
		maxColOffset = 0
	}
	if cursor.colOffset > maxColOffset {
		cursor.colOffset = maxColOffset
	}
	if cursor.colOffset < 0 {
		cursor.colOffset = 0
	}
}

func (m *Model) switchMatrix(active matrixKind) {
	if active < matrixMembership || active > matrixPrivileges {
		return
	}
	from := m.matrix.active
	if from == active {
		m.clampMatrix()
		return
	}
	candidates := m.matrixSwitchCandidates(from)
	m.matrix.active = active
	switch active {
	case matrixPrivileges:
		rows := m.matrixRoleRows()
		if index := firstRoleIndex(rows, candidates); index >= 0 {
			oldVisual := m.matrix.cursors[from].row - m.matrix.cursors[from].rowOffset
			m.matrix.cursors[active].row = index
			m.matrix.cursors[active].rowOffset = index - oldVisual
		}
	case matrixMembership:
		cols := m.matrixRoleCols()
		if index := firstRoleIndex(cols, candidates); index >= 0 {
			oldVisual := m.matrix.cursors[from].col - m.matrix.cursors[from].colOffset
			m.matrix.cursors[active].col = index
			m.matrix.cursors[active].colOffset = index - oldVisual
		}
	}
	m.clampMatrix()
}

// matrixSwitchCandidates prefers the principal in the current column (the
// user/role being toggled) and falls back to the selected row role.
func (m Model) matrixSwitchCandidates(from matrixKind) []domain.RoleRef {
	cursor := m.matrix.cursors[from]
	var candidates []domain.RoleRef
	switch from {
	case matrixMembership:
		if cols := m.matrixRoleCols(); cursor.col >= 0 && cursor.col < len(cols) {
			candidates = append(candidates, cols[cursor.col])
		}
		if rows := m.matrixRoleRows(); cursor.row >= 0 && cursor.row < len(rows) {
			candidates = append(candidates, rows[cursor.row])
		}
	case matrixPrivileges:
		if rows := m.matrixRoleRows(); cursor.row >= 0 && cursor.row < len(rows) {
			candidates = append(candidates, rows[cursor.row])
		}
	}
	return candidates
}

func firstRoleIndex(refs []domain.RoleRef, candidates []domain.RoleRef) int {
	for _, candidate := range candidates {
		if index := roleIndex(refs, candidate.Key()); index >= 0 {
			return index
		}
	}
	return -1
}

func (m *Model) moveMatrixCell(rowDelta, colDelta int) {
	cursor := &m.matrix.cursors[m.matrix.active]
	rows := m.matrixRowCount()
	cols := m.matrixColCount()
	if rows > 0 {
		cursor.row += rowDelta
		if cursor.row < 0 {
			cursor.row = 0
		}
		if cursor.row >= rows {
			cursor.row = rows - 1
		}
	}
	if cols > 0 {
		cursor.col += colDelta
		if cursor.col < 0 {
			cursor.col = 0
		}
		if cursor.col >= cols {
			cursor.col = cols - 1
		}
	}
	m.ensureMatrixVisible()
}

func (m *Model) pageMatrix(delta int) {
	rows, _ := m.matrixViewport()
	m.moveMatrixCell(rows*delta, 0)
}

func (m *Model) edgeMatrix(rowEnd, colEnd bool) {
	cursor := &m.matrix.cursors[m.matrix.active]
	if rowEnd {
		cursor.row = m.matrixRowCount() - 1
	} else {
		cursor.row = 0
	}
	if colEnd {
		cursor.col = m.matrixColCount() - 1
	} else {
		cursor.col = 0
	}
	m.clampMatrix()
}

func (m Model) handleMatrixKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cursor := &m.matrix.cursors[m.matrix.active]
	if cursor.filterPrompt {
		return m.handleMatrixFilterKey(msg)
	}
	m.message = ""
	m.messageLevel = levelNone
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.ForceQuit, keymap.Quit:
		return m, tea.Quit
	case keymap.Cancel:
		return m, m.closeMatrix()
	case keymap.Tab:
		m.switchMatrix((m.matrix.active + 1) % 2)
	case keymap.ShiftTab:
		m.switchMatrix((m.matrix.active + 1) % 2)
	case keymap.Tab1:
		m.switchMatrix(matrixMembership)
	case keymap.Tab2:
		m.switchMatrix(matrixPrivileges)
	case keymap.Up:
		m.moveMatrixCell(-1, 0)
	case keymap.Down:
		m.moveMatrixCell(1, 0)
	case keymap.Left:
		m.moveMatrixCell(0, -1)
	case keymap.Right:
		m.moveMatrixCell(0, 1)
	case keymap.PageUp:
		m.pageMatrix(-1)
	case keymap.PageDown:
		m.pageMatrix(1)
	case keymap.Home, keymap.RangeStart:
		m.edgeMatrix(false, false)
	case keymap.End, keymap.RangeEnd:
		m.edgeMatrix(true, true)
	case keymap.Search:
		cursor.filterPrompt = true
		cursor.filterCursor = len([]rune(cursor.filter))
	case keymap.Space:
		return m.toggleMatrixCell()
	case keymap.EditProfile, keymap.Confirm:
		return m.openMatrixDetails()
	}
	return m, nil
}

func (m Model) handleMatrixFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cursor := &m.matrix.cursors[m.matrix.active]
	if action, ok := keymap.Lookup(msg.String()); ok {
		switch action {
		case keymap.ForceQuit, keymap.Quit:
			return m, tea.Quit
		case keymap.Cancel:
			cursor.filter = ""
			cursor.filterPrompt = false
			cursor.filterCursor = 0
			cursor.row = 0
			cursor.col = 0
			cursor.rowOffset = 0
			cursor.colOffset = 0
			m.clampMatrix()
			return m, nil
		case keymap.Confirm:
			cursor.filterPrompt = false
			return m, nil
		}
	}
	value, position := editFilterText(cursor.filter, cursor.filterCursor, msg)
	cursor.filter = value
	cursor.filterCursor = position
	cursor.row = 0
	cursor.col = 0
	cursor.rowOffset = 0
	cursor.colOffset = 0
	m.clampMatrix()
	return m, nil
}

func (m Model) toggleMatrixCell() (tea.Model, tea.Cmd) {
	if m.session == nil {
		m.setMessage(levelError, "no active connection")
		return m, nil
	}
	if m.session.ReadOnly() {
		m.setMessage(levelError, "Blocked: the access matrix requires read-write mode. Ctrl+X Ctrl+W to switch.")
		return m, nil
	}
	cursor := m.matrix.cursors[m.matrix.active]
	if m.matrix.active == matrixMembership {
		rows := m.matrixRoleRows()
		cols := m.matrixRoleCols()
		if cursor.row >= len(rows) || cursor.col >= len(cols) {
			return m, nil
		}
		role := rows[cursor.row]
		member := cols[cursor.col]
		if member.Key() == role.Key() {
			m.setMessage(levelInfo, "a role cannot be its own member")
			return m, nil
		}
		granted := m.matrix.membership.Members[member.Key()][role.Key()]
		membership := domain.RoleMembership{Role: role, Member: member}
		if granted {
			if roles := m.matrix.membership.Members[member.Key()]; roles != nil {
				delete(roles, role.Key())
			}
			return m.startMatrixMutation(roleUnassign, func(session *app.Session) (domain.MutationResult, error) {
				return session.RevokeRole(context.Background(), membership)
			})
		}
		if m.matrix.membership.Members[member.Key()] == nil {
			m.matrix.membership.Members[member.Key()] = map[string]bool{}
		}
		m.matrix.membership.Members[member.Key()][role.Key()] = true
		return m.startMatrixMutation(roleAssign, func(session *app.Session) (domain.MutationResult, error) {
			return session.GrantRole(context.Background(), membership)
		})
	}
	rows := m.matrixRoleRows()
	cols := m.matrixObjectCols()
	if cursor.row >= len(rows) || cursor.col >= len(cols) {
		return m, nil
	}
	principal := rows[cursor.row]
	object := cols[cursor.col]
	if principal.Builtin {
		m.setMessage(levelInfo, "built-in roles are read-only")
		return m, nil
	}
	privilege := defaultPrivilege(m.engine(), object.Kind)
	if privilege == "" {
		m.setMessage(levelInfo, "no default privilege for this target")
		return m, nil
	}
	grant := domain.PrivilegeGrant{Role: principal, Target: object, Privileges: []string{privilege}}
	cells := m.matrix.privileges.Cells
	if cells[principal.Key()] == nil {
		cells[principal.Key()] = map[string][]string{}
	}
	values := cells[principal.Key()][object.Key()]
	if hasPrivilege(values, privilege) {
		cells[principal.Key()][object.Key()] = withoutPrivilege(values, privilege)
		return m.startMatrixMutation(roleRevoke, func(session *app.Session) (domain.MutationResult, error) {
			return session.RevokePrivileges(context.Background(), grant)
		})
	}
	cells[principal.Key()][object.Key()] = append(values, privilege)
	return m.startMatrixMutation(roleGrant, func(session *app.Session) (domain.MutationResult, error) {
		return session.GrantPrivileges(context.Background(), grant)
	})
}

func defaultPrivilege(engine domain.Engine, kind domain.ObjectKind) string {
	switch engine {
	case domain.EnginePostgreSQL:
		switch kind {
		case domain.KindDatabase:
			return "CONNECT"
		case domain.KindSchema:
			return "USAGE"
		case domain.KindTable:
			return "SELECT"
		}
	case domain.EngineMySQL:
		switch kind {
		case domain.KindDatabase, domain.KindTable:
			return "SELECT"
		}
	case domain.EngineMongoDB:
		switch kind {
		case domain.KindDatabase, domain.KindCollection, domain.KindTable:
			return "find"
		}
	}
	return ""
}

func hasPrivilege(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

func withoutPrivilege(values []string, want string) []string {
	kept := values[:0]
	for _, value := range values {
		if !strings.EqualFold(value, want) {
			kept = append(kept, value)
		}
	}
	return kept
}

func (m Model) openMatrixDetails() (tea.Model, tea.Cmd) {
	cursor := m.matrix.cursors[m.matrix.active]
	if m.matrix.active == matrixMembership {
		rows := m.matrixRoleRows()
		cols := m.matrixRoleCols()
		if cursor.row >= len(rows) || cursor.col >= len(cols) {
			return m, nil
		}
		role := rows[cursor.row]
		member := cols[cursor.col]
		m.matrix.details = matrixDetails{
			open:       true,
			membership: true,
			member:     member,
			role:       role,
			granted:    m.matrix.membership.Members[member.Key()][role.Key()],
		}
		m.modal = modalMatrixDetails
		return m, nil
	}
	rows := m.matrixRoleRows()
	cols := m.matrixObjectCols()
	if cursor.row >= len(rows) || cursor.col >= len(cols) {
		return m, nil
	}
	principal := rows[cursor.row]
	object := cols[cursor.col]
	if principal.Builtin {
		m.setMessage(levelInfo, "built-in roles are read-only")
		return m, nil
	}
	choices := domain.RolePrivilegeChoices(m.engine(), object.Kind)
	if len(choices) == 0 {
		m.setMessage(levelInfo, "no structured privileges for this target")
		return m, nil
	}
	current := append([]string(nil), m.matrix.privileges.Cells[principal.Key()][object.Key()]...)
	chosen := map[string]bool{}
	for _, choice := range choices {
		chosen[choice] = hasPrivilege(current, choice)
	}
	m.matrix.details = matrixDetails{
		open:    true,
		role:    principal,
		object:  object,
		choices: choices,
		chosen:  chosen,
		current: current,
	}
	m.modal = modalMatrixDetails
	return m, nil
}

func (m Model) handleMatrixDetailsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	details := &m.matrix.details
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.ForceQuit, keymap.Quit:
		return m, tea.Quit
	case keymap.Cancel, keymap.EditProfile:
		m.modal = modalNone
		*details = matrixDetails{}
	case keymap.Up, keymap.PageUp:
		step := 1
		if action == keymap.PageUp {
			step = 5
		}
		details.cursor -= step
		if details.cursor < 0 {
			details.cursor = len(details.choices) - 1
		}
		details.offset = matrixDetailsOffset(details.offset, details.cursor)
	case keymap.Down, keymap.PageDown:
		step := 1
		if action == keymap.PageDown {
			step = 5
		}
		details.cursor += step
		if details.cursor >= len(details.choices) {
			details.cursor = 0
		}
		details.offset = matrixDetailsOffset(details.offset, details.cursor)
	case keymap.Space:
		if details.membership {
			m.modal = modalNone
			*details = matrixDetails{}
			return m.toggleMatrixCell()
		}
		if details.cursor >= 0 && details.cursor < len(details.choices) {
			choice := details.choices[details.cursor]
			details.chosen[choice] = !details.chosen[choice]
		}
	case keymap.FetchAll:
		if details.membership {
			return m, nil
		}
		all := true
		for _, choice := range details.choices {
			if !details.chosen[choice] {
				all = false
				break
			}
		}
		for _, choice := range details.choices {
			details.chosen[choice] = !all
		}
	case keymap.Confirm:
		if details.membership {
			m.modal = modalNone
			*details = matrixDetails{}
			return m.toggleMatrixCell()
		}
		return m.applyMatrixDetails()
	}
	return m, nil
}

func matrixDetailsOffset(offset, cursor int) int {
	page := 10
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+page {
		offset = cursor - page + 1
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

func (m Model) applyMatrixDetails() (tea.Model, tea.Cmd) {
	details := m.matrix.details
	toGrant, toRevoke := diffPrivileges(details.current, details.choices, details.chosen)
	if len(toGrant) == 0 && len(toRevoke) == 0 {
		m.modal = modalNone
		m.matrix.details = matrixDetails{}
		return m, nil
	}
	principal := details.role
	object := details.object
	m.modal = modalNone
	m.matrix.details = matrixDetails{}
	return m.startMatrixMutation(roleGrant, func(session *app.Session) (domain.MutationResult, error) {
		var statements []string
		var affected int64
		if len(toGrant) > 0 {
			result, err := session.GrantPrivileges(context.Background(), domain.PrivilegeGrant{
				Role: principal, Target: object, Privileges: toGrant,
			})
			if err != nil {
				return result, err
			}
			statements = append(statements, result.Statement)
			affected += result.Affected
		}
		if len(toRevoke) > 0 {
			result, err := session.RevokePrivileges(context.Background(), domain.PrivilegeGrant{
				Role: principal, Target: object, Privileges: toRevoke,
			})
			if err != nil {
				return result, err
			}
			statements = append(statements, result.Statement)
			affected += result.Affected
		}
		return domain.MutationResult{Statement: strings.Join(statements, "; "), Affected: affected}, nil
	})
}

func diffPrivileges(current, choices []string, chosen map[string]bool) (toGrant, toRevoke []string) {
	have := map[string]bool{}
	for _, value := range current {
		have[strings.ToLower(value)] = true
	}
	for _, choice := range choices {
		want := chosen[choice]
		present := have[strings.ToLower(choice)]
		switch {
		case want && !present:
			toGrant = append(toGrant, choice)
		case !want && present:
			toRevoke = append(toRevoke, choice)
		}
	}
	return toGrant, toRevoke
}

func (m Model) startMatrixMutation(action roleAction, run func(*app.Session) (domain.MutationResult, error)) (tea.Model, tea.Cmd) {
	if m.matrix.busy {
		return m, nil
	}
	m.modal = modalNone
	m.matrix.details = matrixDetails{}
	m.matrix.busy = true
	return m.startRoleMutation(action, run)
}

func (m *Model) closeMatrix() tea.Cmd {
	m.matrixGen++
	m.screen = ScreenWorkspace
	m.modal = modalNone
	m.matrix.details = matrixDetails{}
	m.focus = FocusTree
	if !m.matrixDirty {
		return nil
	}
	m.matrixDirty = false
	commands := []tea.Cmd{
		m.refreshPrincipalGroup(domain.KindUsers),
		m.refreshPrincipalGroup(domain.KindRoles),
		m.refreshMatrixScope(),
		m.refreshCurrentPrivileges(),
	}
	return tea.Batch(commands...)
}

func matrixErrorText(err error) string {
	text := oneLine(err.Error())
	lower := strings.ToLower(text)
	if domain.KindOf(err) == domain.KindPermission ||
		strings.Contains(lower, "not permitted") ||
		strings.Contains(lower, "permission denied") {
		return text + " · this account cannot administer roles · Ctrl+D to reconnect with an admin profile"
	}
	return text
}

func filterMatrixRoles(refs []domain.RoleRef, filter string) []domain.RoleRef {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return refs
	}
	out := make([]domain.RoleRef, 0, len(refs))
	for _, ref := range refs {
		if strings.Contains(strings.ToLower(ref.Name), filter) {
			out = append(out, ref)
		}
	}
	return out
}

func filterMatrixObjects(refs []domain.ObjectRef, filter string) []domain.ObjectRef {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return refs
	}
	out := make([]domain.ObjectRef, 0, len(refs))
	for _, ref := range refs {
		label := strings.ToLower(matrixObjectLabel(ref) + " " + ref.Schema + " " + ref.Database)
		if strings.Contains(label, filter) {
			out = append(out, ref)
		}
	}
	if len(out) == 0 {
		// A principal filter should not hide every object column.
		return refs
	}
	return out
}

func (m Model) matrixObjects() ([]domain.ObjectRef, string) {
	engine := m.engine()
	database := m.currentDatabase()
	if database == "" {
		if node, ok := m.tree.selectedNode(); ok {
			database = node.Ref.Database
		}
	}
	schema := m.matrixSchema(engine)
	scope := database
	if schema != "" {
		scope += " · " + schema
	}
	if node, ok := m.tree.selectedNode(); ok {
		switch node.Kind {
		case domain.KindTable, domain.KindCollection:
			table := node.Name
			if node.Ref.Schema != "" {
				table = node.Ref.Schema + "." + table
			}
			if database != "" {
				scope = database + " · " + table
			} else {
				scope = table
			}
		case domain.KindSchema:
			if database != "" {
				scope = database + " · " + node.Name
			} else {
				scope = node.Name
			}
		}
	}
	if scope == "" {
		scope = "current connection"
	}
	var objects []domain.ObjectRef
	if database != "" {
		objects = append(objects, domain.ObjectRef{
			Engine: engine, Kind: domain.KindDatabase, Database: database, Name: database,
		})
	}
	if engine == domain.EnginePostgreSQL && schema != "" {
		objects = append(objects, domain.ObjectRef{
			Engine: engine, Kind: domain.KindSchema, Database: database, Schema: schema, Name: schema,
		})
	}
	for _, node := range m.tree.nodes {
		ref := node.Node.Ref
		if !matrixObjectKind(engine, ref.Kind) {
			continue
		}
		if ref.Database != database {
			continue
		}
		if engine == domain.EnginePostgreSQL && schema != "" && ref.Schema != schema {
			continue
		}
		objects = append(objects, ref)
	}
	seen := map[string]bool{}
	filtered := objects[:0]
	for _, object := range objects {
		key := object.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		filtered = append(filtered, object)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left, right := matrixObjectRank(filtered[i]), matrixObjectRank(filtered[j])
		if left != right {
			return left < right
		}
		if filtered[i].Schema != filtered[j].Schema {
			return filtered[i].Schema < filtered[j].Schema
		}
		return filtered[i].Name < filtered[j].Name
	})
	return filtered, scope
}

func (m Model) matrixSchema(engine domain.Engine) string {
	if engine != domain.EnginePostgreSQL {
		return ""
	}
	if node, ok := m.tree.selectedNode(); ok {
		if node.Ref.Schema != "" {
			return node.Ref.Schema
		}
		if node.Kind == domain.KindSchema {
			return node.Name
		}
	}
	return "public"
}

func matrixObjectKind(engine domain.Engine, kind domain.ObjectKind) bool {
	switch kind {
	case domain.KindTable:
		return true
	case domain.KindCollection:
		return engine == domain.EngineMongoDB
	default:
		return false
	}
}

func matrixObjectRank(ref domain.ObjectRef) int {
	switch ref.Kind {
	case domain.KindDatabase:
		return 0
	case domain.KindSchema:
		return 1
	case domain.KindTable, domain.KindCollection:
		return 2
	case domain.KindView:
		return 3
	default:
		return 4
	}
}

func (m Model) matrixTabsLine() string {
	membership := "1 Role × Principal"
	privileges := "2 Principal/Role × Object"
	if m.matrix.active == matrixMembership {
		membership = m.styles.Accent.Bold(true).Underline(true).Render(membership)
		privileges = m.styles.Label.Render(privileges)
	} else {
		membership = m.styles.Label.Render(membership)
		privileges = m.styles.Accent.Bold(true).Underline(true).Render(privileges)
	}
	return membership + m.styles.Muted.Render(" · ") + privileges +
		m.styles.Muted.Render("   Tab switch")
}

func (m Model) matrixHeaderLines(width int) []string {
	tabs := m.matrixTabsLine()
	if full, ok := m.matrixTruncatedColumnLabel(); ok {
		hint := m.styles.Muted.Render("column ") + m.styles.Text.Render(full)
		separator := m.styles.Muted.Render(" · ")
		if lipgloss.Width(tabs)+lipgloss.Width(separator)+lipgloss.Width(hint) <= width {
			return []string{tabs + separator + hint}
		}
		return []string{tabs, hint}
	}
	return []string{tabs}
}

func (m Model) matrixBodyLines(width int) []string {
	lines := m.matrixHeaderLines(width)
	cursor := m.matrix.cursors[m.matrix.active]
	if m.matrixFilterVisible() {
		value := m.styles.Text.Render(filterText(cursor.filter))
		if cursor.filterPrompt {
			value = m.inlineInput(cursor.filter, cursor.filterCursor)
		}
		lines = append(lines, m.styles.Label.Render("filter ")+value)
	}
	if m.matrix.active == matrixMembership && m.matrix.membershipErr != "" {
		return append(lines, m.matrixErrorLines(m.matrix.membershipErr, width)...)
	}
	if m.matrix.active == matrixPrivileges && m.matrix.privilegesErr != "" {
		return append(lines, m.matrixErrorLines(m.matrix.privilegesErr, width)...)
	}
	if m.matrix.active == matrixMembership && m.matrix.membershipLoading && !m.matrix.membershipLoaded {
		return append(lines, m.styles.Muted.Render("loading memberships…"))
	}
	if m.matrix.active == matrixPrivileges && m.matrix.privilegesLoading && !m.matrix.privilegesLoaded {
		return append(lines, m.styles.Muted.Render("loading privileges…"))
	}
	rows := m.matrixRoleRows()
	cols := m.matrixColCount()
	if len(rows) == 0 || cols == 0 {
		return append(lines, m.styles.Muted.Render("no principals or objects visible in this scope"))
	}
	principalWidth := m.matrixPrincipalWidth()
	cellWidth := m.matrixCellWidth()
	rowViewport, colViewport := m.matrixViewport()
	colStart := cursor.colOffset
	colEnd := colStart + colViewport
	if colEnd > cols {
		colEnd = cols
	}
	firstHeader := "principal/role"
	if m.matrix.active == matrixMembership {
		firstHeader = "role"
	}
	header := Pad(m.styles.Header.Render(firstHeader), principalWidth) + " "
	for col := colStart; col < colEnd; col++ {
		label := Truncate(m.matrixColLabel(col), cellWidth)
		style := m.styles.Header
		if col == cursor.col {
			style = m.styles.Accent.Bold(true)
		}
		header += style.Render(Pad(label, cellWidth)) + " "
	}
	lines = append(lines, header)
	rowStart := cursor.rowOffset
	rowEnd := rowStart + rowViewport
	if rowEnd > len(rows) {
		rowEnd = len(rows)
	}
	for row := rowStart; row < rowEnd; row++ {
		label := Truncate(m.matrixRowLabel(row), principalWidth)
		style := m.styles.Text
		if row == cursor.row {
			style = m.styles.Accent.Bold(true)
		}
		line := style.Render(Pad(label, principalWidth)) + " "
		for col := colStart; col < colEnd; col++ {
			cell := m.matrixCellText(row, col)
			rendered := m.styles.Text.Render(Pad(cell, cellWidth))
			if row == cursor.row && col == cursor.col {
				rendered = m.styles.Cursor.Render(Pad(cell, cellWidth))
			}
			line += rendered + " "
		}
		lines = append(lines, line)
	}
	return lines
}

func (m Model) matrixErrorLines(text string, width int) []string {
	wrapped := wrapText(text, width)
	lines := make([]string, 0, len(wrapped))
	for _, line := range wrapped {
		lines = append(lines, m.styles.Error.Render(line))
	}
	return lines
}

func (m Model) viewMatrix() string {
	top := m.viewTopStatus(m.width)
	body := strings.Join(m.matrixBodyLines(m.width-4), "\n")
	box := Box("access matrix · "+oneLine(m.matrix.scope), body, m.width, m.height-2, m.styles, true)
	return strings.Join([]string{top, box, m.matrixFooter(m.width)}, "\n")
}

func (m Model) matrixFooter(width int) string {
	left := m.message
	style := m.messageStyle()
	if left == "" {
		left = fmt.Sprintf("scope %s · %s", oneLine(m.matrix.scope), m.matrixScopeText())
		style = m.styles.Accent
		if (m.matrix.active == matrixMembership && m.matrix.membershipLoading) ||
			(m.matrix.active == matrixPrivileges && m.matrix.privilegesLoading) {
			left = "loading matrix…"
			style = m.styles.Info
		}
	}
	right := "Tab switch · 1 membership · 2 privileges · Space toggle · e/Enter details · / filter · Esc close"
	if width < 120 {
		right = "Tab/1/2 · Space · e/Enter details · / filter · Esc close"
	}
	if m.matrix.cursors[m.matrix.active].filterPrompt {
		right = "type to filter · Enter keep · Esc clear"
	}
	if strings.Contains(left, "Ctrl+D") {
		right = "Ctrl+D admin profile · Esc close"
	}
	return ComposeLR(style.Render(oneLine(left)), m.styles.Muted.Render(right), width)
}

func (m Model) matrixScopeText() string {
	if m.matrix.active == matrixMembership {
		return "membership"
	}
	return "privileges"
}

func (m Model) viewMatrixDetails() string {
	width := modalWidth(m.width, 72)
	inner := width - 4
	details := m.matrix.details
	var body []string
	if details.membership {
		body = append(body, field("member", 10)+" "+m.styles.Text.Render(oneLine(details.member.Name)))
		body = append(body, field("role", 10)+" "+m.styles.Text.Render(oneLine(details.role.Name)))
		state := "[ ]"
		if details.granted {
			state = "[x]"
		}
		body = append(body, field("state", 10)+" "+m.styles.Text.Render(state))
		body = append(body, "")
		for _, line := range wrapText("Space toggles this membership. Native admin options such as WITH ADMIN OPTION or INHERIT are not editable in this slice.", inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
		body = append(body, "", m.actionLine("Space", "toggle", "Enter", "toggle/close"))
		body = append(body, m.styles.Muted.Render("e/Esc close"))
		return modalBox("membership details", body, width, m.styles)
	}
	body = append(body, field("principal", 10)+" "+m.styles.Text.Render(oneLine(details.role.Name)))
	body = append(body, field("target", 10)+" "+m.styles.Text.Render(oneLine(matrixObjectLabel(details.object))))
	body = append(body, "")
	body = append(body, m.styles.Header.Render("native privileges · Space toggle · Ctrl+A all"))
	body = append(body, m.viewMatrixChoices(inner)...)
	if details.err != "" {
		body = append(body, m.styles.Error.Render(Truncate(details.err, inner)))
	}
	body = append(body, "", m.actionLine("Enter", "apply", "Ctrl+A", "toggle all"))
	body = append(body, m.styles.Muted.Render("e/Esc cancel"))
	return modalBox("privilege details", body, width, m.styles)
}

func (m Model) viewMatrixChoices(width int) []string {
	details := m.matrix.details
	if len(details.choices) == 0 {
		return []string{m.styles.Muted.Render("no editable privileges for this target")}
	}
	start := details.offset
	end := start + 10
	if end > len(details.choices) {
		end = len(details.choices)
	}
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		mark := "[ ]"
		if details.chosen[details.choices[i]] {
			mark = "[x]"
		}
		line := "  " + mark + " " + details.choices[i]
		if i == details.cursor {
			line = "> " + mark + " " + details.choices[i]
			lines = append(lines, m.styles.Select.Render(Pad(line, width)))
		} else {
			lines = append(lines, m.styles.Text.Render(line))
		}
	}
	return lines
}
