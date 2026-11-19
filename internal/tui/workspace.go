package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

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
	caseText := "case: insensitive"
	if !m.caseInsensitive {
		caseText = "case: sensitive"
	}
	left := m.styles.Text.Render(oneLine(p.Label)) +
		m.styles.Muted.Render(" · "+p.Engine.DisplayName()+" · "+oneLine(database)+" · ") +
		m.modeBadge()
	right := m.styles.Muted.Render(caseText + " · " + oneLine(p.User))
	if m.elevated {
		right += m.styles.Warning.Render(" · privileged")
	}
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
				label := treeRowLabel(row)
				access := treeRowAccess(row)
				if access == "" {
					lines = append(lines, m.styles.Text.Render(Truncate(label, inner)))
					break
				}
				suffix := "  " + access
				labelWidth := inner - lipgloss.Width(suffix)
				if labelWidth < 1 {
					labelWidth = 1
				}
				lines = append(lines, m.styles.Text.Render(Truncate(label, labelWidth))+
					m.styles.Muted.Render(suffix))
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
	case "Results":
		if m.results.err != "" {
			lines = append(lines, m.viewResultsError(width-4, bodyHeight)...)
		} else {
			lines = append(lines, m.viewData(width-4, bodyHeight)...)
		}
	case "Data":
		lines = append(lines, m.viewData(width-4, bodyHeight)...)
	case "Indexes":
		lines = append(lines, m.renderDetailPane(indexLines(&m.details, width-4), bodyHeight)...)
	case "Privileges":
		lines = append(lines, m.renderDetailPane(privilegeLines(&m.details), bodyHeight)...)
	default:
		detail := detailLines(&m.details, m.currentDatabase(), m.tree.childCount(m.details.node.Ref.Key()), width-4)
		lines = append(lines, m.renderDetailPane(detail, bodyHeight)...)
	}
	return Box("", strings.Join(lines, "\n"), width, height, m.styles, m.focus == FocusMain)
}

func (m Model) viewResultsError(width, height int) []string {
	lines := []string{m.styles.Error.Render(Truncate("ERROR: "+oneLine(m.results.err), width))}
	lines = append(lines, "")
	for _, line := range wrapText(m.results.statement, width) {
		lines = append(lines, m.styles.Text.Render(line))
	}
	lines = append(lines, "", m.styles.Muted.Render(Truncate(
		"Esc returns to "+oneLine(queryObjectName(m.resultsSaved.node))+".", width)))
	return clipLines(lines, height)
}

func (m Model) renderDetailPane(detail []detailLine, bodyHeight int) []string {
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
	out := make([]string, 0, end-offset+1)
	for _, line := range detail[offset:end] {
		out = append(out, m.renderDetailLine(line))
	}
	if hasPosition {
		out = append(out, m.styles.Muted.Render(detailPosition(offset, end-offset, len(detail))))
	}
	return out
}

func (m Model) renderDetailLine(line detailLine) string {
	switch {
	case line.Rule:
		return m.styles.Border.Render(line.Value)
	case line.Error:
		return m.styles.Error.Render(oneLine(line.Value))
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
	engine := m.profileForStatus().Engine
	label := "SQL"
	if engine == domain.EngineMongoDB {
		label = "MQL"
	}
	marker := "  "
	promptStyle := m.styles.ModeRO
	valueStyle := m.styles.Text
	if m.focus == FocusQuery {
		marker = m.styles.Focused.Render("▸ ")
		promptStyle = m.styles.Focused
		valueStyle = m.styles.Focused
	}
	prompt := promptStyle.Render(label) + " " + m.styles.Accent.Render("›")

	if available, hint := m.queryAvailability(); !available && strings.TrimSpace(m.query.Value()) == "" {
		return Pad(marker+prompt+" "+m.styles.Muted.Render(Truncate(hint, width-10)), width)
	}
	var value string
	switch {
	case m.queryRunning:
		value = m.styles.Muted.Render(Truncate(oneLine(m.query.Value()), width-10))
	case m.focus == FocusQuery:
		value = m.query.View()
	case strings.TrimSpace(m.query.Value()) == "":
		value = m.styles.Muted.Render(Truncate(queryPlaceholder(engine, m.currentQueryRef()), width-10))
	default:
		value = valueStyle.Render(Truncate(oneLine(m.query.Value()), width-10))
	}
	return Pad(marker+prompt+" "+value, width)
}

func queryPlaceholder(engine domain.Engine, ref domain.ObjectRef) string {
	name := ""
	schema := ""
	switch ref.Kind {
	case domain.KindTable, domain.KindCollection, domain.KindView:
		name = oneLine(ref.Name)
		schema = oneLine(ref.Schema)
	}
	if engine == domain.EngineMongoDB {
		if name == "" {
			name = "collection"
		}
		return "db." + name + ".find({}).limit(100)"
	}
	if name == "" {
		name = "table"
	}
	if schema != "" {
		name = schema + "." + name
	}
	return "SELECT * FROM " + name + " LIMIT 100"
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
	if m.queryRunning {
		left = m.resultsSummary()
		style = m.styles.Info
	}
	if m.resultsActive && m.message == "" && !m.queryRunning {
		left = m.resultsSummary()
		style = m.styles.Muted
		if m.queryRunning || m.data.loading {
			style = m.styles.Info
		}
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
		right = "Enter run · Ctrl+Space complete"
	}
	if m.queryRunning {
		right = "Esc cancels"
	}
	if m.screen == ScreenWorkspace && m.gridActive() {
		if m.message == "" && !m.resultsActive {
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
			switch {
			case m.resultsActive:
				right = "↑↓ · m more · / filter · Esc release"
			case m.data.bufferPattern != "":
				right = "n/N match · m more · Esc clear"
			case m.data.serverFilter != nil:
				right = "m more · Ctrl+F regexp · Esc clear"
			default:
				right = "↑↓ · m more · / filter · Ctrl+F · ` JSON"
			}
		case FocusQuery:
			right = "Enter run · Ctrl+Space complete"
		default:
			right = "Enter open"
		}
	}
	if m.queryRunning {
		right = "Esc cancels"
	}
	switch m.data.prompt {
	case promptBuffer:
		right = "type to filter · Enter keep · Esc clear"
	case promptServer:
		right = "regexp · Enter apply · Esc cancel"
	}
	return ComposeLR(style.Render(oneLine(left)), m.styles.Muted.Render(right), width)
}
