package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/state"
)

func TestResizePersistsPerProfile(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	layoutPath := filepath.Join(e.home, "state", "db-adm-tui", "layouts.json")
	layoutStore := state.NewStore(layoutPath)
	e.svc.SetLayoutStore(layoutStore)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	if m.treeWidth != 28 || m.queryHeight != 1 {
		t.Fatalf("defaults = %d/%d", m.treeWidth, m.queryHeight)
	}
	m, _ = press(t, m, "ctrl+right")
	m, _ = press(t, m, "ctrl+right")
	m, _ = press(t, m, "ctrl+up")
	m, _ = press(t, m, "ctrl+up")
	if m.treeWidth != 32 || m.queryHeight != 3 {
		t.Fatalf("resized = %d/%d", m.treeWidth, m.queryHeight)
	}
	if got := m.regions().TreeWidth; got != 32 {
		t.Fatalf("regions tree = %d", got)
	}
	if got := m.regions().QueryHeight; got != 3 {
		t.Fatalf("regions query = %d", got)
	}
	if m.viewQuery(80) == "" || strings.Count(m.viewQuery(80), "\n") != 2 {
		t.Fatalf("three-line query missing:\n%s", m.viewQuery(80))
	}
	saved, err := layoutStore.Load(t.Context(), m.layoutKey)
	if err != nil {
		t.Fatal(err)
	}
	if saved.TreeWidth != 32 || saved.QueryHeight != 3 {
		t.Fatalf("saved = %+v", saved)
	}

	next := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	if next.treeWidth != 32 || next.queryHeight != 3 {
		t.Fatalf("reloaded = %d/%d", next.treeWidth, next.queryHeight)
	}
}

func TestResizeFromQueryFocus(t *testing.T) {
	m := workspace(t)
	m = focusQuery(t, m)
	m, _ = press(t, m, "ctrl+right")
	m, _ = press(t, m, "ctrl+up")
	if m.treeWidth != 30 || m.queryHeight != 2 {
		t.Fatalf("resized from query = %d/%d", m.treeWidth, m.queryHeight)
	}
	if m.focus != FocusQuery {
		t.Fatalf("resize moved focus to %v", m.focus)
	}
}

func TestResizeClampsWithoutOverwritingOnSmallTerminal(t *testing.T) {
	m := workspace(t)
	m.treeWidth = 60
	m.queryHeight = 3
	regions := m.regions()
	if regions.TreeWidth != 40 {
		t.Fatalf("display tree = %d, want 40", regions.TreeWidth)
	}
	if m.treeWidth != 60 {
		t.Fatalf("saved tree changed to %d", m.treeWidth)
	}
	m, _ = press(t, m, "ctrl+left")
	if m.treeWidth != 38 {
		t.Fatalf("explicit resize = %d, want 38", m.treeWidth)
	}
	m, _ = press(t, m, "ctrl+down")
	if m.queryHeight != 2 {
		t.Fatalf("query height = %d, want 2", m.queryHeight)
	}
}

func TestQueryHeightWrapsWithoutChangingLogicalStatement(t *testing.T) {
	m := workspace(t)
	m.queryHeight = 3
	statement := "SELECT 1234567890 FROM public.customers WHERE status = 'active' ORDER BY id LIMIT 100"
	m.query.SetValue(statement)
	m.query.SetCursor(len([]rune(statement)))
	view := m.viewQuery(80)
	if strings.Count(view, "\n") != 2 {
		t.Fatalf("query height lines = %d:\n%s", strings.Count(view, "\n")+1, view)
	}
	if !strings.Contains(StripANSI(view), "SELECT 1234567890") {
		t.Fatalf("logical statement missing:\n%s", StripANSI(view))
	}
	if m.query.Value() != statement {
		t.Fatal("display wrapping changed the statement")
	}
	if lines := strings.Split(m.View().Content, "\n"); len(lines) != 24 {
		t.Fatalf("view height = %d, want 24", len(lines))
	}
	line, _ := m.queryCursor(80)
	if line == 0 {
		t.Fatal("cursor did not move to the wrapped line")
	}
	x, y := m.completionAnchor(20, 6)
	if x < 8 || y < 0 {
		t.Fatalf("completion anchor = %d,%d", x, y)
	}
}
