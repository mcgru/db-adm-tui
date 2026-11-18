package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

// modalBox draws an amber-bordered modal with one cell of inner padding.
// width includes the frame; the returned height is len(body)+2.
func modalBox(title string, body []string, width int, st theme.Styles) string {
	if width < 8 {
		width = 8
	}
	innerW := width - 4
	titlePart := ""
	if title != "" {
		titlePart = " " + Truncate(title, width-4) + " "
	}
	fill := width - 2 - lipgloss.Width(titlePart)
	if fill < 0 {
		fill = 0
	}
	top := st.BorderModal.Render("┌") + st.Title.Render(titlePart) +
		st.BorderModal.Render(strings.Repeat("─", fill)+"┐")
	lines := make([]string, 0, len(body)+2)
	lines = append(lines, top)
	for _, line := range body {
		lines = append(lines, st.BorderModal.Render("│")+" "+Pad(line, innerW)+" "+st.BorderModal.Render("│"))
	}
	bottom := st.BorderModal.Render("└" + strings.Repeat("─", width-2) + "┘")
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

// overlayCenter draws modal over the already-laid-out base view. The base is
// stripped of styling so the modal is the only colored surface, matching the
// inactive-background rule for modals.
func overlayCenter(base, modal string, width, height int) string {
	if width <= 0 || height <= 0 || modal == "" {
		return base
	}
	modalLines := strings.Split(modal, "\n")
	modalWidth := 0
	for _, line := range modalLines {
		if w := lipgloss.Width(line); w > modalWidth {
			modalWidth = w
		}
	}
	modalHeight := len(modalLines)
	if modalWidth > width-2 {
		modalWidth = width - 2
	}
	if modalHeight > height-2 {
		modalHeight = height - 2
	}
	x := (width - modalWidth) / 2
	y := (height - modalHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	baseLines := strings.Split(base, "\n")
	out := make([]string, 0, height)
	for row := 0; row < height; row++ {
		line := ""
		if row < len(baseLines) {
			line = StripANSI(baseLines[row])
		}
		line = Pad(Truncate(line, width), width)
		if row >= y && row < y+modalHeight {
			modalLine := modalLines[row-y]
			if lipgloss.Width(modalLine) > modalWidth {
				modalLine = Truncate(modalLine, modalWidth)
			}
			modalLine = Pad(modalLine, modalWidth)
			left := ansi.Cut(line, 0, x)
			right := ansi.Cut(line, x+modalWidth, width)
			out = append(out, left+modalLine+right)
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// wrapText word-wraps text at width cells. Long words are hard-truncated by
// the caller's Pad/Truncate when rendered.
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	text = oneLine(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, word := range words {
		for lipgloss.Width(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, ansi.Cut(word, 0, width))
			word = ansi.Cut(word, width, lipgloss.Width(word))
		}
		if word == "" {
			continue
		}
		if current == "" {
			current = word
			continue
		}
		if lipgloss.Width(current)+1+lipgloss.Width(word) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// modalWidth clamps a modal width to the terminal and the DESIGN limits.
func modalWidth(terminalWidth int, preferred int) int {
	if preferred > terminalWidth-4 {
		preferred = terminalWidth - 4
	}
	if preferred < 8 {
		preferred = 8
	}
	return preferred
}
