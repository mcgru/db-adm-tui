package tui

import (
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

type treeNode struct {
	Node     domain.Node
	Depth    int
	Parent   string
	Children []string
	Loaded   bool
	Expanded bool
	Loading  bool
	Err      string
}

type treeRow struct {
	Key        string
	Node       domain.Node
	Depth      int
	Expandable bool
	Expanded   bool
	Info       string
	Error      bool
}

type treeState struct {
	nodes       map[string]*treeNode
	roots       []string
	rows        []treeRow
	selected    int
	loadingRoot bool
	rootErr     string

	searchPrompt  bool
	searchPattern string
	searchCursor  int
}

func newTree() treeState {
	return treeState{nodes: map[string]*treeNode{}}
}

func (t *treeState) reset() {
	*t = newTree()
}

func (t *treeState) addRoot(node domain.Node) {
	key := node.Ref.Key()
	if _, ok := t.nodes[key]; ok {
		return
	}
	t.nodes[key] = &treeNode{Node: node}
	t.roots = append(t.roots, key)
}

func (t *treeState) setRoots(nodes []domain.Node) {
	t.roots = nil
	t.nodes = map[string]*treeNode{}
	for _, node := range nodes {
		t.addRoot(node)
	}
	t.loadingRoot = false
	t.rootErr = ""
	t.rebuild()
	t.selectFirst()
}

func (t *treeState) addChildren(parentKey string, nodes []domain.Node) {
	parent, ok := t.nodes[parentKey]
	if !ok {
		return
	}
	selectedKey := t.selectedKey()
	parent.Loaded = true
	parent.Loading = false
	parent.Err = ""
	parent.Children = nil
	for _, node := range nodes {
		key := node.Ref.Key()
		if existing, ok := t.nodes[key]; ok {
			existing.Node = node
			existing.Parent = parentKey
			existing.Depth = parent.Depth + 1
			parent.Children = append(parent.Children, key)
			continue
		}
		t.nodes[key] = &treeNode{Node: node, Parent: parentKey, Depth: parent.Depth + 1}
		parent.Children = append(parent.Children, key)
	}
	t.rebuild()
	t.selectKey(selectedKey)
}

func (t *treeState) setLoading(key string, loading bool) {
	if node, ok := t.nodes[key]; ok {
		node.Loading = loading
		if loading {
			node.Expanded = true
			node.Err = ""
		}
		t.rebuild()
	}
}

func (t *treeState) setError(key, message string) {
	if node, ok := t.nodes[key]; ok {
		node.Loading = false
		node.Err = message
		node.Expanded = true
		t.rebuild()
	}
}

func (t *treeState) toggle(key string) {
	if node, ok := t.nodes[key]; ok && node.Node.Expandable {
		node.Expanded = !node.Expanded
		t.rebuild()
	}
}

func (t *treeState) expand(key string) {
	if node, ok := t.nodes[key]; ok && node.Node.Expandable {
		node.Expanded = true
		t.rebuild()
	}
}

func (t *treeState) collapse(key string) bool {
	if node, ok := t.nodes[key]; ok && node.Expanded {
		node.Expanded = false
		t.rebuild()
		return true
	}
	return false
}

func (t *treeState) rebuild() {
	t.rows = nil
	for _, key := range t.roots {
		t.appendNode(key)
	}
	t.clamp()
}

func (t *treeState) appendNode(key string) {
	node, ok := t.nodes[key]
	if !ok {
		return
	}
	t.rows = append(t.rows, treeRow{
		Key:        key,
		Node:       node.Node,
		Depth:      node.Depth,
		Expandable: node.Node.Expandable,
		Expanded:   node.Expanded,
	})
	if !node.Expanded {
		return
	}
	if node.Loading {
		t.rows = append(t.rows, treeRow{Key: key + "\x00loading", Depth: node.Depth + 1, Info: "loading…"})
		return
	}
	if node.Err != "" {
		t.rows = append(t.rows, treeRow{Key: key + "\x00error", Depth: node.Depth + 1, Info: "error " + node.Err, Error: true})
		return
	}
	if node.Loaded && len(node.Children) == 0 {
		t.rows = append(t.rows, treeRow{Key: key + "\x00empty", Depth: node.Depth + 1, Info: "no objects visible"})
		return
	}
	for _, child := range node.Children {
		t.appendNode(child)
	}
}

func (t *treeState) clamp() {
	if len(t.rows) == 0 {
		t.selected = 0
		return
	}
	if t.selected < 0 {
		t.selected = 0
	}
	if t.selected >= len(t.rows) {
		t.selected = len(t.rows) - 1
	}
	if t.rows[t.selected].Info != "" {
		t.move(1)
	}
}

func (t *treeState) move(delta int) {
	if len(t.rows) == 0 {
		return
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	target := t.selected + delta
	if target < 0 {
		target = 0
	}
	if target >= len(t.rows) {
		target = len(t.rows) - 1
	}
	for pos := target; pos >= 0 && pos < len(t.rows); pos += step {
		if t.rows[pos].Info == "" {
			t.selected = pos
			return
		}
	}
}

func (t *treeState) selectFirst() {
	for i, row := range t.rows {
		if row.Info == "" {
			t.selected = i
			return
		}
	}
}

func (t *treeState) selectLast() {
	for i := len(t.rows) - 1; i >= 0; i-- {
		if t.rows[i].Info == "" {
			t.selected = i
			return
		}
	}
}

func (t *treeState) selectKey(key string) {
	for i, row := range t.rows {
		if row.Key == key && row.Info == "" {
			t.selected = i
			return
		}
	}
}

func (t *treeState) selectedNode() (domain.Node, bool) {
	if t.selected < 0 || t.selected >= len(t.rows) {
		return domain.Node{}, false
	}
	row := t.rows[t.selected]
	if row.Info != "" {
		return domain.Node{}, false
	}
	return row.Node, true
}

func (t *treeState) selectedKey() string {
	if t.selected < 0 || t.selected >= len(t.rows) {
		return ""
	}
	return t.rows[t.selected].Key
}

func (t *treeState) node(key string) (*treeNode, bool) {
	node, ok := t.nodes[key]
	return node, ok
}

func (t *treeState) childCount(key string) int {
	if node, ok := t.nodes[key]; ok && node.Loaded {
		return len(node.Children)
	}
	return -1
}

func (t *treeState) objectCount() int {
	count := 0
	for _, node := range t.nodes {
		switch node.Node.Kind {
		case domain.KindTable, domain.KindCollection, domain.KindView, domain.KindSequence,
			domain.KindUser, domain.KindRole:
			count++
		}
	}
	return count
}

func (t treeState) visibleRows(height int) []treeRow {
	if height <= 0 || len(t.rows) == 0 {
		return nil
	}
	start := t.selected - height/2
	if start < 0 {
		start = 0
	}
	if start+height > len(t.rows) {
		start = len(t.rows) - height
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(t.rows) {
		end = len(t.rows)
	}
	return t.rows[start:end]
}

func treeRowText(row treeRow) string {
	line := treeRowLabel(row)
	if access := treeRowAccess(row); access != "" {
		line += "  " + access
	}
	return line
}

// treeRowLabel is the row text without the access marker, so the view can
// render the marker in a quieter style than the object name.
func treeRowLabel(row treeRow) string {
	indent := strings.Repeat("  ", row.Depth)
	if row.Info != "" {
		return indent + row.Info
	}
	prefix := "·"
	if row.Expandable {
		prefix = "▸"
		if row.Expanded {
			prefix = "▾"
		}
	}
	return indent + prefix + " " + oneLine(row.Node.Name)
}

func treeRowAccess(row treeRow) string {
	if row.Node.HasAccessMarker() && row.Node.Access != domain.AccessUnknown {
		return string(row.Node.Access)
	}
	return ""
}
