package tui

import (
	"strings"

	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

func field(s string, width int) string {
	return Pad(Truncate(s, width), width)
}

func profileFields(p Profile) string {
	return strings.Join([]string{
		field(p.Label, 14),
		field(p.Engine, 10),
		field(p.Host, 19),
		field(p.User, 11),
		field(p.Source, 14),
		field(p.LastUsed, 7),
	}, " ")
}

func (m Model) profileLine(p Profile, selected bool, width int) string {
	plain := profileFields(p)
	if selected {
		return m.styles.Select.Render(Pad(plain, width))
	}
	label := field(p.Label, 14)
	engine := field(p.Engine, 10)
	rest := strings.Join([]string{
		field(p.Host, 19),
		field(p.User, 11),
		field(p.Source, 14),
		field(p.LastUsed, 7),
	}, " ")
	return m.styles.Text.Render(label) + " " + m.styles.Accent2.Render(engine) + " " + m.styles.Muted.Render(rest)
}

func (m Model) viewPicker(regions layout.Regions) string {
	width, height := regions.Width, regions.Height
	title := m.styles.Title.Render("db-adm-tui") + " " + m.styles.Muted.Render("· choose a connection")
	header := m.styles.Header.Render(strings.Join([]string{
		field("label", 14),
		field("engine", 10),
		field("host", 19),
		field("user", 11),
		field("source", 14),
		field("last", 7),
	}, " "))

	lines := []string{title, "", header}
	for i, p := range m.profiles {
		lines = append(lines, m.profileLine(p, i == m.pickerSel, width))
	}
	lines = append(lines, "")

	message := ""
	if m.message != "" {
		message = m.styles.Info.Render(Truncate(m.message, width))
	}
	lines = append(lines, message)

	footer := strings.Join([]string{
		m.styles.Accent.Render("Enter") + " connect",
		m.styles.Accent.Render("n") + " new",
		m.styles.Accent.Render("i") + " import",
		m.styles.Accent.Render("?") + " help",
		m.styles.Accent.Render("q") + " quit",
	}, " · ")
	for len(lines) < height-1 {
		lines = append(lines, "")
	}
	lines = append(lines, footer)
	return strings.Join(FitLines(lines, width, height), "\n")
}
