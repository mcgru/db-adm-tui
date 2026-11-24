package tui

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

// Screen is the current top-level surface.
type Screen int

const (
	ScreenPicker Screen = iota
	ScreenWorkspace
	ScreenMatrix
)

// Focus is the workspace focus zone.
type Focus int

const (
	FocusTree Focus = iota
	FocusMain
	FocusQuery
)

type modalKind int

const (
	modalNone modalKind = iota
	modalImport
	modalNew
	modalPassword
	modalSave
	modalEdit
	modalDeleteProfile
	modalStatement
	modalHelp
	modalMode
	modalRoleMenu
	modalRoleCreate
	modalRoleConfirm
	modalMatrixDetails
	modalUserForm
	modalUserConfirm
	modalUserSave
	modalUserHome
	modalRowInsert
	modalRowEdit
	modalRowDelete
)

type messageLevel int

const (
	levelNone messageLevel = iota
	levelInfo
	levelSuccess
	levelWarning
	levelError
)

type connectOrigin int

const (
	originProfile connectOrigin = iota
	originPassword
	originNew
)

type chordTimeoutMsg struct {
	seq int
}

type connectDoneMsg struct {
	gen     uint64
	origin  connectOrigin
	profile domain.Profile
	result  app.ConnectResult
	err     error
}

type queryDoneMsg struct {
	gen       uint64
	statement string
	page      domain.QueryPage
	err       error
}

type resultsPageMsg struct {
	gen  uint64
	page domain.QueryPage
	err  error
}

type queryTickMsg struct {
	gen uint64
}

type escChordTimeoutMsg struct {
	seq int
}

type modeDoneMsg struct {
	gen      uint64
	readOnly bool
	err      error
}

type treeRootMsg struct {
	gen   uint64
	nodes []domain.Node
	err   error
}

type treeChildrenMsg struct {
	gen   uint64
	key   string
	nodes []domain.Node
	err   error
}

type treePrincipalsMsg struct {
	gen        uint64
	key        string
	principals []domain.Principal
	err        error
}

// Model is the root Bubble Tea model. It owns view state only; the app service
// owns storage, credential resolution, and connections.
type Model struct {
	width, height int
	screen        Screen
	focus         Focus
	styles        theme.Styles
	svc           *app.Service

	profiles     []app.ProfileView
	pickerSel    int
	pickerOffset int
	rowErrors    map[string]string
	storageErr   bool
	message      string
	messageLevel messageLevel
	pendingChord bool
	chordSeq     int

	// caseInsensitive is the session-wide search case mode. It starts on for
	// every session and applies to both the buffer and server filters.
	caseInsensitive bool

	active        domain.Profile
	session       *app.Session
	sessionCtx    context.Context
	sessionCancel context.CancelFunc
	tree          treeState
	details       detailsState
	data          dataState
	dataNotice    string
	dataNoticeLvl messageLevel
	treeGen       uint64
	dataGen       uint64
	detailsGen    uint64
	exactGen      uint64
	restore       *restoreState

	query           textinput.Model
	statementInput  textinput.Model
	statementOrigin string
	role            roleState
	matrix          matrixState
	user            userState
	userHome        userHomeState
	rowInsert       rowInsertState
	rowEdit         rowEditState
	rowDelete       rowDeleteState
	rowEditBlink    uint64
	history         []string
	historyIndex    int
	historyDraft    string
	completion      completionState
	helpTerm        string
	helpBody        string
	helpReturn      modalKind
	helpExtended    bool
	helpScroll      int

	resultsActive  bool
	results        resultsState
	resultsSaved   detailsState
	resultsData    dataState
	resultsDataGen uint64
	queryGen       uint64
	resultsGen     uint64
	queryRunning   bool
	queryStarted   time.Time
	escChord       bool
	escSeq         int
	elevated       bool
	rootLaunch     bool
	modeBusy       bool
	modeGen        uint64
	saveResult     app.ConnectResult
	treeWidth      int
	queryHeight    int
	layoutKey      string
	roleGen        uint64
	matrixGen      uint64
	matrixDirty    bool
	userGen        uint64
	rowGen         uint64

	connecting      bool
	connectingKey   string
	connectingLabel string
	connectGen      uint64
	cancel          context.CancelFunc

	modal            modalKind
	importEntries    []app.ImportEntry
	importStatus     []string
	importSel        map[string]bool
	importCursor     int
	importOffset     int
	importEmpty      bool
	importNotice     string
	importAllowNoTLS bool

	newEngine        domain.Engine
	newLabel         textinput.Model
	newHost          textinput.Model
	newPort          textinput.Model
	newDatabase      textinput.Model
	newUser          textinput.Model
	newAuth          textinput.Model
	newPassword      textinput.Model
	newAllowNoTLS    bool
	newFocus         newField
	newNeedsPassword bool
	newIgnoreSource  bool
	newBusy          bool
	newErr           string

	passwordProfile domain.Profile
	passwordInput   textinput.Model
	passwordBusy    bool

	saveProfile domain.Profile
	saveSession *app.Session

	editProfile     domain.Profile
	editOriginalKey string
	editLabel       textinput.Model
	editHost        textinput.Model
	editPort        textinput.Model
	editDatabase    textinput.Model
	editUser        textinput.Model
	editAuth        textinput.Model
	editAllowNoTLS  bool
	editFocus       editField
	editErr         string

	deleteProfile domain.Profile
	deleteErr     string
}

// New builds the root model around the application service and loads stored
// profiles synchronously.
func New(svc *app.Service) Model {
	m := Model{
		width:           layout.MinWidth,
		height:          layout.MinHeight,
		screen:          ScreenPicker,
		focus:           FocusTree,
		styles:          theme.New(),
		svc:             svc,
		rowErrors:       map[string]string{},
		importSel:       map[string]bool{},
		tree:            newTree(),
		data:            newDataState(),
		newFocus:        fieldEngine,
		caseInsensitive: true,
		rootLaunch:      os.Geteuid() == 0,
		treeWidth:       layout.TreeWidth,
		queryHeight:     layout.QueryLines,
	}
	m.newEngine = domain.EnginePostgreSQL
	m.newLabel = newInput("label", false)
	m.newHost = newInput("host", false)
	m.newPort = newInput("5432", false)
	m.newDatabase = newInput("database", false)
	m.newUser = newInput("user", false)
	m.newAuth = newMutedInput("auth.db")
	m.newPassword = newInput("password", true)
	m.passwordInput = newInput("", true)
	m.query = newInput("", false)
	m.statementInput = newInput("", false)
	m.role = newRoleState()
	m.matrix = newMatrixState()
	m.user = newUserState()
	m.editLabel = newInput("label", false)
	m.editHost = newInput("host", false)
	m.editPort = newInput("5432", false)
	m.editDatabase = newInput("database", false)
	m.editUser = newInput("user", false)
	m.editAuth = newMutedInput("auth.db")
	m.resizeInputs()
	m.reloadProfiles(context.Background(), "")
	return m
}

func newInput(placeholder string, secret bool) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.SetVirtualCursor(true)
	input.SetWidth(60)
	styles := input.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Text))
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Cursor.Color = lipgloss.Color(theme.Nightshift.Accent)
	styles.Cursor.Blink = false
	input.SetStyles(styles)
	if secret {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '•'
	}
	return input
}

func newMutedInput(placeholder string) textinput.Model {
	input := newInput(placeholder, false)
	styles := input.Styles()
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Nightshift.Muted))
	styles.Focused.Text = muted
	styles.Blurred.Text = muted
	styles.Focused.Placeholder = muted
	styles.Blurred.Placeholder = muted
	styles.Cursor.Color = lipgloss.Color(theme.Nightshift.Muted)
	input.SetStyles(styles)
	return input
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.data.endDown = 0
		m.data.endPg = 0
		m.resizeInputs()
		if m.screen == ScreenMatrix {
			m.clampMatrix()
		}
		if m.regions().TooSmall {
			m.dismissModal()
			if m.connecting {
				m.cancelConnect()
			}
		}
		return m, nil
	case chordTimeoutMsg:
		if m.pendingChord && msg.seq == m.chordSeq {
			m.pendingChord = false
			m.setMessage(levelInfo, "cancelled")
		}
		return m, nil
	case connectDoneMsg:
		return m.handleConnectDone(msg)
	case treeRootMsg:
		return m.handleTreeRoot(msg)
	case treeChildrenMsg:
		return m.handleTreeChildren(msg)
	case treePrincipalsMsg:
		return m.handleTreePrincipals(msg)
	case dataPageMsg:
		return m.handleDataPage(msg)
	case structureMsg:
		return m.handleStructure(msg)
	case indexesMsg:
		return m.handleIndexes(msg)
	case statisticsMsg:
		return m.handleStatistics(msg)
	case exactCountMsg:
		return m.handleExactCount(msg)
	case exactTickMsg:
		return m.handleExactTick(msg)
	case privilegesMsg:
		return m.handlePrivileges(msg)
	case queryDoneMsg:
		return m.handleQueryDone(msg)
	case resultsPageMsg:
		return m.handleResultsPage(msg)
	case queryTickMsg:
		return m.handleQueryTick(msg)
	case escChordTimeoutMsg:
		return m.handleEscChordTimeout(msg)
	case modeDoneMsg:
		return m.handleModeDone(msg)
	case roleDoneMsg:
		return m.handleRoleDone(msg)
	case userDoneMsg:
		return m.handleUserDone(msg)
	case rowDoneMsg:
		return m.handleRowDone(msg)
	case rowEditTickMsg:
		return m.handleRowEditTick(msg)
	case matrixMembershipMsg:
		return m.handleMatrixMembership(msg)
	case matrixPrivilegesMsg:
		return m.handleMatrixPrivileges(msg)
	case tea.PasteMsg:
		return m.handlePaste(msg)
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.regions().TooSmall {
		action, ok := keymap.Lookup(key)
		if !ok {
			return m, nil
		}
		switch action {
		case keymap.ForceQuit, keymap.Quit:
			return m, tea.Quit
		}
		return m, nil
	}
	if m.screen == ScreenMatrix {
		if m.modal == modalMatrixDetails {
			return m.handleMatrixDetailsKey(msg)
		}
		return m.handleMatrixKey(msg)
	}
	if m.modal != modalNone {
		m.data.endDown = 0
		m.data.endPg = 0
	} else {
		m.resetDataGestures(key)
	}
	switch m.modal {
	case modalNew:
		return m.handleNewKey(msg)
	case modalPassword:
		return m.handlePasswordKey(msg)
	case modalImport:
		return m.handleImportKey(msg)
	case modalSave:
		return m.handleSaveKey(msg)
	case modalEdit:
		return m.handleEditKey(msg)
	case modalDeleteProfile:
		return m.handleDeleteProfileKey(msg)
	case modalStatement:
		return m.handleStatementKey(msg)
	case modalHelp:
		return m.handleHelpKey(msg)
	case modalMode:
		return m.handleModeKey(msg)
	case modalRoleMenu:
		return m.handleRoleMenuKey(msg)
	case modalRoleCreate:
		return m.handleRoleCreateKey(msg)
	case modalRoleConfirm:
		return m.handleRoleConfirmKey(msg)
	case modalUserForm:
		return m.handleUserFormKey(msg)
	case modalUserConfirm:
		return m.handleUserConfirmKey(msg)
	case modalUserSave:
		return m.handleUserSaveKey(msg)
	case modalUserHome:
		return m.handleUserHomeKey(msg)
	case modalRowInsert:
		return m.handleRowInsertKey(msg)
	case modalRowEdit:
		return m.handleRowEditKey(msg)
	case modalRowDelete:
		return m.handleRowDeleteKey(msg)
	}
	if m.pendingChord {
		m.pendingChord = false
		action, ok := keymap.LookupChord(key)
		if !ok {
			m.setMessage(levelInfo, "cancelled")
			return m, nil
		}
		return m.apply(action)
	}
	if action, ok := keymap.Lookup(key); ok && action == keymap.ChordX {
		m.chordSeq++
		seq := m.chordSeq
		m.pendingChord = true
		m.escChord = false
		m.setMessage(levelInfo, "Ctrl+X …")
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return chordTimeoutMsg{seq: seq}
		})
	}
	if m.tree.searchPrompt {
		if next, cmd, handled := m.handleTreeSearchKey(msg); handled {
			return next, cmd
		}
	}
	if m.completion.active {
		if next, cmd, handled := m.handleCompletionKey(msg); handled {
			return next, cmd
		}
	}
	if action, ok := keymap.Lookup(key); ok {
		switch action {
		case keymap.ResizeTreeLeft, keymap.ResizeTreeRight,
			keymap.ResizeQueryUp, keymap.ResizeQueryDown:
			return m.apply(action)
		}
	}
	if m.screen == ScreenWorkspace && m.modal == modalNone && m.focus == FocusQuery {
		return m.handleQueryKey(msg)
	}
	if m.data.prompt != promptNone {
		if next, cmd, handled := m.handleFilterKey(msg); handled {
			return next, cmd
		}
	}
	m.message = ""
	m.messageLevel = levelNone
	action, ok := keymap.Lookup(key)
	if !ok {
		return m, nil
	}
	return m.apply(action)
}

func (m Model) apply(action keymap.Action) (tea.Model, tea.Cmd) {
	switch action {
	case keymap.ForceQuit:
		return m, tea.Quit
	case keymap.Quit:
		if m.screen == ScreenWorkspace {
			m.savePlace()
		}
		return m, tea.Quit
	case keymap.Confirm:
		if m.screen == ScreenPicker {
			return m.enterPicker()
		}
		if m.screen == ScreenWorkspace {
			return m.openSelected()
		}
	case keymap.New:
		if m.screen == ScreenWorkspace && m.focus == FocusTree &&
			m.tree.searchPattern != "" && !m.tree.searchPrompt {
			return m, m.nextTreeMatch(1)
		}
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			if node, ok := m.tree.selectedNode(); ok &&
				(node.Kind == domain.KindSecurity || node.Kind == domain.KindUsers) {
				return m.startUserAdd()
			}
		}
		if m.filterKeepsNewKey() {
			if (m.data.bufferPattern != "" || m.data.serverFilter != nil) && m.data.prompt == promptNone {
				return m, m.nextMatch(1)
			}
			return m, nil
		}
		if m.screen == ScreenPicker && !m.connecting {
			m.openNewModal()
		}
	case keymap.Import:
		if m.screen == ScreenPicker && !m.connecting {
			m.openImportModal()
		}
	case keymap.EditProfile:
		if m.screen == ScreenPicker && !m.connecting {
			if len(m.profiles) == 0 {
				m.setMessage(levelInfo, "No profiles yet. Press n to create one, or i to import.")
				return m, nil
			}
			if p, ok := m.currentPickerProfile(); ok {
				m.openEditModal(p)
			}
			return m, nil
		}
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			return m.startTreeMatrix(matrixMembership)
		}
		if m.screen == ScreenWorkspace && m.rowDataTab() {
			return m.startRowEdit()
		}
	case keymap.Add:
		if m.screen == ScreenWorkspace {
			if m.focus == FocusTree {
				return m.startRoleAdd()
			}
			if m.focus == FocusMain {
				return m.startRowAdd()
			}
			m.setMessage(levelInfo, "row changes are available in the Data tab")
		}
	case keymap.AddUser:
		if m.screen != ScreenWorkspace || m.focus != FocusTree {
			return m, nil
		}
		node, ok := m.tree.selectedNode()
		if !ok {
			return m, nil
		}
		switch node.Kind {
		case domain.KindSecurity, domain.KindUsers:
			return m.startUserAdd()
		case domain.KindDatabase, domain.KindSchema, domain.KindCollection, domain.KindTable:
			return m.startTreeMatrix(matrixMembership)
		default:
			m.setMessage(levelInfo, "select Security/users or a database/schema/table/collection")
		}
	case keymap.HomeDatabase:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			return m.startUserHome()
		}
	case keymap.MatrixPrivileges:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			return m.startTreeMatrix(matrixPrivileges)
		}
	case keymap.TreeMode:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			return m.toggleMode()
		}
	case keymap.Remove:
		if m.screen == ScreenPicker && !m.connecting {
			return m.openDeleteProfileModal()
		}
		if m.screen == ScreenWorkspace {
			if m.focus == FocusTree {
				return m.startRoleRemove()
			}
			if m.focus == FocusMain {
				return m.startRowRemove()
			}
			m.setMessage(levelInfo, "row changes are available in the Data tab")
		}
	case keymap.DeleteProfile:
		if m.screen == ScreenPicker && !m.connecting {
			return m.openDeleteProfileModal()
		}
	case keymap.Help:
		m.setMessage(levelInfo, "help — story 9")
	case keymap.Tab:
		if m.screen == ScreenWorkspace {
			m.completion = completionState{}
			if m.focus == FocusTree {
				m.clearTreeSearch()
			}
			m.focus = (m.focus + 1) % 3
			m.syncQueryFocus()
			m.syncQueryPlaceholder()
			if m.focus == FocusMain {
				return m, m.ensureData()
			}
		}
	case keymap.ShiftTab:
		if m.screen == ScreenWorkspace {
			m.completion = completionState{}
			if m.focus == FocusTree {
				m.clearTreeSearch()
			}
			m.focus = (m.focus + 2) % 3
			m.syncQueryFocus()
			m.syncQueryPlaceholder()
			if m.focus == FocusMain {
				return m, m.ensureData()
			}
		}
	case keymap.Up:
		switch {
		case m.screen == ScreenPicker:
			m.pickerMove(-1)
		case m.focus == FocusTree:
			m.tree.move(-1)
			return m, m.syncDetails()
		case m.focus == FocusMain:
			if m.dataTabActive() {
				return m, m.moveDataRow(-1)
			}
			m.scrollDetails(-1)
		}
	case keymap.Down:
		switch {
		case m.screen == ScreenPicker:
			m.pickerMove(1)
		case m.focus == FocusTree:
			m.tree.move(1)
			return m, m.syncDetails()
		case m.focus == FocusMain:
			if m.dataTabActive() {
				return m, m.moveDataRow(1)
			}
			m.scrollDetails(1)
		}
	case keymap.PageUp:
		switch {
		case m.screen == ScreenPicker:
			m.pickerMove(-8)
		case m.focus == FocusTree:
			m.tree.move(-m.treePage())
			return m, m.syncDetails()
		case m.focus == FocusMain:
			if m.dataTabActive() {
				return m, m.moveDataPage(-1)
			}
			m.scrollDetails(-m.detailsPage())
		}
	case keymap.PageDown:
		switch {
		case m.screen == ScreenPicker:
			m.pickerMove(8)
		case m.focus == FocusTree:
			m.tree.move(m.treePage())
			return m, m.syncDetails()
		case m.focus == FocusMain:
			if m.dataTabActive() {
				return m, m.moveDataPage(1)
			}
			m.scrollDetails(m.detailsPage())
		}
	case keymap.Home:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				m.tree.selectFirst()
				return m, m.syncDetails()
			case FocusMain:
				if m.dataTabActive() {
					m.moveDataColumnEdge(false)
				}
			}
		}
	case keymap.End:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				m.tree.selectLast()
				return m, m.syncDetails()
			case FocusMain:
				if m.dataTabActive() {
					m.moveDataColumnEdge(true)
				}
			}
		}
	case keymap.RangeStart:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				m.tree.selectFirst()
				return m, m.syncDetails()
			case FocusMain:
				if m.dataTabActive() {
					return m, m.dataRangeStart()
				}
			}
		}
	case keymap.RangeEnd:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				m.tree.selectLast()
				return m, m.syncDetails()
			case FocusMain:
				if m.dataTabActive() {
					return m, m.dataRangeEnd()
				}
			}
		}
	case keymap.Left:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				return m, m.collapseSelected()
			case FocusMain:
				if m.dataTabActive() {
					m.moveDataColumn(-1)
				}
			}
		}
	case keymap.Right:
		if m.screen == ScreenWorkspace {
			switch m.focus {
			case FocusTree:
				return m.expandSelected()
			case FocusMain:
				if m.dataTabActive() {
					m.moveDataColumn(1)
				}
			}
		}
	case keymap.Tab1:
		if m.screen == ScreenWorkspace {
			return m, m.selectTab(0)
		}
	case keymap.Tab2:
		if m.screen == ScreenWorkspace {
			return m, m.selectTab(1)
		}
	case keymap.Tab3:
		if m.screen == ScreenWorkspace {
			return m, m.selectTab(2)
		}
	case keymap.Tab4:
		if m.screen == ScreenWorkspace {
			return m, m.selectTab(3)
		}
	case keymap.PrevTab:
		if m.screen == ScreenWorkspace {
			return m, m.cycleTab(-1)
		}
	case keymap.NextTab:
		if m.screen == ScreenWorkspace {
			return m, m.cycleTab(1)
		}
	case keymap.Disconnect:
		if m.screen == ScreenWorkspace {
			m.disconnect()
		}
	case keymap.Cancel:
		if m.screen == ScreenWorkspace && m.focus == FocusTree && m.tree.searchPattern != "" {
			m.clearTreeSearch()
			return m, nil
		}
		if m.queryRunning {
			return m.cancelResults()
		}
		if m.details.exactLoading {
			m.cancelExactCount()
			return m, nil
		}
		if m.data.loading {
			if m.resultsActive {
				return m.cancelResults()
			}
			m.cancelDataFetch()
			m.setMessage(levelInfo, "cancelled")
			return m, nil
		}
		if m.data.bufferPattern != "" {
			m.clearLocalFilter()
			return m, nil
		}
		if m.data.serverFilter != nil {
			return m, m.clearServerFilter()
		}
		if m.data.selectedCount() > 0 {
			m.clearDataSelection()
			m.setMessage(levelInfo, "selection cleared")
			return m, nil
		}
		if m.resultsActive {
			m.leaveResults()
			return m, nil
		}
		if m.connecting {
			m.cancelConnect()
		}
	case keymap.Search:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			m.openTreeSearch()
			return m, nil
		}
		return m, m.openBufferFilter()
	case keymap.InsertSelected:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			if !m.tree.searchPrompt {
				m.openTreeSearch()
			}
			m.treeSearchInsertValue(true)
			return m, nil
		}
	case keymap.ServerFilter:
		return m, m.openServerFilter()
	case keymap.More:
		if m.dataFocused() {
			return m, m.nextDataPage()
		}
		m.setMessage(levelInfo, "m fetches the next page in the Data tab")
	case keymap.FetchAll:
		if m.dataFocused() {
			return m, m.fetchAllData()
		}
		m.setMessage(levelInfo, "Ctrl+A fetches rows in the Data tab")
	case keymap.ExpandJSON:
		if m.dataFocused() {
			m.toggleJSONColumns()
		}
	case keymap.Execute:
		return m.startQuery()
	case keymap.Edit:
		if m.screen == ScreenWorkspace {
			m.openStatementModal()
		}
	case keymap.ContextHelp:
		if m.screen == ScreenWorkspace {
			m.openContextHelp()
		}
	case keymap.Space:
		if m.screen == ScreenWorkspace && m.focus == FocusTree {
			return m.openSelected()
		}
		if m.dataFocused() {
			return m, m.toggleDataSelection()
		}
	case keymap.ExactCount:
		return m, m.startExactCount()
	case keymap.CaseMode:
		return m, m.toggleCaseMode()
	case keymap.NextMatch:
		if m.screen == ScreenWorkspace && m.focus == FocusTree &&
			m.tree.searchPattern != "" && !m.tree.searchPrompt {
			return m, m.nextTreeMatch(-1)
		}
		return m, m.nextMatch(-1)
	case keymap.Matrix:
		if m.screen == ScreenWorkspace {
			return m.openMatrixWithDB(matrixMembership, nil, m.matrixEnsureDatabase())
		}
	case keymap.ToggleMode:
		if m.screen != ScreenWorkspace {
			return m, nil
		}
		return m.toggleMode()
	case keymap.ResizeTreeLeft:
		m.resizeTree(-2)
	case keymap.ResizeTreeRight:
		m.resizeTree(2)
	case keymap.ResizeQueryUp:
		m.resizeQuery(1)
	case keymap.ResizeQueryDown:
		m.resizeQuery(-1)
	case keymap.Completion:
		m.setMessage(levelInfo, "not available in this story")
	}
	return m, nil
}

func (m *Model) setMessage(level messageLevel, text string) {
	m.message = text
	m.messageLevel = level
}

func (m Model) messageStyle() lipgloss.Style {
	switch m.messageLevel {
	case levelSuccess:
		return m.styles.Success
	case levelWarning:
		return m.styles.Warning
	case levelError:
		return m.styles.Error
	case levelInfo:
		return m.styles.Info
	default:
		return m.styles.Muted
	}
}

func (m Model) enterPicker() (tea.Model, tea.Cmd) {
	if m.connecting {
		return m, nil
	}
	if len(m.profiles) == 0 {
		m.setMessage(levelInfo, "No profiles yet. Press n to create one, or i to import.")
		return m, nil
	}
	if m.pickerSel < 0 {
		m.pickerSel = 0
	}
	if m.pickerSel >= len(m.profiles) {
		m.pickerSel = len(m.profiles) - 1
	}
	p := m.profiles[m.pickerSel].Profile
	delete(m.rowErrors, p.Key())
	if p.Source == nil || p.Source.Kind == domain.SourcePrompt {
		m.openPasswordModal(p)
		return m, nil
	}
	return m.beginConnect(originProfile, p, app.Credential{})
}

func (m Model) beginConnect(origin connectOrigin, p domain.Profile, cred app.Credential) (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}
	m.connectGen++
	gen := m.connectGen
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.connecting = true
	m.connectingKey = p.Key()
	m.connectingLabel = p.Label
	m.setMessage(levelInfo, "connecting to "+p.Label+"…")
	if origin == originNew {
		m.newBusy = true
	}
	if origin == originPassword {
		m.passwordBusy = true
	}
	svc := m.svc
	return m, func() tea.Msg {
		result, err := svc.Connect(ctx, p, cred)
		return connectDoneMsg{gen: gen, origin: origin, profile: p, result: result, err: err}
	}
}

func (m Model) handleConnectDone(msg connectDoneMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.connectGen {
		if msg.result.Session != nil {
			_ = msg.result.Session.Close()
		}
		return m, nil
	}
	m.connecting = false
	m.connectingKey = ""
	m.cancel = nil
	switch msg.origin {
	case originProfile:
		if msg.err != nil {
			m.rowErrors[msg.profile.Key()] = oneLine(msg.err.Error())
			m.setMessage(levelError, oneLine(msg.err.Error()))
			return m, nil
		}
		next := m.openWorkspace(msg.result)
		return next.loadRoot()
	case originPassword:
		m.passwordBusy = false
		if msg.err != nil {
			m.closePasswordModal()
			m.rowErrors[msg.profile.Key()] = oneLine(msg.err.Error())
			m.setMessage(levelError, oneLine(msg.err.Error()))
			return m, nil
		}
		m.closePasswordModal()
		next := m.openWorkspace(msg.result)
		return next.loadRoot()
	case originNew:
		m.newBusy = false
		if msg.err != nil {
			if errors.Is(msg.err, app.ErrPasswordRequired) {
				m.newNeedsPassword = true
				m.newFocus = fieldPassword
				m.newErr = ""
				m.focusNewField()
				return m, nil
			}
			if domain.KindOf(msg.err) == domain.KindAuth && msg.profile.Source != nil {
				// The source password was rejected. Let the user supply a
				// one-time password instead of dead-ending the form.
				m.newNeedsPassword = true
				m.newIgnoreSource = true
				m.newFocus = fieldPassword
				m.newErr = oneLine(msg.err.Error())
				m.focusNewField()
				return m, nil
			}
			m.newErr = oneLine(msg.err.Error())
			m.newPassword.SetValue("")
			return m, nil
		}
		m.closeNewModal()
		m.saveProfile = msg.result.Profile
		m.saveSession = msg.result.Session
		m.saveResult = msg.result
		m.modal = modalSave
		m.reloadProfiles(context.Background(), msg.result.Profile.Key())
		m.setMessage(levelSuccess, "connected")
		return m, nil
	}
	return m, nil
}

func (m Model) openWorkspace(result app.ConnectResult) Model {
	if m.session != nil && m.session != result.Session {
		_ = m.session.Close()
	}
	if m.sessionCancel != nil {
		m.sessionCancel()
	}
	m.session = result.Session
	m.sessionCtx, m.sessionCancel = context.WithCancel(context.Background())
	m.active = result.Profile
	m.screen = ScreenWorkspace
	m.focus = FocusTree
	m.caseInsensitive = true
	m.layoutKey = result.Profile.Key()
	savedLayout := m.svc.Layout(context.Background(), m.layoutKey)
	m.treeWidth = savedLayout.TreeWidth
	if m.treeWidth <= 0 {
		m.treeWidth = layout.TreeWidth
	}
	m.queryHeight = savedLayout.QueryHeight
	if m.queryHeight <= 0 {
		m.queryHeight = layout.QueryLines
	}
	m.history = m.svc.History(context.Background(), result.Profile)
	m.historyIndex = len(m.history)
	m.historyDraft = ""
	m.queryGen++
	m.resultsGen++
	m.query.SetValue("")
	m.query.SetCursor(0)
	m.query.Placeholder = queryPlaceholder(result.Profile.Engine, domain.ObjectRef{
		Engine: result.Profile.Engine, Database: result.Profile.Database,
	})
	m.query.Blur()
	m.resultsActive = false
	m.queryRunning = false
	m.modeBusy = false
	m.modeGen++
	m.roleGen++
	m.escChord = false
	m.completion = completionState{}
	warnings := []string{}
	if result.Warning != "" {
		warnings = append(warnings, result.Warning)
	}
	if result.Elevated {
		warnings = append(warnings, "This account has elevated rights. Read-only mode is a product guard, not a security boundary.")
	}
	if result.Profile.Engine == domain.EngineMongoDB {
		warnings = append(warnings, "Read-only is enforced by the tool; MongoDB has no session-level read-only.")
	}
	if m.rootLaunch {
		warnings = append(warnings, "The tool is running as root. Read-only mode is a product guard, not a security boundary.")
	}
	m.elevated = result.Elevated
	m.tree.reset()
	m.details.clear()
	m.detailsGen++
	m.exactGen++
	m.restore = nil
	m.resetData()
	m.message = "connected"
	m.messageLevel = levelSuccess
	if len(warnings) > 0 {
		m.setMessage(levelWarning, oneLine(strings.Join(warnings, " ")))
	}
	m.reloadProfiles(context.Background(), result.Profile.Key())
	return m
}

func (m *Model) disconnect() {
	m.savePlace()
	if m.sessionCancel != nil {
		m.sessionCancel()
		m.sessionCancel = nil
	}
	if m.session != nil {
		_ = m.session.Close()
		m.session = nil
	}
	key := m.active.Key()
	m.treeGen++
	m.screen = ScreenPicker
	m.focus = FocusTree
	m.queryGen++
	m.resultsGen++
	m.query.Blur()
	m.resultsActive = false
	m.queryRunning = false
	m.modeBusy = false
	m.modeGen++
	m.roleGen++
	m.elevated = false
	m.escChord = false
	m.completion = completionState{}
	m.layoutKey = ""
	m.treeWidth = layout.TreeWidth
	m.queryHeight = layout.QueryLines
	m.history = nil
	m.historyIndex = 0
	m.tree.reset()
	m.details.clear()
	m.detailsGen++
	m.exactGen++
	m.restore = nil
	m.resetData()
	m.reloadProfiles(context.Background(), key)
	m.setMessage(levelInfo, "disconnected")
}

func (m *Model) cancelConnect() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.connectGen++
	m.connecting = false
	m.connectingKey = ""
	m.connectingLabel = ""
	m.newBusy = false
	m.passwordBusy = false
	m.setMessage(levelInfo, "connection cancelled")
}

func (m *Model) reloadProfiles(ctx context.Context, selectKey string) {
	views, err := m.svc.Profiles(ctx)
	if err != nil {
		m.profiles = nil
		m.storageErr = true
		m.pickerSel = 0
		m.pickerOffset = 0
		return
	}
	m.storageErr = false
	m.profiles = views
	if selectKey != "" {
		for i, view := range views {
			if view.Profile.Key() == selectKey {
				m.pickerSel = i
				break
			}
		}
	}
	if m.pickerSel >= len(m.profiles) {
		m.pickerSel = len(m.profiles) - 1
	}
	if m.pickerSel < 0 {
		m.pickerSel = 0
	}
	m.clampPickerOffset()
}

func (m *Model) pickerMove(delta int) {
	if len(m.profiles) == 0 {
		return
	}
	m.pickerSel += delta
	if m.pickerSel < 0 {
		m.pickerSel = 0
	}
	if m.pickerSel >= len(m.profiles) {
		m.pickerSel = len(m.profiles) - 1
	}
	m.clampPickerOffset()
}

func (m *Model) clampPickerOffset() {
	if len(m.profiles) == 0 {
		m.pickerOffset = 0
		return
	}
	if m.pickerSel < m.pickerOffset {
		m.pickerOffset = m.pickerSel
	}
	limit := m.pickerVisibleRows()
	if limit < 1 {
		limit = 1
	}
	for m.pickerOffset < m.pickerSel {
		total := 0
		for i := m.pickerOffset; i <= m.pickerSel && i < len(m.profiles); i++ {
			total += m.pickerRowHeight(i)
		}
		if total <= limit {
			break
		}
		m.pickerOffset++
	}
	if m.pickerOffset < 0 {
		m.pickerOffset = 0
	}
	if m.pickerOffset >= len(m.profiles) {
		m.pickerOffset = len(m.profiles) - 1
	}
}

func (m Model) pickerRowHeight(index int) int {
	if index < 0 || index >= len(m.profiles) {
		return 0
	}
	height := 1
	if m.rowErrors[m.profiles[index].Profile.Key()] != "" {
		height++
	}
	return height
}

func (m Model) pickerVisibleRows() int {
	// Title, blank, header, footer, and the status line when one is present.
	rows := m.height - 4
	if m.connecting || m.message != "" {
		rows--
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m Model) currentPickerProfile() (domain.Profile, bool) {
	if m.pickerSel < 0 || m.pickerSel >= len(m.profiles) {
		return domain.Profile{}, false
	}
	return m.profiles[m.pickerSel].Profile, true
}

func (m *Model) resizeInputs() {
	newWidth := modalWidth(m.width, 60) - 17
	if newWidth < 10 {
		newWidth = 10
	}
	passwordWidth := modalWidth(m.width, 60) - 17
	if passwordWidth < 10 {
		passwordWidth = 10
	}
	m.newLabel.SetWidth(newWidth)
	m.newHost.SetWidth(newWidth)
	m.newPort.SetWidth(newWidth)
	m.newDatabase.SetWidth(newWidth)
	m.newUser.SetWidth(newWidth)
	m.newAuth.SetWidth(newWidth)
	m.newPassword.SetWidth(newWidth)
	m.passwordInput.SetWidth(passwordWidth)
	queryWidth := m.width - 12
	if queryWidth < 10 {
		queryWidth = 10
	}
	m.query.SetWidth(queryWidth)
	statementWidth := modalWidth(m.width, 70) - 10
	if statementWidth < 10 {
		statementWidth = 10
	}
	m.statementInput.SetWidth(statementWidth)
	m.user.name.SetWidth(newWidth)
	m.user.host.SetWidth(newWidth)
	m.user.database.SetWidth(newWidth)
	m.user.hosts.SetWidth(newWidth)
	m.user.password.SetWidth(newWidth)
	m.editLabel.SetWidth(newWidth)
	m.editHost.SetWidth(newWidth)
	m.editPort.SetWidth(newWidth)
	m.editDatabase.SetWidth(newWidth)
	m.editUser.SetWidth(newWidth)
	m.editAuth.SetWidth(newWidth)
}

func (m Model) regions() layout.Regions {
	return layout.ComputeSize(m.width, m.height, m.treeWidth, m.queryHeight)
}

func (m Model) View() tea.View {
	regions := m.regions()
	var content string
	switch {
	case regions.TooSmall:
		content = m.viewTooSmall()
	default:
		base := m.viewPicker(regions)
		if m.screen == ScreenWorkspace {
			base = m.viewWorkspace(regions)
		}
		if m.screen == ScreenMatrix {
			base = m.viewMatrix()
		}
		if m.modal != modalNone {
			content = overlayCenter(base, m.viewModal(), m.width, m.height)
		} else {
			content = base
		}
		if m.completion.active && (m.modal == modalNone || m.modal == modalStatement) {
			if popup := m.completionPopup(); popup != "" {
				lines := strings.Split(popup, "\n")
				popupWidth := 0
				for _, line := range lines {
					if w := lipgloss.Width(line); w > popupWidth {
						popupWidth = w
					}
				}
				x, y := m.completionAnchor(popupWidth, len(lines))
				content = overlayAt(content, popup, m.width, m.height, x, y)
			}
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = lipgloss.Color(theme.Nightshift.Base)
	return v
}

func (m Model) viewTooSmall() string {
	line := "Terminal too small: need 80×24. Current " + strconv.Itoa(m.width) + "×" + strconv.Itoa(m.height) + "."
	return Pad(m.styles.Error.Render(Truncate(line, m.width)), m.width)
}

func (m Model) viewModal() string {
	switch m.modal {
	case modalImport:
		return m.viewImport()
	case modalNew:
		return m.viewNewConn()
	case modalPassword:
		return m.viewPassword()
	case modalSave:
		return m.viewSave()
	case modalEdit:
		return m.viewEdit()
	case modalDeleteProfile:
		return m.viewDeleteProfile()
	case modalStatement:
		return m.viewStatement()
	case modalHelp:
		return m.viewHelp()
	case modalMode:
		return m.viewModeConfirm()
	case modalRoleMenu:
		return m.viewRoleMenu()
	case modalRoleCreate:
		return m.viewRoleCreate()
	case modalRoleConfirm:
		return m.viewRoleConfirm()
	case modalMatrixDetails:
		return m.viewMatrixDetails()
	case modalUserForm:
		return m.viewUserForm()
	case modalUserConfirm:
		return m.viewUserConfirm()
	case modalUserSave:
		return m.viewUserSave()
	case modalUserHome:
		return m.viewUserHome()
	case modalRowInsert:
		return m.viewRowInsert()
	case modalRowEdit:
		return m.viewRowEdit()
	case modalRowDelete:
		return m.viewRowDelete()
	}
	return ""
}

// dismissModal closes the current modal without applying it. A pending
// save-as-profile prompt is treated as skipped so its live session continues.
func (m *Model) dismissModal() {
	switch m.modal {
	case modalSave:
		m.finishSave(false)
		return
	case modalNew:
		m.closeNewModal()
	case modalPassword:
		m.closePasswordModal()
	case modalImport:
		m.modal = modalNone
	case modalEdit:
		m.closeEditModal()
	case modalDeleteProfile:
		m.closeDeleteProfileModal()
	case modalStatement:
		m.closeStatementModal()
	case modalHelp:
		m.modal = modalNone
		m.helpReturn = modalNone
		m.helpTerm = ""
		m.helpBody = ""
	case modalMode:
		m.modal = modalNone
	case modalRoleMenu, modalRoleCreate, modalRoleConfirm:
		m.modal = modalNone
	case modalMatrixDetails:
		m.modal = modalNone
		m.matrix.details = matrixDetails{}
	case modalUserForm, modalUserConfirm, modalUserSave:
		m.modal = modalNone
		m.user = newUserState()
	case modalUserHome:
		m.closeUserHome()
	case modalRowInsert, modalRowEdit, modalRowDelete:
		m.modal = modalNone
	}
}

// Used by tests and views to keep external strings on one rendered line.
func oneLine(s string) string {
	s = Sanitize(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.TrimSpace(s)
}
