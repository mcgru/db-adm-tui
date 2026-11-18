package tui

import (
	"fmt"
	"strconv"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

type detailsState struct {
	node   domain.Node
	has    bool
	tabs   []string
	tab    int
	offset int
}

type detailLine struct {
	Label   string
	Value   string
	Header  bool
	Mutable bool
}

func (d *detailsState) selectNode(node domain.Node) {
	d.node = node
	d.has = true
	d.tabs = tabsFor(node.Kind)
	d.tab = defaultTab(node.Kind)
	d.offset = 0
}

func (d *detailsState) clear() {
	*d = detailsState{}
}

func tabsFor(kind domain.ObjectKind) []string {
	switch kind {
	case domain.KindTable, domain.KindCollection:
		return []string{"Data", "Details", "Privileges"}
	case domain.KindView, domain.KindSequence:
		return []string{"Details"}
	case domain.KindDatabase, domain.KindSchema, domain.KindUser, domain.KindRole:
		return []string{"Details", "Privileges"}
	case domain.KindSecurity, domain.KindUsers, domain.KindRoles:
		return []string{"Details"}
	default:
		return nil
	}
}

// defaultTab returns the tab opened by a new selection. Tables and collections
// open on Data; every other context opens Details.
func defaultTab(domain.ObjectKind) int { return 0 }

func (d *detailsState) currentTab() string {
	if d.tab < 0 || d.tab >= len(d.tabs) {
		return ""
	}
	return d.tabs[d.tab]
}

func detailLines(node domain.Node, currentDB string, childCount int) []detailLine {
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
		lines = append(lines, detailLine{Value: "statistics are not available in this story", Mutable: true})
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
