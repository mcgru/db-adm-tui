package tui

import (
	"fmt"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

type Screen int

const (
	ScreenPicker Screen = iota
	ScreenWorkspace
)

type Focus int

const (
	FocusTree Focus = iota
	FocusMain
	FocusQuery
)

type Profile struct {
	Label    string
	Engine   string
	Host     string
	User     string
	Source   string
	LastUsed string
}

type chordTimeoutMsg struct {
	seq int
}

type Model struct {
	width, height int
	screen        Screen
	focus         Focus
	pickerSel     int
	activeProfile int
	tab           int
	treeSel       int
	pendingChord  bool
	chordSeq      int
	message       string
	styles        theme.Styles
	profiles      []Profile
}

func New() Model {
	return Model{
		width:   layout.MinWidth,
		height:  layout.MinHeight,
		screen:  ScreenPicker,
		focus:   FocusTree,
		tab:     0,
		treeSel: 2,
		styles:  theme.New(),
		profiles: []Profile{
			{Label: "prod-ro", Engine: "PostgreSQL", Host: "db1.internal", User: "app_ro", Source: "~/.pgpass", LastUsed: "2m ago"},
			{Label: "analytics", Engine: "MongoDB", Host: "mongo1:27017", User: "analyst", Source: "~/.mongo.analyst", LastUsed: "yesterday"},
			{Label: "billing", Engine: "MySQL", Host: "mysql1:3306", User: "billing_ro", Source: "manual", LastUsed: "3d ago"},
		},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case chordTimeoutMsg:
		if m.pendingChord && msg.seq == m.chordSeq {
			m.pendingChord = false
			m.message = "cancelled"
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	m.message = ""
	if m.pendingChord {
		m.pendingChord = false
		action, ok := keymap.LookupChord(key)
		if !ok {
			m.message = "cancelled"
			return m, nil
		}
		return m.apply(action)
	}

	action, ok := keymap.Lookup(key)
	if !ok {
		return m, nil
	}
	if layout.Compute(m.width, m.height).TooSmall &&
		action != keymap.ForceQuit && action != keymap.Quit {
		return m, nil
	}
	if action == keymap.ChordX {
		m.chordSeq++
		seq := m.chordSeq
		m.pendingChord = true
		m.message = "Ctrl+X …"
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return chordTimeoutMsg{seq: seq}
		})
	}
	return m.apply(action)
}

func (m Model) apply(action keymap.Action) (tea.Model, tea.Cmd) {
	switch action {
	case keymap.ForceQuit:
		return m, tea.Quit
	case keymap.Quit:
		if m.screen == ScreenPicker {
			return m, tea.Quit
		}
		m.message = "q quits only at the picker"
	case keymap.Confirm:
		if m.screen == ScreenPicker {
			m.activeProfile = m.pickerSel
			m.screen = ScreenWorkspace
			m.focus = FocusTree
			m.tab = 0
			m.treeSel = 2
			m.message = "connected (fake)"
		}
	case keymap.New:
		if m.screen == ScreenPicker {
			m.message = "new connection — story 2"
		}
	case keymap.Import:
		if m.screen == ScreenPicker {
			m.message = "import — story 2"
		}
	case keymap.Help:
		m.message = "help — story 9"
	case keymap.Tab:
		if m.screen == ScreenWorkspace {
			m.focus = (m.focus + 1) % 3
		}
	case keymap.ShiftTab:
		if m.screen == ScreenWorkspace {
			m.focus = (m.focus + 2) % 3
		}
	case keymap.Up:
		switch {
		case m.screen == ScreenPicker && m.pickerSel > 0:
			m.pickerSel--
		case m.screen == ScreenWorkspace && m.focus == FocusTree && m.treeSel > 0:
			m.treeSel--
		}
	case keymap.Down:
		switch {
		case m.screen == ScreenPicker && m.pickerSel < len(m.profiles)-1:
			m.pickerSel++
		case m.screen == ScreenWorkspace && m.focus == FocusTree && m.treeSel < len(fakeTree)-1:
			m.treeSel++
		}
	case keymap.Left, keymap.Right:
		// Tree expansion lands with the catalog story.
	case keymap.Tab1:
		if m.screen == ScreenWorkspace {
			m.tab = 0
		}
	case keymap.Tab2:
		if m.screen == ScreenWorkspace {
			m.tab = 1
		}
	case keymap.Tab3:
		if m.screen == ScreenWorkspace {
			m.tab = 2
		}
	case keymap.PrevTab:
		if m.screen == ScreenWorkspace {
			m.tab = (m.tab + 2) % 3
		}
	case keymap.NextTab:
		if m.screen == ScreenWorkspace {
			m.tab = (m.tab + 1) % 3
		}
	case keymap.Disconnect:
		if m.screen == ScreenWorkspace {
			m.screen = ScreenPicker
			m.pickerSel = m.activeProfile
			m.focus = FocusTree
			m.tab = 0
			m.treeSel = 2
			m.message = "disconnected"
		}
	case keymap.Search:
		m.message = "search — story 4"
	case keymap.More:
		m.message = "more — story 4"
	case keymap.Execute:
		m.message = "execute — story 5"
	case keymap.ExactCount, keymap.ToggleMode, keymap.Edit, keymap.CaseMode,
		keymap.Completion, keymap.ContextHelp, keymap.ServerFilter, keymap.FetchAll,
		keymap.Space, keymap.NextMatch, keymap.Home, keymap.End,
		keymap.PageUp, keymap.PageDown, keymap.ResizeTreeLeft, keymap.ResizeTreeRight,
		keymap.ResizeQueryUp, keymap.ResizeQueryDown:
		m.message = "not available in this story"
	case keymap.Cancel:
		m.message = ""
	}
	return m, nil
}

func (m Model) View() tea.View {
	regions := layout.Compute(m.width, m.height)
	var content string
	switch {
	case regions.TooSmall:
		content = m.viewTooSmall()
	case m.screen == ScreenPicker:
		content = m.viewPicker(regions)
	default:
		content = m.viewWorkspace(regions)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = lipgloss.Color(theme.Nightshift.Base)
	return v
}

func (m Model) viewTooSmall() string {
	line := fmt.Sprintf("Terminal too small: need 80×24. Current %d×%d.", m.width, m.height)
	return Pad(m.styles.Error.Render(Truncate(line, m.width)), m.width)
}

func (m Model) profile() Profile {
	if len(m.profiles) == 0 {
		return Profile{Label: "prod-ro", Engine: "PostgreSQL", User: "app_ro"}
	}
	if m.activeProfile < 0 || m.activeProfile >= len(m.profiles) {
		return m.profiles[0]
	}
	return m.profiles[m.activeProfile]
}
