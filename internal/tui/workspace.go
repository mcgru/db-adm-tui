package tui

import (
	"strings"

	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

type treeRow struct {
	prefix   string
	text     string
	marker   string
	indent   int
	security bool
}

var fakeTree = []treeRow{
	{prefix: "▾", text: "orders"},
	{prefix: "▾", text: "public", indent: 1},
	{prefix: "·", text: "customers", marker: "ro", indent: 2},
	{prefix: "·", text: "invoices", marker: "rw", indent: 2},
	{prefix: "·", text: "audit_log", marker: "—", indent: 2},
	{prefix: "▸", text: "billing"},
	{prefix: "▾", text: "Security", security: true},
	{prefix: "·", text: "users", indent: 1, security: true},
	{prefix: "·", text: "roles", indent: 1, security: true},
}

func (r treeRow) line() string {
	line := strings.Repeat("  ", r.indent) + r.prefix + " " + r.text
	if r.marker != "" {
		line += "  " + r.marker
	}
	return line
}

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
	p := m.profile()
	left := m.styles.Text.Render(p.Label) +
		m.styles.Muted.Render(" · "+p.Engine+" · orders · ") +
		m.styles.ModeRO.Render("r/o")
	right := m.styles.Muted.Render(p.User)
	return ComposeLR(left, right, width)
}

func (m Model) viewTree(height, width int) string {
	inner := width - 4
	lines := make([]string, 0, len(fakeTree))
	for i, row := range fakeTree {
		line := row.line()
		switch {
		case i == m.treeSel:
			lines = append(lines, m.styles.Select.Render(Pad(line, inner)))
		case row.security:
			lines = append(lines, m.styles.Info.Render(line))
		default:
			lines = append(lines, m.styles.Text.Render(line))
		}
	}
	return Box("objects 3", strings.Join(lines, "\n"), width, height, m.styles, m.focus == FocusTree)
}

func (m Model) viewMain(height, width int) string {
	tabs := []string{"1 Data", "2 Details", "3 Privileges"}
	parts := make([]string, 0, len(tabs))
	for i, label := range tabs {
		if i == m.tab {
			parts = append(parts, m.styles.Accent.Bold(true).Underline(true).Render(label))
		} else {
			parts = append(parts, m.styles.Label.Render(label))
		}
	}
	lines := []string{strings.Join(parts, m.styles.Muted.Render(" · ")), ""}

	switch m.tab {
	case 1:
		lines = append(lines,
			m.styles.Header.Render("estimated   12,480 rows"),
			m.styles.Text.Render("size        2.1 GB"),
			m.styles.Header.Render("indexes"),
			m.styles.Text.Render("  customers_pkey"),
			m.styles.Header.Render("structure"),
			m.styles.Text.Render("  id          bigint"),
			m.styles.Text.Render("  email       text"),
			m.styles.Text.Render("  status      text"),
		)
	case 2:
		lines = append(lines,
			m.styles.Header.Render("grants"),
			m.styles.Text.Render("  CONNECT   orders"),
			m.styles.Text.Render("  USAGE     public"),
			m.styles.Text.Render("  SELECT    customers"),
			m.styles.Header.Render("memberships"),
			m.styles.Text.Render("  reporting (via role)"),
			m.styles.Header.Render("effective"),
			m.styles.Muted.Render("  native terms only"),
		)
	default:
		lines = append(lines,
			m.styles.Header.Render("id    email                    status"),
			m.styles.Text.Render("1041  anna@example.com         active"),
			m.styles.Select.Render(Pad("1042  ali@example.com          active", width-4)),
			m.styles.Text.Render("1043  mark@example.com         blocked"),
			m.styles.Text.Render("1044  nina@example.com         active"),
			m.styles.Text.Render("1045  omar@example.com         active"),
			m.styles.Text.Render("1046  sara@example.com         blocked"),
		)
	}
	return Box("", strings.Join(lines, "\n"), width, height, m.styles, m.focus == FocusMain)
}

func (m Model) viewQuery(width int) string {
	label := "SQL"
	if m.profile().Engine == "MongoDB" {
		label = "MQL"
	}
	promptStyle := m.styles.ModeRO
	if m.focus == FocusQuery {
		promptStyle = m.styles.Focused
	}
	prompt := promptStyle.Render(label) + " " + m.styles.Accent.Render("›")
	statement := m.styles.Text.Render("SELECT * FROM public.customers WHERE status = 'active' LIMIT 100;")
	return Pad(prompt+" "+statement, width)
}

func (m Model) viewBottom(width int) string {
	left := "loaded 100 of ~12,480 · case: sensitive · / search · ? help"
	style := m.styles.Muted
	if m.message != "" {
		left = m.message
		style = m.styles.Info
	}
	if m.pendingChord {
		left = "Ctrl+X …"
		style = m.styles.Accent
	}
	right := "Tab focus"
	if m.focus == FocusQuery {
		right = "Tab focus · Ctrl+X chord"
	}
	return ComposeLR(style.Render(Truncate(left, width-20)), m.styles.Muted.Render(right), width)
}
