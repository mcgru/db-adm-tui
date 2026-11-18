package tui

import (
	"context"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func (m Model) treePage() int {
	if m.height > 12 {
		return m.height - 8
	}
	return 4
}

func (m Model) detailsPage() int {
	if m.height > 12 {
		return m.height - 8
	}
	return 4
}

func (m *Model) syncDetails() tea.Cmd {
	node, ok := m.tree.selectedNode()
	if !ok {
		m.details.clear()
		m.detailsGen++
		m.resetData()
		return nil
	}
	if m.details.has && m.details.node.Ref.Key() == node.Ref.Key() {
		m.details.node = node
		return nil
	}
	m.details.selectNode(node)
	m.detailsGen++
	if node.Kind == domain.KindTable || node.Kind == domain.KindCollection {
		m.resetDataFor(node.Ref)
	} else {
		m.resetData()
	}
	return m.ensureTabInfo()
}

func (m *Model) selectTab(index int) tea.Cmd {
	if !m.details.has {
		m.setMessage(levelInfo, "select an object first")
		return nil
	}
	if index < 0 || index >= len(m.details.tabs) {
		m.setMessage(levelInfo, m.details.tabNote(index))
		return nil
	}
	m.details.tab = index
	m.details.offset = 0
	return tea.Batch(m.ensureData(), m.ensureTabInfo())
}

func (m *Model) cycleTab(delta int) tea.Cmd {
	if !m.details.has || len(m.details.tabs) == 0 {
		m.setMessage(levelInfo, "select an object first")
		return nil
	}
	m.details.tab = (m.details.tab + delta + len(m.details.tabs)) % len(m.details.tabs)
	m.details.offset = 0
	return tea.Batch(m.ensureData(), m.ensureTabInfo())
}

func (m *Model) scrollDetails(delta int) {
	if !m.details.has {
		return
	}
	var lines []detailLine
	width := m.dataInnerWidth()
	switch m.details.currentTab() {
	case "Details":
		lines = detailLines(&m.details, m.currentDatabase(), m.tree.childCount(m.details.node.Ref.Key()), width)
	case "Indexes":
		lines = indexLines(&m.details, width)
	default:
		return
	}
	height := m.detailsPage()
	m.details.scroll(delta, height, len(lines))
}

func (m *Model) collapseSelected() tea.Cmd {
	node, ok := m.tree.selectedNode()
	if !ok {
		return nil
	}
	key := node.Ref.Key()
	if m.tree.collapse(key) {
		return m.syncDetails()
	}
	if tn, ok := m.tree.node(key); ok && tn.Parent != "" {
		m.tree.selectKey(tn.Parent)
		return m.syncDetails()
	}
	return nil
}

func (m Model) expandSelected() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok {
		if m.tree.rootErr != "" {
			return m.loadRoot()
		}
		return m, nil
	}
	if !node.Expandable {
		if node.IsObject() || node.IsPrincipal() {
			return m, m.syncDetails()
		}
		return m, nil
	}
	key := node.Ref.Key()
	if node.Kind == domain.KindSecurity {
		m.addSecurityGroups(key)
		return m, nil
	}
	tn, ok := m.tree.node(key)
	if !ok {
		return m, nil
	}
	if tn.Err != "" || !tn.Loaded {
		m.tree.setLoading(key, true)
		return m, m.loadChildrenCmd(node)
	}
	if tn.Expanded {
		return m, nil
	}
	m.tree.expand(key)
	return m, m.syncDetails()
}

func (m Model) openSelected() (tea.Model, tea.Cmd) {
	node, ok := m.tree.selectedNode()
	if !ok {
		if m.tree.rootErr != "" {
			return m.loadRoot()
		}
		return m, nil
	}
	if node.Expandable {
		return m.expandSelected()
	}
	syncCmd := m.syncDetails()
	if node.IsObject() || node.IsPrincipal() {
		m.focus = FocusMain
		if index := tabIndex(m.details.tabs, "Data"); index >= 0 {
			m.details.tab = index
			m.details.offset = 0
			return m, tea.Batch(syncCmd, m.ensureData())
		}
	}
	return m, syncCmd
}

func tabIndex(tabs []string, want string) int {
	for i, tab := range tabs {
		if tab == want {
			return i
		}
	}
	return -1
}

func (m Model) loadRoot() (tea.Model, tea.Cmd) {
	if m.session == nil {
		return m, nil
	}
	m.treeGen++
	gen := m.treeGen
	m.tree.reset()
	m.tree.loadingRoot = true
	m.details.clear()
	m.detailsGen++
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return m, func() tea.Msg {
		nodes, err := session.Databases(ctx)
		return treeRootMsg{gen: gen, nodes: nodes, err: err}
	}
}

func (m Model) loadChildrenCmd(node domain.Node) tea.Cmd {
	session := m.session
	ctx := m.sessionCtx
	if ctx == nil {
		ctx = context.Background()
	}
	gen := m.treeGen
	key := node.Ref.Key()
	if node.Kind == domain.KindUsers || node.Kind == domain.KindRoles {
		return func() tea.Msg {
			principals, err := session.Principals(ctx, node.Ref)
			return treePrincipalsMsg{gen: gen, key: key, principals: principals, err: err}
		}
	}
	return func() tea.Msg {
		nodes, err := session.Children(ctx, node.Ref)
		return treeChildrenMsg{gen: gen, key: key, nodes: nodes, err: err}
	}
}

func (m Model) handleTreeRoot(msg treeRootMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.treeGen {
		return m, nil
	}
	m.tree.loadingRoot = false
	if msg.err != nil {
		if m.session != nil && !m.session.Alive() {
			return m.leaveWorkspace(msg.err)
		}
		m.tree.rootErr = treeErrorMessage(msg.err)
		m.setMessage(levelError, m.tree.rootErr)
		return m, nil
	}
	nodes := m.stampNodes(msg.nodes)
	m.tree.setRoots(nodes)
	m.tree.addRoot(domain.Node{
		Ref:        domain.ObjectRef{Engine: m.engine(), Kind: domain.KindSecurity, Name: "security"},
		Name:       "Security",
		Kind:       domain.KindSecurity,
		Expandable: true,
	})
	m.tree.rebuild()
	m.tree.selectFirst()
	if m.messageLevel != levelError && m.messageLevel != levelWarning {
		m.setMessage(levelSuccess, "connected · "+databaseCount(len(nodes)))
	}
	restoreCmd := m.startRestore()
	syncCmd := m.syncDetails()
	if restoreCmd != nil {
		return m, tea.Batch(syncCmd, restoreCmd)
	}
	return m, syncCmd
}

func (m Model) handleTreeChildren(msg treeChildrenMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.treeGen {
		return m, nil
	}
	if msg.err != nil {
		if m.session != nil && !m.session.Alive() {
			return m.leaveWorkspace(msg.err)
		}
		m.tree.setError(msg.key, treeErrorMessage(msg.err))
		m.setMessage(levelError, treeErrorMessage(msg.err))
		return m, nil
	}
	m.tree.addChildren(msg.key, m.stampNodes(msg.nodes))
	if node, ok := m.tree.node(msg.key); ok && node.Node.Kind == domain.KindDatabase {
		m.setMessage(levelInfo, "database "+node.Node.Ref.Database)
	}
	restoreCmd := m.advanceRestore()
	syncCmd := m.syncDetails()
	if restoreCmd != nil {
		return m, tea.Batch(syncCmd, restoreCmd)
	}
	return m, syncCmd
}

func (m Model) handleTreePrincipals(msg treePrincipalsMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.treeGen {
		return m, nil
	}
	if msg.err != nil {
		if m.session != nil && !m.session.Alive() {
			return m.leaveWorkspace(msg.err)
		}
		m.tree.setError(msg.key, treeErrorMessage(msg.err))
		m.setMessage(levelError, treeErrorMessage(msg.err))
		return m, nil
	}
	var nodes []domain.Node
	for _, principal := range msg.principals {
		ref := domain.ObjectRef{
			Engine:   m.engine(),
			Kind:     principal.Kind,
			Database: principal.Database,
			Name:     principal.Name,
		}
		nodes = append(nodes, domain.Node{
			Ref:        ref,
			Name:       principal.Name,
			Kind:       principal.Kind,
			Scope:      principal.Scope,
			Attributes: principal.Attributes,
		})
	}
	m.tree.addChildren(msg.key, nodes)
	m.tree.expand(msg.key)
	return m, m.syncDetails()
}

func (m Model) stampNodes(nodes []domain.Node) []domain.Node {
	engine := m.engine()
	for i := range nodes {
		if nodes[i].Ref.Engine == "" {
			nodes[i].Ref.Engine = engine
		}
	}
	return nodes
}

func (m Model) engine() domain.Engine {
	if m.session != nil {
		return m.session.ProfileSnapshot().Engine
	}
	return m.active.Engine
}

func (m Model) currentDatabase() string {
	if m.session != nil {
		return m.session.CurrentDatabase()
	}
	return m.active.Database
}

func (m Model) profileForStatus() domain.Profile {
	if m.session != nil {
		p := m.session.ProfileSnapshot()
		if p.Engine.Valid() {
			return p
		}
	}
	if m.active.Engine.Valid() {
		return m.active
	}
	if len(m.profiles) > 0 {
		return m.profiles[0].Profile
	}
	return domain.Profile{Label: "—", Engine: domain.EnginePostgreSQL, User: "—"}
}

func (m Model) leaveWorkspace(err error) (tea.Model, tea.Cmd) {
	m.savePlace()
	if m.sessionCancel != nil {
		m.sessionCancel()
		m.sessionCancel = nil
	}
	key := m.active.Key()
	if m.session != nil {
		_ = m.session.Close()
		m.session = nil
	}
	m.treeGen++
	m.screen = ScreenPicker
	m.focus = FocusTree
	m.tree.reset()
	m.details.clear()
	m.detailsGen++
	m.restore = nil
	m.resetData()
	m.reloadProfiles(context.Background(), key)
	m.rowErrors[key] = oneLine(err.Error())
	m.setMessage(levelError, oneLine(err.Error()))
	return m, nil
}

func treeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if domain.KindOf(err) == domain.KindPermission {
		return "not permitted"
	}
	return oneLine(err.Error())
}

func (m *Model) addSecurityGroups(key string) {
	engine := m.engine()
	m.tree.addChildren(key, []domain.Node{
		{
			Ref:        domain.ObjectRef{Engine: engine, Kind: domain.KindUsers, Name: "users"},
			Name:       "users",
			Kind:       domain.KindUsers,
			Expandable: true,
		},
		{
			Ref:        domain.ObjectRef{Engine: engine, Kind: domain.KindRoles, Name: "roles"},
			Name:       "roles",
			Kind:       domain.KindRoles,
			Expandable: true,
		},
	})
	m.tree.expand(key)
	m.syncDetails()
}

func databaseCount(n int) string {
	if n == 1 {
		return "1 database"
	}
	return strconv.Itoa(n) + " databases"
}
