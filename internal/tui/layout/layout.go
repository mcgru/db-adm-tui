package layout

const (
	MinWidth   = 80
	MinHeight  = 24
	TreeWidth  = 28
	Gap        = 1
	QueryLines = 1
)

// Regions describes the 80x24 shell split. Height is the terminal height;
// BodyHeight excludes the top status, query bar, and bottom status.
//
// The tree width and query height are fixed in story 1. Resizable, persisted
// sizes (EXPERIENCE.md -> Resize) land with the workspace polish story; at the
// 80x24 floor the fixed values always fit, so no clamp branches exist here yet.
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

func Compute(width, height int) Regions {
	r := Regions{Width: width, Height: height, Gap: Gap}
	if width < MinWidth || height < MinHeight {
		r.TooSmall = true
		return r
	}

	r.TopHeight = 1
	r.BottomHeight = 1
	r.QueryHeight = QueryLines
	r.BodyHeight = height - r.TopHeight - r.QueryHeight - r.BottomHeight
	r.TreeWidth = TreeWidth
	r.MainWidth = width - r.TreeWidth - r.Gap
	return r
}
