package layout

const (
	MinWidth   = 80
	MinHeight  = 24
	TreeWidth  = 28
	Gap        = 1
	QueryLines = 1

	MinTreeWidth  = 16
	MinQueryLines = 1
	MaxQueryLines = 3
)

// Regions describes the shell split. Height is the terminal height; BodyHeight
// excludes the top status, query bar, and bottom status.
type Regions struct {
	Width, Height int
	TooSmall      bool

	TopHeight    int
	BodyHeight   int
	QueryHeight  int
	BottomHeight int
	TreeWidth    int
	MainWidth    int
	Gap          int
}

// Compute uses the 80x24 default tree width and one query line.
func Compute(width, height int) Regions {
	return ComputeSize(width, height, TreeWidth, QueryLines)
}

// ComputeSize clamps persisted tree and query sizes to the current terminal.
// The saved values stay untouched; the caller can apply them again when the
// terminal grows back.
func ComputeSize(width, height, treeWidth, queryHeight int) Regions {
	r := Regions{Width: width, Height: height, Gap: Gap}
	if width < MinWidth || height < MinHeight {
		r.TooSmall = true
		return r
	}

	r.TopHeight = 1
	r.BottomHeight = 1
	r.QueryHeight = ClampQueryHeight(queryHeight)
	r.BodyHeight = height - r.TopHeight - r.QueryHeight - r.BottomHeight
	r.TreeWidth = ClampTreeWidth(width, treeWidth)
	r.MainWidth = width - r.TreeWidth - r.Gap
	return r
}

// ClampTreeWidth keeps the tree at 16 cells through half the terminal width.
func ClampTreeWidth(width, treeWidth int) int {
	if treeWidth < MinTreeWidth {
		treeWidth = MinTreeWidth
	}
	max := width / 2
	if max < MinTreeWidth {
		max = MinTreeWidth
	}
	if treeWidth > max {
		treeWidth = max
	}
	return treeWidth
}

// ClampQueryHeight keeps the query bar at one to three lines.
func ClampQueryHeight(queryHeight int) int {
	if queryHeight < MinQueryLines {
		return MinQueryLines
	}
	if queryHeight > MaxQueryLines {
		return MaxQueryLines
	}
	return queryHeight
}
