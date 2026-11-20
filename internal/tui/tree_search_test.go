package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestTreeSearchPromptNavigatesMatches(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindTable)
	for i, row := range m.tree.rows {
		if row.Node.Name == "invoices" {
			m.tree.selected = i
			break
		}
	}
	m, _ = press(t, m, "/")
	if !m.tree.searchPrompt {
		t.Fatal("/ did not open the tree search prompt")
	}
	m = typeString(t, m, "customer")
	if got := len(m.treeSearchMatches()); got != 2 {
		t.Fatalf("matches = %d, want 2", got)
	}
	if node, _ := m.tree.selectedNode(); node.Name != "customers" {
		t.Fatalf("typing selected %+v, want the first match", node)
	}
	view := StripANSI(m.View().Content)
	if !strings.Contains(view, "/ customer") {
		t.Fatalf("tree search bar missing:\n%s", view)
	}
	m, _ = press(t, m, "enter")
	if m.tree.searchPrompt {
		t.Fatal("Enter did not keep the tree search")
	}
	node, ok := m.tree.selectedNode()
	if !ok || node.Name != "customers" {
		t.Fatalf("Enter selected %+v, want the first match", node)
	}
	m, _ = press(t, m, "n")
	node, ok = m.tree.selectedNode()
	if !ok || node.Name != "customer_view" {
		t.Fatalf("n selected %+v", node)
	}
	m, _ = press(t, m, "N")
	node, ok = m.tree.selectedNode()
	if !ok || node.Name != "customers" {
		t.Fatalf("N selected %+v", node)
	}
	m, _ = press(t, m, "esc")
	if m.tree.searchPattern != "" || m.tree.searchPrompt {
		t.Fatalf("Esc left search state: %q", m.tree.searchPattern)
	}
}

func TestTreeSearchSurvivesExpandableEnter(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindDatabase)
	m.tree.searchPattern = "o"
	m.tree.searchPrompt = false
	m, _ = press(t, m, "enter")
	if m.tree.searchPattern != "o" {
		t.Fatalf("expanding a database cleared the search: %q", m.tree.searchPattern)
	}
}

func TestTreeSearchMovesBelowTheCurrentRow(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindTable)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "customer")
	m, _ = press(t, m, "enter")
	// Leave the matching row and walk into a non-matching table below it.
	m, _ = press(t, m, "down")
	if node, _ := m.tree.selectedNode(); node.Name != "invoices" {
		t.Fatalf("walk selected %+v", node)
	}
	m, _ = press(t, m, "n")
	if node, _ := m.tree.selectedNode(); node.Name != "customer_view" {
		t.Fatalf("n from a non-match selected %+v, want customer_view", node)
	}
	m, _ = press(t, m, "N")
	if node, _ := m.tree.selectedNode(); node.Name != "customers" {
		t.Fatalf("N from a non-match selected %+v, want customers", node)
	}
}

func TestTreeSearchHighlightsMatches(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindTable)
	m.tree.searchPattern = "customer"
	m.tree.searchPrompt = false
	view := m.View().Content
	if !strings.Contains(view, m.styles.Match.Render("customer")) {
		t.Fatalf("match style missing from tree view:\n%s", view)
	}
}

func TestTreeSearchNAtPromptInserts(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "n")
	if m.tree.searchPattern != "n" {
		t.Fatalf("n while prompt active = %q", m.tree.searchPattern)
	}
	if node, _ := m.tree.selectedNode(); node.Name == "customer_view" {
		t.Fatal("n navigated instead of typing")
	}
}

func TestTreeSearchLeavesMainPaneFilterAlone(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "tab")
	if m.focus != FocusMain {
		t.Fatalf("focus = %v", m.focus)
	}
	m, _ = press(t, m, "/")
	if m.tree.searchPrompt {
		t.Fatal("main-pane / opened tree search")
	}
	if m.data.prompt != promptBuffer {
		t.Fatalf("data prompt = %v", m.data.prompt)
	}
}

func TestTreeSearchDoubleSlashInsertsSelectedName(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindTable)
	node, ok := m.tree.selectedNode()
	if !ok || node.Name == "" {
		t.Fatal("no table selected")
	}
	m, _ = press(t, m, "/")
	m, _ = press(t, m, "/")
	if m.tree.searchPattern != node.Name {
		t.Fatalf("double slash pattern = %q, want %q", m.tree.searchPattern, node.Name)
	}
	if !m.tree.searchPrompt {
		t.Fatal("double slash closed the search prompt")
	}
}

func TestTreeSearchInsertKeyInsertsQualifiedName(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindTable)
	node, ok := m.tree.selectedNode()
	if !ok || node.Ref.Schema == "" {
		t.Fatalf("selected table has no schema: %+v", node)
	}
	m, _ = press(t, m, "insert")
	if !m.tree.searchPrompt {
		t.Fatal("Ins did not open the tree search prompt")
	}
	want := node.Ref.Schema + "." + node.Name
	if m.tree.searchPattern != want {
		t.Fatalf("Ins pattern = %q, want %q", m.tree.searchPattern, want)
	}
}
