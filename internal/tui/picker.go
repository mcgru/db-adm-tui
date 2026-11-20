package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/layout"
)

const (
	colLabel  = 14
	colEngine = 10
	colHost   = 19
	colUser   = 11
	colSource = 14
	colLast   = 7
)

func field(s string, width int) string {
	return Pad(Truncate(oneLine(s), width), width)
}

func lastUsed(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "—"
	}
	delta := time.Since(*t)
	switch {
	case delta < 0 || delta < time.Minute:
		return "now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	default:
		return t.UTC().Format("01-02")
	}
}

func sourceField(view app.ProfileView, connecting bool, p domain.Profile) string {
	switch {
	case connecting:
		return "connecting…"
	case view.SourceState == "ignored":
		return "ignored"
	case view.SourceState == "missing":
		return "missing"
	case view.SourceState == "unreadable":
		return "unreadable"
	case view.SourceState != "":
		return view.SourceState
	case view.Source != "":
		return view.Source
	default:
		return p.SourceLabel()
	}
}

func profileFields(view app.ProfileView, connecting bool) string {
	p := view.Profile
	last := lastUsed(p.LastUsed)
	if connecting {
		last = ""
	}
	return strings.Join([]string{
		field(p.Label, colLabel),
		field(p.Engine.DisplayName(), colEngine),
		field(p.Addr(), colHost),
		field(p.User, colUser),
		field(sourceField(view, connecting, p), colSource),
		field(last, colLast),
	}, " ")
}

func (m Model) profileLine(view app.ProfileView, selected, connecting bool, width int) string {
	p := view.Profile
	plain := profileFields(view, connecting)
	if selected {
		return m.styles.Select.Render(Pad(plain, width))
	}
	source := sourceField(view, connecting, p)
	sourceStyle := m.styles.Muted
	switch view.SourceState {
	case "ignored", "missing", "unreadable", "error", "stale":
		sourceStyle = m.styles.Warning
	}
	if connecting {
		sourceStyle = m.styles.Info
	}
	label := field(p.Label, colLabel)
	engine := field(p.Engine.DisplayName(), colEngine)
	host := field(p.Addr(), colHost)
	user := field(p.User, colUser)
	last := field(lastUsed(p.LastUsed), colLast)
	if connecting {
		last = strings.Repeat(" ", colLast)
	}
	return m.styles.Text.Render(label) + " " +
		m.styles.Accent2.Render(engine) + " " +
		m.styles.Muted.Render(host) + " " +
		m.styles.Muted.Render(user) + " " +
		sourceStyle.Render(field(source, colSource)) + " " +
		m.styles.Muted.Render(last)
}

func (m Model) viewPicker(regions layout.Regions) string {
	width, height := regions.Width, regions.Height
	hasStatus := m.connecting || m.message != ""
	contentLimit := height - 1
	if hasStatus {
		contentLimit--
	}
	if contentLimit < 0 {
		contentLimit = 0
	}
	count := "0 profiles"
	if len(m.profiles) == 1 {
		count = "1 profile"
	} else if len(m.profiles) > 1 {
		count = strconv.Itoa(len(m.profiles)) + " profiles"
	}
	title := m.styles.Title.Render("db-adm-tui") + " " + m.styles.Muted.Render("· choose a connection")
	lines := []string{ComposeLR(title, m.styles.Muted.Render(count), width), ""}

	if m.storageErr {
		for _, line := range wrapText("Profile storage is unreadable. Showing no profiles.", width) {
			lines = append(lines, m.styles.Error.Render(line))
		}
		lines = append(lines, "")
	}

	if len(m.profiles) == 0 {
		for _, line := range wrapText("No profiles yet. Press n to create one, or i to import from ~/.pgpass, ~/.mypass, or ~/.mongo.*.", width) {
			lines = append(lines, m.styles.Muted.Render(line))
		}
	} else {
		header := m.styles.Header.Render(strings.Join([]string{
			field("label", colLabel),
			field("engine", colEngine),
			field("host", colHost),
			field("user", colUser),
			field("source", colSource),
			field("last", colLast),
		}, " "))
		lines = append(lines, header)
		end := m.pickerOffset + m.pickerVisibleRows()
		if end > len(m.profiles) {
			end = len(m.profiles)
		}
		for i := m.pickerOffset; i < end; i++ {
			if len(lines) >= contentLimit {
				break
			}
			view := m.profiles[i]
			p := view.Profile
			connecting := m.connecting && m.connectingKey == p.Key()
			lines = append(lines, m.profileLine(view, i == m.pickerSel, connecting, width))
			if message := m.rowErrors[p.Key()]; message != "" && len(lines) < contentLimit {
				lines = append(lines, m.styles.Error.Render(Truncate("  error "+message, width)))
			}
		}
	}

	status := ""
	if m.connecting {
		status = m.styles.Info.Render(Truncate("connecting to "+oneLine(m.connectingLabel)+"… · Esc cancels", width))
	} else if m.message != "" {
		status = m.messageStyle().Render(Truncate(oneLine(m.message), width))
	}
	for len(lines) < contentLimit {
		lines = append(lines, "")
	}
	if len(lines) > contentLimit {
		lines = lines[:contentLimit]
	}
	if status != "" {
		lines = append(lines, status)
	}
	footer := strings.Join([]string{
		m.styles.Accent.Render("Enter") + " connect",
		m.styles.Accent.Render("n") + " new",
		m.styles.Accent.Render("i") + " import",
		m.styles.Accent.Render("e") + " edit",
		m.styles.Accent.Render("Del/-") + " delete",
		m.styles.Accent.Render("?") + " help",
		m.styles.Accent.Render("Ctrl+Q") + " quit",
	}, " · ")
	lines = append(lines, footer)
	return strings.Join(FitLines(lines, width, height), "\n")
}
