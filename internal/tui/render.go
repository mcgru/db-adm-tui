package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

// StripANSI removes SGR/OSC sequences so golden tests compare layout, not
// terminal color support.
func StripANSI(s string) string {
	return ansi.Strip(s)
}

// Sanitize removes terminal escape and control sequences from external text.
// It preserves newlines and tabs for legitimate layout input. C0/C1 controls,
// DEL, and format controls (bidi/zero-width) are stripped.
func Sanitize(s string) string {
	s = ansi.Strip(s)
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) || isUnsafeFormat(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isUnsafeFormat strips bidi overrides/isolates that can disguise terminal
// text, while preserving ZWNJ/ZWJ, RTL marks, and variation selectors needed by
// legitimate labels and emoji sequences.
func isUnsafeFormat(r rune) bool {
	return (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069)
}

// Truncate keeps at most width cells, ending with an ellipsis when content is
// cut. It is ANSI-aware and never splits an escape sequence.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

func Pad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w > width {
		return Truncate(s, width)
	}
	if w == width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func ComposeLR(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	rw := lipgloss.Width(right)
	if rw >= width {
		return Truncate(right, width)
	}
	maxLeft := width - rw - 1
	if lipgloss.Width(left) > maxLeft {
		left = Truncate(left, maxLeft)
	}
	gap := width - lipgloss.Width(left) - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func FitLines(lines []string, width, height int) []string {
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out = append(out, Pad(lines[i], width))
		} else {
			out = append(out, strings.Repeat(" ", width))
		}
	}
	return out
}

func JoinColumns(left, right []string, gap int) []string {
	height := len(left)
	if len(right) > height {
		height = len(right)
	}
	out := make([]string, 0, height)
	spacer := strings.Repeat(" ", gap)
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, l+spacer+r)
	}
	return out
}

// Box draws a single-line frame with an optional title, one cell of inner
// padding, and a focused border/title when focused. width and height include
// the frame. Content is expected to be pre-styled.
func Box(title, body string, width, height int, st theme.Styles, focused bool) string {
	if width < 4 || height < 2 {
		return Truncate(body, width)
	}
	border := st.Border
	titleStyle := st.Header
	if focused {
		border = st.Focused
		titleStyle = st.Focused
	}
	innerW := width - 4
	innerH := height - 2

	titlePart := ""
	if title != "" {
		titlePart = " " + Truncate(title, width-4) + " "
	}
	fill := width - 2 - lipgloss.Width(titlePart)
	if fill < 0 {
		fill = 0
	}
	top := border.Render("┌") + titleStyle.Render(titlePart) + border.Render(strings.Repeat("─", fill)+"┐")

	lines := strings.Split(body, "\n")
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	content := make([]string, 0, innerH)
	for _, line := range lines {
		content = append(content, border.Render("│")+" "+Pad(line, innerW)+" "+border.Render("│"))
	}
	for len(content) < innerH {
		content = append(content, border.Render("│")+" "+strings.Repeat(" ", innerW)+" "+border.Render("│"))
	}
	bottom := border.Render("└" + strings.Repeat("─", width-2) + "┘")
	return strings.Join(append(append([]string{top}, content...), bottom), "\n")
}
