package keymap

// Action is a semantic shell action. UI code never branches on raw key strings.
type Action string

const (
	Quit            Action = "quit"
	ForceQuit       Action = "force-quit"
	Confirm         Action = "confirm"
	New             Action = "new"
	Import          Action = "import"
	EditProfile     Action = "edit-profile"
	Help            Action = "help"
	Up              Action = "up"
	Down            Action = "down"
	Left            Action = "left"
	Right           Action = "right"
	Home            Action = "home"
	End             Action = "end"
	RangeStart      Action = "range-start"
	RangeEnd        Action = "range-end"
	PageUp          Action = "page-up"
	PageDown        Action = "page-down"
	Tab             Action = "tab"
	ShiftTab        Action = "shift-tab"
	Tab1            Action = "tab-1"
	Tab2            Action = "tab-2"
	Tab3            Action = "tab-3"
	Tab4            Action = "tab-4"
	PrevTab         Action = "prev-tab"
	NextTab         Action = "next-tab"
	Disconnect      Action = "disconnect"
	Search          Action = "search"
	More            Action = "more"
	ExactCount      Action = "exact-count"
	ToggleMode      Action = "toggle-mode"
	Matrix          Action = "matrix"
	Edit            Action = "edit"
	CaseMode        Action = "case-mode"
	Execute         Action = "execute"
	Completion      Action = "completion"
	ContextHelp     Action = "context-help"
	ServerFilter    Action = "server-filter"
	FetchAll        Action = "fetch-all"
	ExpandJSON      Action = "expand-json"
	Space           Action = "space"
	NextMatch       Action = "next-match"
	Add             Action = "add"
	Remove          Action = "remove"
	DeleteProfile   Action = "delete-profile"
	ResizeTreeLeft  Action = "resize-tree-left"
	ResizeTreeRight Action = "resize-tree-right"
	ResizeQueryUp   Action = "resize-query-up"
	ResizeQueryDown Action = "resize-query-down"
	Cancel          Action = "cancel"
	ChordX          Action = "chord-x"
)

var single = map[string]Action{
	// Latin and Cyrillic physical positions.
	"n": New, "т": New,
	"N": NextMatch, "Т": NextMatch,
	"i": Import, "ш": Import,
	"e": EditProfile, "у": EditProfile,
	"m": More, "ь": More,
	"?": Help, ",": Help,
	"f1": Help,

	"enter":     Confirm,
	"esc":       Cancel,
	"tab":       Tab,
	"shift+tab": ShiftTab,
	"up":        Up,
	"down":      Down,
	"left":      Left,
	"right":     Right,
	"home":      Home,
	"end":       End,
	"ctrl+home": RangeStart,
	"ctrl+end":  RangeEnd,
	"pgup":      PageUp,
	"pgdown":    PageDown,
	"space":     Space,
	" ":         Space,

	"1": Tab1,
	"2": Tab2,
	"3": Tab3,
	"4": Tab4,
	"[": PrevTab, "х": PrevTab,
	"]": NextTab, "ъ": NextTab,

	"/": Search, ".": Search,
	"`": ExpandJSON, "ё": ExpandJSON,
	"+": Add, "=": Add,
	"-": Remove,

	"ctrl+a":     FetchAll,
	"delete":     DeleteProfile,
	"ctrl+f":     ServerFilter,
	"ctrl+space": Completion, "ctrl+@": Completion,
	"ctrl+f1":   ContextHelp,
	"ctrl+left": ResizeTreeLeft, "ctrl+right": ResizeTreeRight,
	"ctrl+up": ResizeQueryUp, "ctrl+down": ResizeQueryDown,

	"ctrl+d": Disconnect, "ctrl+в": Disconnect,
	"ctrl+x": ChordX, "ctrl+ч": ChordX,
	"ctrl+q": Quit, "ctrl+й": Quit,
	"ctrl+c": ForceQuit, "ctrl+с": ForceQuit,
}

var chord = map[string]Action{
	"ctrl+w": ToggleMode, "ctrl+ц": ToggleMode, "ц": ToggleMode,
	"ctrl+r": Matrix, "ctrl+к": Matrix, "к": Matrix,
	"ctrl+e": Edit, "ctrl+у": Edit, "у": Edit,
	"ctrl+x": Execute, "ctrl+ч": Execute,
	"s": ExactCount, "ы": ExactCount,
	"i": CaseMode, "ш": CaseMode,
	"ctrl+c": ForceQuit, "ctrl+с": ForceQuit,
}

func Lookup(key string) (Action, bool) {
	a, ok := single[key]
	return a, ok
}

func LookupChord(key string) (Action, bool) {
	a, ok := chord[key]
	return a, ok
}
