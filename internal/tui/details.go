package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

type detailsState struct {
	node   domain.Node
	has    bool
	tabs   []string
	tab    int
	offset int

	statistics        domain.Statistics
	statisticsErr     string
	statisticsLoaded  bool
	statisticsLoading bool

	structure        domain.Structure
	structureErr     string
	structureLoaded  bool
	structureLoading bool

	indexes        []domain.Index
	indexesErr     string
	indexesLoaded  bool
	indexesLoading bool

	exactCount   int64
	exactSet     bool
	exactErr     string
	exactLoading bool
	exactStarted time.Time
	exactCancel  context.CancelFunc
}

type detailLine struct {
	Label   string
	Value   string
	Header  bool
	Mutable bool
	Error   bool
	Rule    bool
}

func (d *detailsState) selectNode(node domain.Node) {
	if d.exactCancel != nil {
		d.exactCancel()
	}
	previousTab := ""
	if d.has {
		previousTab = d.currentTab()
	}
	d.node = node
	d.has = true
	d.tabs = tabsFor(node.Kind)
	d.tab = defaultTab(node.Kind)
	if previousTab != "" {
		if index := tabIndex(d.tabs, previousTab); index >= 0 {
			d.tab = index
		}
	}
	d.offset = 0
	d.statistics = domain.Statistics{}
	d.statisticsErr = ""
	d.statisticsLoaded = false
	d.statisticsLoading = false
	d.structure = domain.Structure{}
	d.structureErr = ""
	d.structureLoaded = false
	d.structureLoading = false
	d.indexes = nil
	d.indexesErr = ""
	d.indexesLoaded = false
	d.indexesLoading = false
	d.exactCount = 0
	d.exactSet = false
	d.exactErr = ""
	d.exactLoading = false
	d.exactStarted = time.Time{}
	d.exactCancel = nil
}

func (d *detailsState) clear() {
	if d.exactCancel != nil {
		d.exactCancel()
	}
	*d = detailsState{}
}

func tabsFor(kind domain.ObjectKind) []string {
	switch kind {
	case domain.KindTable, domain.KindCollection:
		return []string{"Details", "Indexes", "Privileges", "Data"}
	case domain.KindView, domain.KindSequence:
		return []string{"Details"}
	case domain.KindDatabase, domain.KindSchema, domain.KindUser, domain.KindRole:
		return []string{"Details", "Privileges"}
	case domain.KindSecurity, domain.KindUsers, domain.KindRoles, domain.KindSchemas:
		return []string{"Details"}
	default:
		return nil
	}
}

// defaultTab returns the tab opened by a new selection. Tables and collections
// open on Details; Enter explicitly switches them to Data.
func defaultTab(domain.ObjectKind) int { return 0 }

func (d *detailsState) currentTab() string {
	if d.tab < 0 || d.tab >= len(d.tabs) {
		return ""
	}
	return d.tabs[d.tab]
}

func detailLines(d *detailsState, currentDB string, childCount, width int) []detailLine {
	node := d.node
	switch node.Kind {
	case domain.KindTable, domain.KindCollection:
		name := node.Name
		if node.Ref.Schema != "" {
			name = node.Ref.Schema + "." + name
		}
		lines := []detailLine{
			{Label: "object", Value: name},
			{Label: "database", Value: node.Ref.Database},
		}
		if node.Ref.Schema != "" {
			lines = append(lines, detailLine{Label: "schema", Value: node.Ref.Schema})
		}
		if node.Access != domain.AccessUnknown {
			lines = append(lines, detailLine{Label: "access", Value: string(node.Access)})
		}
		lines = append(lines, detailLine{Header: true, Value: "Statistics"})
		lines = append(lines, statisticsDetailLines(d)...)
		lines = append(lines, detailLine{}, detailLine{Header: true, Value: "Structure"})
		lines = append(lines, structureDetailLines(d, width)...)
		return lines
	case domain.KindView, domain.KindSequence:
		return []detailLine{{Value: "detail view not available in v1", Mutable: true}}
	case domain.KindDatabase:
		lines := []detailLine{
			{Label: "database", Value: node.Name},
			{Label: "kind", Value: "database"},
		}
		if node.Ref.Database == currentDB {
			lines = append(lines, detailLine{Label: "state", Value: "current"})
		}
		if childCount >= 0 {
			lines = append(lines, detailLine{Label: "children", Value: strconv.Itoa(childCount)})
		}
		return lines
	case domain.KindSchema:
		lines := []detailLine{
			{Label: "schema", Value: node.Name},
			{Label: "database", Value: node.Ref.Database},
			{Label: "kind", Value: "schema"},
		}
		if childCount >= 0 {
			lines = append(lines, detailLine{Label: "children", Value: strconv.Itoa(childCount)})
		}
		return lines
	case domain.KindSchemas:
		lines := []detailLine{
			{Label: "group", Value: "schemas"},
			{Label: "database", Value: node.Ref.Database},
			{Label: "kind", Value: "schema group"},
		}
		if childCount >= 0 {
			lines = append(lines, detailLine{Label: "children", Value: strconv.Itoa(childCount)})
		}
		lines = append(lines, detailLine{Value: "expand to load schemas", Mutable: true})
		return lines
	case domain.KindUser, domain.KindRole:
		lines := []detailLine{
			{Label: "name", Value: node.Name},
			{Label: "kind", Value: string(node.Kind)},
		}
		if node.Scope != "" {
			lines = append(lines, detailLine{Label: "scope", Value: node.Scope})
		}
		if len(node.Attributes) > 0 {
			lines = append(lines, detailLine{Header: true, Value: "attributes"})
			for _, attr := range node.Attributes {
				lines = append(lines, detailLine{Value: "  " + attr})
			}
		}
		return lines
	case domain.KindSecurity:
		return []detailLine{
			{Label: "name", Value: "Security"},
			{Value: "expand users or roles", Mutable: true},
		}
	case domain.KindUsers, domain.KindRoles:
		return []detailLine{
			{Label: "group", Value: string(node.Kind)},
			{Value: "expand to load principals", Mutable: true},
		}
	default:
		return []detailLine{{Value: "nothing selected", Mutable: true}}
	}
}

func statisticsDetailLines(d *detailsState) []detailLine {
	switch {
	case d.statisticsLoading:
		return []detailLine{{Value: "loading statistics…", Mutable: true}}
	case d.statisticsErr != "":
		return []detailLine{{Value: "error: " + d.statisticsErr, Error: true}}
	}
	stats := d.statistics
	count := "unavailable"
	if stats.EstimateAvailable() {
		count = fmt.Sprintf("~%s (%s)", formatCount(stats.EstimatedTotal), stats.EstimateSource)
	}
	if d.exactSet {
		count = formatCount(d.exactCount) + " exact"
	}
	lines := []detailLine{{Label: "count", Value: count}}
	if stats.SizeBytes >= 0 || stats.IndexSizeBytes >= 0 {
		lines = append(lines, detailLine{Label: "size", Value: formatBytes(stats.SizeBytes)})
		if stats.IndexSizeBytes >= 0 {
			lines = append(lines, detailLine{Label: "index size", Value: formatBytes(stats.IndexSizeBytes)})
		}
	} else {
		lines = append(lines, detailLine{Label: "size", Value: "unavailable"})
	}
	switch {
	case d.exactLoading:
		elapsed := int(time.Since(d.exactStarted).Seconds())
		lines = append(lines, detailLine{
			Label: "exact",
			Value: fmt.Sprintf("counting… %ds · Esc cancels", elapsed),
		})
	case d.exactErr != "":
		lines = append(lines, detailLine{Label: "exact", Value: "error: " + d.exactErr, Error: true})
	case !d.exactSet:
		lines = append(lines, detailLine{Label: "exact", Value: "Ctrl+X s computes exact count", Mutable: true})
	}
	return lines
}

func structureDetailLines(d *detailsState, width int) []detailLine {
	switch {
	case d.structureLoading:
		return []detailLine{{Value: "loading structure…", Mutable: true}}
	case d.structureErr != "":
		return []detailLine{{Value: "error: " + d.structureErr, Error: true}}
	case len(d.structure.Fields) == 0:
		return []detailLine{{Value: "structure unavailable", Mutable: true}}
	}
	if d.structure.Source == domain.StructureSample {
		rows := make([][]string, 0, len(d.structure.Fields))
		for _, field := range d.structure.Fields {
			present := "—"
			if field.Sampled > 0 {
				present = fmt.Sprintf("%d/%d (%d%%)",
					field.Present, field.Sampled, field.Present*100/field.Sampled)
			}
			rows = append(rows, []string{field.Name, field.Type, present})
		}
		lines := []detailLine{{
			Value:   fmt.Sprintf("sampled from %d documents", d.structure.Sampled),
			Mutable: true,
		}}
		lines = append(lines, tableDetailLines(
			[]string{"field", "type", "present"}, rows, []int{30, 14, 18}, width)...)
		return lines
	}
	rows := make([][]string, 0, len(d.structure.Fields))
	for _, field := range d.structure.Fields {
		nullable := "no"
		if field.Nullable {
			nullable = "yes"
		}
		defaultValue := field.Default
		if defaultValue == "" {
			defaultValue = "—"
		}
		rows = append(rows, []string{field.Name, field.Type, nullable, defaultValue})
	}
	return tableDetailLines(
		[]string{"column", "type", "null", "default"}, rows, []int{28, 24, 6, 36}, width)
}

func indexLines(d *detailsState, width int) []detailLine {
	switch {
	case d.indexesLoading:
		return []detailLine{{Value: "loading indexes…", Mutable: true}}
	case d.indexesErr != "":
		return []detailLine{{Value: "error: " + d.indexesErr, Error: true}}
	case len(d.indexes) == 0:
		return []detailLine{{Value: "no indexes", Mutable: true}}
	}
	details := make([][]string, 0, len(d.indexes))
	statistics := make([][]string, 0, len(d.indexes))
	structure := make([][]string, 0, len(d.indexes))
	for _, index := range d.indexes {
		unique := "no"
		if index.Unique {
			unique = "yes"
		}
		method := index.Method
		if method == "" {
			method = "—"
		}
		details = append(details, []string{index.Name, unique, method})
		stats := index.Stats
		if stats == "" {
			stats = "—"
		}
		statistics = append(statistics, []string{index.Name, formatBytes(index.SizeBytes), stats})
		columns := strings.Join(index.Columns, ", ")
		if columns == "" {
			columns = "—"
		}
		structure = append(structure, []string{index.Name, columns})
	}
	var lines []detailLine
	lines = append(lines, detailLine{Header: true, Value: "Details"})
	lines = append(lines, tableDetailLines(
		[]string{"index", "unique", "method"}, details, []int{24, 6, 10}, width)...)
	lines = append(lines, detailLine{})
	lines = append(lines, detailLine{Header: true, Value: "Statistics"})
	lines = append(lines, tableDetailLines(
		[]string{"index", "size", "stats"}, statistics, []int{24, 10, 32}, width)...)
	lines = append(lines, detailLine{})
	lines = append(lines, detailLine{Header: true, Value: "Structure"})
	lines = append(lines, tableDetailLines(
		[]string{"index", "columns"}, structure, []int{24, 48}, width)...)
	return lines
}

// tableDetailLines renders a small table with cell separators. Widths are
// content-sized, clamped by maxWidths when supplied.
func tableDetailLines(headers []string, rows [][]string, maxWidths []int, width int) []detailLine {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = lipgloss.Width(oneLine(header))
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			if width := lipgloss.Width(oneLine(row[i])); width > widths[i] {
				widths[i] = width
			}
		}
	}
	for i := range widths {
		if i < len(maxWidths) && maxWidths[i] > 0 && widths[i] > maxWidths[i] {
			widths[i] = maxWidths[i]
		}
		if widths[i] < 3 {
			widths[i] = 3
		}
	}
	fitTableWidths(widths, width)
	format := func(cells []string) string {
		parts := make([]string, len(headers))
		for i := range headers {
			cell := ""
			if i < len(cells) {
				cell = oneLine(cells[i])
			}
			parts[i] = Pad(Truncate(cell, widths[i]), widths[i])
		}
		return strings.Join(parts, " │ ")
	}
	lines := []detailLine{{Header: true, Value: format(headers)}}
	rules := make([]string, len(widths))
	for i, width := range widths {
		rules[i] = strings.Repeat("─", width)
	}
	lines = append(lines, detailLine{Rule: true, Value: strings.Join(rules, "─┼─")})
	for _, row := range rows {
		lines = append(lines, detailLine{Value: format(row)})
	}
	return lines
}

// fitTableWidths shrinks the widest columns until the table fits the pane.
// Each column keeps at least three cells so headers remain visible.
func fitTableWidths(widths []int, width int) {
	if width < 1 || len(widths) == 0 {
		return
	}
	separators := 3 * (len(widths) - 1)
	total := separators
	for _, column := range widths {
		total += column
	}
	for total > width {
		widest := -1
		for i, column := range widths {
			if column > 3 && (widest < 0 || column > widths[widest]) {
				widest = i
			}
		}
		if widest < 0 {
			return
		}
		widths[widest]--
		total--
	}
}

func formatBytes(size int64) string {
	if size < 0 {
		return "unavailable"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	value := float64(size)
	unit := -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func (d *detailsState) scroll(delta, height, total int) {
	if total <= height {
		d.offset = 0
		return
	}
	d.offset += delta
	if d.offset < 0 {
		d.offset = 0
	}
	if d.offset > total-height {
		d.offset = total - height
	}
}

func (d *detailsState) tabNote(index int) string {
	if index < 0 || index >= len(d.tabs) {
		return fmt.Sprintf("tab %d is not available in this context", index+1)
	}
	return ""
}

func detailPosition(offset, visible, total int) string {
	if total <= visible {
		return ""
	}
	return fmt.Sprintf("%d-%d of %d", offset+1, offset+visible, total)
}
