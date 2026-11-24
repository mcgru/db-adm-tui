package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/tui/keymap"
)

func (m *Model) openImportModal() {
	discovery := m.svc.Discover()
	m.importEntries = discovery.Entries
	m.importStatus = discovery.Status
	m.importEmpty = discovery.Empty
	m.importSel = map[string]bool{}
	for _, entry := range discovery.Entries {
		if entry.Selectable && (entry.Kind == "pgpass" || entry.Kind == "mypass") {
			m.importSel[entry.ID] = true
		}
	}
	m.importCursor = 0
	m.importOffset = 0
	m.importNotice = ""
	m.importAllowNoTLS = false
	m.modal = modalImport
	m.message = ""
	m.messageLevel = levelNone
}

func (m *Model) toggleImportNoTLS() {
	m.importAllowNoTLS = !m.importAllowNoTLS
	m.importNotice = ""
}

func (m Model) handleImportKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "е" {
		m.toggleImportNoTLS()
		return m, nil
	}
	action, ok := keymap.Lookup(msg.String())
	if !ok {
		return m, nil
	}
	switch action {
	case keymap.Cancel:
		m.modal = modalNone
		m.importNotice = ""
	case keymap.AllowNoTLS:
		m.toggleImportNoTLS()
	case keymap.Up:
		m.importMove(-1)
	case keymap.Down:
		m.importMove(1)
	case keymap.PageUp:
		m.importMove(-8)
	case keymap.PageDown:
		m.importMove(8)
	case keymap.Home:
		m.importCursor = 0
		m.importOffset = 0
	case keymap.End:
		if len(m.importEntries) > 0 {
			m.importCursor = len(m.importEntries) - 1
			m.importMove(0)
		}
	case keymap.Space:
		if entry, ok := m.currentImportEntry(); ok && entry.Selectable {
			m.importSel[entry.ID] = !m.importSel[entry.ID]
			m.importNotice = ""
		}
	case keymap.Confirm:
		var selected []app.ImportEntry
		skipped := 0
		for _, entry := range m.importEntries {
			if !entry.Selectable {
				skipped++
				continue
			}
			if m.importSel[entry.ID] {
				if entry.Engine == domain.EnginePostgreSQL {
					entry.AllowNoTLS = m.importAllowNoTLS
				}
				selected = append(selected, entry)
			}
		}
		if len(selected) == 0 {
			if skipped > 0 {
				m.importNotice = fmt.Sprintf("No selectable entries; %d skipped.", skipped)
			} else {
				m.importNotice = "No entries selected."
			}
			return m, nil
		}
		summary, err := m.svc.Import(context.Background(), selected)
		if err != nil {
			m.importNotice = oneLine(err.Error())
			return m, nil
		}
		summary.Skipped += skipped
		m.modal = modalNone
		m.importNotice = ""
		m.reloadProfiles(context.Background(), "")
		m.setMessage(levelSuccess, importSummaryText(summary))
	}
	return m, nil
}

func importSummaryText(summary app.ImportSummary) string {
	total := summary.Created + summary.Updated
	var text string
	switch {
	case total == 0:
		text = "nothing imported"
	case total == 1 && summary.Created == 1:
		text = "imported 1 profile"
	case total == 1:
		text = "refreshed 1 profile"
	default:
		text = fmt.Sprintf("imported %d profiles", total)
	}
	if summary.Skipped > 0 {
		text += fmt.Sprintf(", %d skipped", summary.Skipped)
	}
	return text
}

func (m *Model) importMove(delta int) {
	if len(m.importEntries) == 0 {
		return
	}
	m.importCursor += delta
	if m.importCursor < 0 {
		m.importCursor = 0
	}
	if m.importCursor >= len(m.importEntries) {
		m.importCursor = len(m.importEntries) - 1
	}
	rows := m.importVisibleRows()
	if rows < 1 {
		rows = 1
	}
	if m.importCursor < m.importOffset {
		m.importOffset = m.importCursor
	}
	if m.importCursor >= m.importOffset+rows {
		m.importOffset = m.importCursor - rows + 1
	}
	if m.importOffset < 0 {
		m.importOffset = 0
	}
}

func (m Model) importVisibleRows() int {
	statusLines := 0
	inner := modalWidth(m.width, 76) - 4
	for _, status := range m.importStatus {
		statusLines += len(wrapText(status, inner))
	}
	rows := m.height - 8 - statusLines - 1
	if m.importHasPostgres() {
		rows--
	}
	if m.importNotice != "" {
		rows--
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m Model) importHasPostgres() bool {
	for _, entry := range m.importEntries {
		if entry.Engine == domain.EnginePostgreSQL {
			return true
		}
	}
	return false
}

func (m Model) currentImportEntry() (app.ImportEntry, bool) {
	if m.importCursor < 0 || m.importCursor >= len(m.importEntries) {
		return app.ImportEntry{}, false
	}
	return m.importEntries[m.importCursor], true
}

func (m Model) viewImport() string {
	width := modalWidth(m.width, 76)
	inner := width - 4
	var body []string

	for _, status := range m.importStatus {
		for _, line := range wrapText(status, inner) {
			body = append(body, m.styles.Warning.Render(line))
		}
	}
	if len(m.importStatus) > 0 && len(m.importEntries) > 0 {
		body = append(body, "")
	}

	if m.importEmpty || (len(m.importEntries) == 0 && len(m.importStatus) == 0) {
		for _, line := range wrapText("No ~/.pgpass, ~/.mypass, or ~/.mongo.* files found. Press n to create a connection.", inner) {
			body = append(body, m.styles.Muted.Render(line))
		}
	} else if len(m.importEntries) > 0 {
		visible := m.importVisibleRows()
		end := m.importOffset + visible
		if end > len(m.importEntries) {
			end = len(m.importEntries)
		}
		for i := m.importOffset; i < end; i++ {
			body = append(body, m.importLine(i, m.importEntries[i], inner))
		}
	}

	if m.importNotice != "" {
		body = append(body, m.styles.Error.Render(Truncate(m.importNotice, inner)))
	}
	if m.importHasPostgres() {
		check := "[ ]"
		if m.importAllowNoTLS {
			check = "[x]"
		}
		body = append(body, m.styles.Accent.Render("t")+" "+
			m.styles.Muted.Render("no TLS ")+m.styles.Text.Render(check)+
			m.styles.Muted.Render("  applies to imported PostgreSQL profiles"))
	}
	footer := strings.Join([]string{
		m.styles.Accent.Render("Space") + " select",
		m.styles.Accent.Render("Enter") + " import",
		m.styles.Accent.Render("Esc") + " cancel",
	}, " · ")
	body = append(body, "", footer)
	return modalBox("import "+strconv.Itoa(len(m.importEntries)), body, width, m.styles)
}

func (m Model) importLine(index int, entry app.ImportEntry, width int) string {
	check := "☐"
	if m.importSel[entry.ID] {
		check = "☑"
	}
	if !entry.Selectable {
		check = " "
	}
	selector := importSelector(entry)
	tag := ""
	tagStyle := m.styles.Muted
	if entry.Warning != "" {
		tag = "warning: " + entry.Warning
		tagStyle = m.styles.Warning
	}
	if entry.Error != "" {
		tag = "error: " + entry.Error
		tagStyle = m.styles.Error
	}
	if tag != "" {
		selector = tag
	}
	plain := strings.Join([]string{
		check,
		field(entry.Source, 18),
		field(selector, width-32),
		entry.Engine.DisplayName(),
	}, " ")
	if index == m.importCursor {
		return m.styles.Select.Render(Pad(plain, width))
	}
	return m.styles.Text.Render(check) + " " +
		m.styles.Muted.Render(field(entry.Source, 18)) + " " +
		tagStyle.Render(field(selector, width-32)) + " " +
		m.styles.Accent2.Render(entry.Engine.DisplayName())
}

func importSelector(entry app.ImportEntry) string {
	if entry.User != "" {
		return fmt.Sprintf("%s@%s/%s", entry.User, fieldAddr(entry.Host, entry.Port), entry.Database)
	}
	if entry.Host != "" {
		return fmt.Sprintf("%s/%s", fieldAddr(entry.Host, entry.Port), entry.Database)
	}
	if entry.Line > 0 {
		return fmt.Sprintf("line %d", entry.Line)
	}
	return ""
}

func fieldAddr(host string, port int) string {
	if host == "" {
		return ""
	}
	if port > 0 {
		return fmt.Sprintf("%s:%d", host, port)
	}
	return host
}
