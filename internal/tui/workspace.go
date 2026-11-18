package tui

import (
	"strconv"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

func (m Model) viewWorkspace(regions layout.Regions) string {
	top := m.viewTopStatus(regions.Width)
	tree := m.viewTree(regions.BodyHeight, regions.TreeWidth)
	main := m.viewMain(regions.BodyHeight, regions.MainWidth)
	body := strings.Join(
		JoinColumns(strings.Split(tree, "\n"), strings.Split(main, "\n"), regions.Gap),
		"\n",
	)
	query := m.viewQuery(regions.Width)
	bottom := m.viewBottom(regions.Width)
	return strings.Join([]string{top, body, query, bottom}, "\n")
}

func (m Model) viewTopStatus(width int) string {
	p := m.profileForStatus()
	database := p.Database
	if database == "" {
		database = "—"
	}
	left := m.styles.Text.Render(oneLine(p.Label)) +
		m.styles.Muted.Render(" · "+p.Engine.DisplayName()+" · "+oneLine(database)+" · ") +
		m.styles.ModeRO.Render("r/o")
	right := m.styles.Muted.Render(oneLine(p.User))
	return ComposeLR(left, right, width)
}

func (m Model) viewTree(height, width int) string {
	inner := width - 4
	visible := m.tree.visibleRows(height - 2)
	var lines []string
	switch {
	case m.tree.loadingRoot:
		lines = append(lines, m.styles.Muted.Render("loading…"))
	case m.tree.rootErr != "":
		lines = append(lines, m.styles.Error.Render(Truncate("error "+m.tree.rootErr, inner)))
	case len(m.tree.rows) == 0:
		lines = append(lines, m.styles.Muted.Render("no databases visible"))
	default:
		selectedKey := m.tree.selectedKey()
		for _, row := range visible {
			line := treeRowText(row)
			switch {
			case row.Info != "" && row.Error:
				lines = append(lines, m.styles.Error.Render(Truncate(line, inner)))
			case row.Info != "":
				lines = append(lines, m.styles.Muted.Render(Truncate(line, inner)))
			case row.Key == selectedKey:
				lines = append(lines, m.styles.Select.Render(Pad(line, inner)))
			case row.Node.Kind == domain.KindSecurity || row.Node.Kind == domain.KindUsers ||
				row.Node.Kind == domain.KindRoles || row.Node.Kind == domain.KindUser ||
				row.Node.Kind == domain.KindRole:
				lines = append(lines, m.styles.Info.Render(Truncate(line, inner)))
			default:
				lines = append(lines, m.styles.Text.Render(Truncate(line, inner)))
			}
		}
	}
	title := "objects " + strconv.Itoa(m.tree.objectCount())
	return Box(title, strings.Join(lines, "\n"), width, height, m.styles, m.focus == FocusTree)
}

func (m Model) viewMain(height, width int) string {
	if !m.details.has || len(m.details.tabs) == 0 {
		body := m.styles.Muted.Render("Select a table or collection to browse.")
		return Box("", body, width, height, m.styles, m.focus == FocusMain)
	}
	parts := make([]string, 0, len(m.details.tabs))
	for i, label := range m.details.tabs {
		numbered := strconv.Itoa(i+1) + " " + label
		if i == m.details.tab {
			parts = append(parts, m.styles.Accent.Bold(true).Underline(true).Render(numbered))
		} else {
			parts = append(parts, m.styles.Label.Render(numbered))
		}
	}
	lines := []string{strings.Join(parts, m.styles.Muted.Render(" · ")), ""}
	innerHeight := height - 2
	bodyHeight := innerHeight - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	switch m.details.currentTab() {
	case "Data":
		lines = append(lines, m.viewData(width-4, bodyHeight)...)
	case "Privileges":
		lines = append(lines, m.styles.Muted.Render("privilege inspection lands in story 7"))
	default:
		detail := detailLines(m.details.node, m.currentDatabase(), m.tree.childCount(m.details.node.Ref.Key()))
		offset := m.details.offset
		if offset > len(detail)-bodyHeight {
			offset = len(detail) - bodyHeight
		}
		if offset < 0 {
			offset = 0
		}
		end := offset + bodyHeight
		if end > len(detail) {
			end = len(detail)
		}
		hasPosition := len(detail) > bodyHeight
		renderHeight := bodyHeight
		if hasPosition {
			renderHeight--
			if renderHeight < 1 {
				renderHeight = 1
			}
			end = offset + renderHeight
			if end > len(detail) {
				end = len(detail)
			}
		}
		for _, line := range detail[offset:end] {
			lines = append(lines, m.renderDetailLine(line))
		}
		if hasPosition {
			lines = append(lines, m.styles.Muted.Render(detailPosition(offset, end-offset, len(detail))))
		}
	}
	return Box("", strings.Join(lines, "\n"), width, height, m.styles, m.focus == FocusMain)
}

func (m Model) renderDetailLine(line detailLine) string {
	switch {
	case line.Header:
		return m.styles.Header.Render(line.Value)
	case line.Label == "":
		value := oneLine(line.Value)
		if line.Mutable {
			return m.styles.Muted.Render(value)
		}
		return m.styles.Text.Render(value)
	default:
		return m.styles.Muted.Render(field(line.Label, 12)) + " " + m.styles.Text.Render(oneLine(line.Value))
	}
}

func (m Model) viewQuery(width int) string {
	label := "SQL"
	statement := "SELECT * FROM public.customers WHERE status = 'active' LIMIT 100;"
	switch m.profileForStatus().Engine {
	case domain.EngineMongoDB:
		label = "MQL"
		statement = "db.customers.find({ status: 'active' }).limit(100);"
	case domain.EngineMySQL:
		statement = "SELECT * FROM customers WHERE status = 'active' LIMIT 100;"
	}
	marker := "  "
	promptStyle := m.styles.ModeRO
	statementStyle := m.styles.Text
	if m.focus == FocusQuery {
		marker = m.styles.Focused.Render("▸ ")
		promptStyle = m.styles.Focused
		statementStyle = m.styles.Focused
	}
	prompt := promptStyle.Render(label) + " " + m.styles.Accent.Render("›")
	return Pad(marker+prompt+" "+statementStyle.Render(statement), width)
}

func (m Model) viewBottom(width int) string {
	left := "connected · " + oneLine(m.currentDatabase())
	style := m.styles.Muted
	if m.tree.loadingRoot {
		left = "loading catalog…"
		style = m.styles.Info
	}
	if m.message != "" {
		left = m.message
		style = m.messageStyle()
	}
	if m.pendingChord {
		left = "Ctrl+X …"
		style = m.styles.Accent
	}
	right := "Tab focus"
	switch m.focus {
	case FocusTree:
		right = "Enter open · Right expand"
	case FocusMain:
		right = "1/2/3 tab · Up/Down scroll"
	case FocusQuery:
		right = "query bar · Ctrl+X chord"
	}
	if m.screen == ScreenWorkspace && m.dataTabActive() {
		if m.message == "" {
			if summary := m.dataSummary(); summary != "" {
				left = summary
				style = m.styles.Muted
				if m.data.loading {
					style = m.styles.Info
				}
			}
		}
		switch m.focus {
		case FocusMain:
			right = "↑↓ rows · m more"
			if m.profileForStatus().Engine == domain.EngineMongoDB {
				right += " · ` JSON"
			}
		case FocusQuery:
			right = "query bar · Ctrl+X chord"
		default:
			right = "Enter open"
		}
	}
	return ComposeLR(style.Render(oneLine(left)), m.styles.Muted.Render(right), width)
}
