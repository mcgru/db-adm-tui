package theme

import "testing"

func TestNightshiftPaletteIsPinned(t *testing.T) {
	want := map[string]string{
		"Base":        "#04060A",
		"Surface":     "#0A0F15",
		"Raised":      "#121A23",
		"Border":      "#26333F",
		"BorderFocus": "#3EE0D0",
		"BorderModal": "#F5B85C",
		"Text":        "#F0F6FA",
		"Muted":       "#94A6B8",
		"Accent":      "#3EE0D0",
		"Accent2":     "#B49BFF",
		"Success":     "#58E6A0",
		"Warning":     "#F5C65F",
		"Error":       "#FF6B7D",
		"Info":        "#6DB5FF",
		"Match":       "#FFD166",
		"Selection":   "#0F3A44",
		"ReadOnly":    "#7CC4FF",
		"ReadWrite":   "#FF9D4D",
	}
	got := map[string]string{
		"Base":        Nightshift.Base,
		"Surface":     Nightshift.Surface,
		"Raised":      Nightshift.Raised,
		"Border":      Nightshift.Border,
		"BorderFocus": Nightshift.BorderFocus,
		"BorderModal": Nightshift.BorderModal,
		"Text":        Nightshift.Text,
		"Muted":       Nightshift.Muted,
		"Accent":      Nightshift.Accent,
		"Accent2":     Nightshift.Accent2,
		"Success":     Nightshift.Success,
		"Warning":     Nightshift.Warning,
		"Error":       Nightshift.Error,
		"Info":        Nightshift.Info,
		"Match":       Nightshift.Match,
		"Selection":   Nightshift.Selection,
		"ReadOnly":    Nightshift.ReadOnly,
		"ReadWrite":   Nightshift.ReadWrite,
	}
	for token, value := range want {
		if got[token] != value {
			t.Fatalf("%s = %q, want %q", token, got[token], value)
		}
	}
}

func TestSelectorStylesAreLegible(t *testing.T) {
	s := New()
	if !s.Select.GetBold() || !s.Select.GetReverse() {
		t.Fatal("selection must remain visible without color (bold+reverse)")
	}
	if !s.Focused.GetBold() {
		t.Fatal("focused frame must be bold")
	}
	if s.Focused.GetForeground() == nil {
		t.Fatal("focused frame must have an accent foreground")
	}
	if s.Select.GetForeground() == nil || s.Select.GetBackground() == nil {
		t.Fatal("selection must carry both foreground and background")
	}
}
