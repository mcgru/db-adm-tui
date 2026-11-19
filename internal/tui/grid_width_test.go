package tui

import "testing"

func TestFitColumnWidths(t *testing.T) {
	cases := []struct {
		name      string
		widths    []int
		available int
		want      []int
	}{
		{"expand", []int{20, 20}, 50, []int{25, 24}},
		{"waterfill", []int{50, 50}, 82, []int{41, 40}},
		{"narrow-first", []int{10, 50}, 42, []int{10, 31}},
		{"scroll-fallback", []int{20, 20, 20, 20}, 42, []int{20, 20, 20, 20}},
	}
	for _, tc := range cases {
		got := fitColumnWidths(tc.widths, tc.available)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: %v", tc.name, got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
			}
		}
	}
}

func TestFitColumnWidthsUsesFullPane(t *testing.T) {
	widths := fitColumnWidths([]int{50, 50}, 82)
	total := 0
	for _, width := range widths {
		total += width
	}
	total += len(widths) - 1
	if total != 82 {
		t.Fatalf("total = %d, want 82", total)
	}
}

func TestFitColumnWidthsFallbackCaps(t *testing.T) {
	widths := fitColumnWidths([]int{100, 100, 100, 100}, 42)
	for _, width := range widths {
		if width > dataMaxColumnWidth {
			t.Fatalf("width = %d", width)
		}
	}
}
