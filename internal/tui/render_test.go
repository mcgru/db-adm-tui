package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/mcgru/db-adm-tui/internal/tui/theme"
)

func TestStripAndSanitize(t *testing.T) {
	styled := "\x1b[31mhi\x1b[0m"
	if got := StripANSI(styled); got != "hi" {
		t.Fatalf("StripANSI = %q", got)
	}
	if got := Sanitize(styled + "\x07\n\tok\x00\x7f"); got != "hi\n\tok" {
		t.Fatalf("Sanitize = %q", got)
	}
}

func TestTruncateIsANSISafe(t *testing.T) {
	styled := "\x1b[38;2;1;2;3mabcdef\x1b[0m"
	got := Truncate(styled, 3)
	if w := lipgloss.Width(got); w != 3 {
		t.Fatalf("width = %d, want 3 (%q)", w, got)
	}
	plain := StripANSI(got)
	if !strings.HasPrefix(plain, "ab") || !strings.HasSuffix(plain, "…") {
		t.Fatalf("plain = %q, want ab…", plain)
	}
	if strings.Contains(plain, "\x1b") {
		t.Fatal("truncate split an escape sequence")
	}
	if Truncate("abc", 0) != "" {
		t.Fatal("width 0 must be empty")
	}
	if Pad("abc", 1) != "…" {
		t.Fatalf("Pad narrow = %q", Pad("abc", 1))
	}
}

func TestPadPreservesExactWidthStyledString(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("abc")
	if got := Pad(styled, 3); got != styled {
		t.Fatalf("Pad changed an exact-width string: %q", got)
	}
	padded := Pad(styled, 5)
	if lipgloss.Width(padded) != 5 {
		t.Fatalf("padded width = %d, want 5", lipgloss.Width(padded))
	}
	if StripANSI(padded) != "abc  " {
		t.Fatalf("padded plain = %q", StripANSI(padded))
	}
}

func TestComposeLR(t *testing.T) {
	got := ComposeLR("abc", "xyz", 10)
	if got != "abc    xyz" {
		t.Fatalf("ComposeLR = %q", got)
	}
	got = ComposeLR("abcdef", "0123456789ABC", 10)
	if lipgloss.Width(got) != 10 {
		t.Fatalf("width = %d, want 10 (%q)", lipgloss.Width(got), got)
	}
	if ComposeLR("a", "b", 0) != "" {
		t.Fatal("width 0 must be empty")
	}
}

func TestFitLines(t *testing.T) {
	lines := FitLines([]string{"a", "bb"}, 5, 4)
	if len(lines) != 4 {
		t.Fatalf("len = %d, want 4", len(lines))
	}
	for i, line := range lines {
		if lipgloss.Width(line) != 5 {
			t.Fatalf("line %d width = %d, want 5", i, lipgloss.Width(line))
		}
	}
	if lines[0] != "a    " || lines[2] != "     " {
		t.Fatalf("unexpected lines: %q", lines)
	}
}

func TestJoinColumns(t *testing.T) {
	got := JoinColumns([]string{"a", "b"}, []string{"1", "2", "3"}, 2)
	want := []string{"a  1", "b  2", "  3"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBoxDimensionsAndFocus(t *testing.T) {
	st := theme.New()
	body := "one\ntwo\nthree\nfour"
	got := Box("title", body, 12, 4, st, true)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("height = %d, want 4", len(lines))
	}
	for i, line := range lines {
		if lipgloss.Width(line) != 12 {
			t.Fatalf("line %d width = %d, want 12 (%q)", i, lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(got, st.Focused.Render("┌")) {
		t.Fatal("focused box must use the focused border")
	}
	if !strings.Contains(got, st.Focused.Render(" title ")) {
		t.Fatal("focused box must use the focused title style")
	}
	if strings.Contains(StripANSI(got), "four") {
		t.Fatal("box must clip body lines to its inner height")
	}

	plain := Box("", "x", 6, 3, st, false)
	if !strings.Contains(plain, st.Border.Render("┌")) {
		t.Fatal("unfocused box must use the regular border")
	}
}

func TestBoxTinyDegrades(t *testing.T) {
	st := theme.New()
	got := Box("title", "body", 3, 1, st, false)
	if got != "bo…" {
		t.Fatalf("tiny box = %q, want truncated body", got)
	}
}
