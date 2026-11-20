package keymap

import "testing"

func TestLatinCyrillicAliases(t *testing.T) {
	pairs := []struct {
		latin, cyrillic string
		want            Action
	}{
		{"n", "т", New},
		{"i", "ш", Import},
		{"e", "у", EditProfile},
		{"ctrl+q", "ctrl+й", Quit},
		{"m", "ь", More},
		{"`", "ё", ExpandJSON},
		{"?", ",", Help},
		{"[", "х", PrevTab},
		{"]", "ъ", NextTab},
		{"/", ".", Search},
		{"ctrl+d", "ctrl+в", Disconnect},
		{"ctrl+x", "ctrl+ч", ChordX},
		{"ctrl+c", "ctrl+с", ForceQuit},
	}
	for _, pair := range pairs {
		a, ok := Lookup(pair.latin)
		if !ok {
			t.Fatalf("Lookup(%q) not found", pair.latin)
		}
		b, ok := Lookup(pair.cyrillic)
		if !ok {
			t.Fatalf("Lookup(%q) not found", pair.cyrillic)
		}
		if a != pair.want || b != pair.want {
			t.Fatalf("%q -> %q, %q -> %q; want %q", pair.latin, a, pair.cyrillic, b, pair.want)
		}
	}
}

func TestPlainQIsNotBound(t *testing.T) {
	for _, key := range []string{"q", "й"} {
		if action, ok := Lookup(key); ok {
			t.Fatalf("Lookup(%q) = %q, want unbound", key, action)
		}
	}
}

func TestSingleActions(t *testing.T) {
	cases := map[string]Action{
		"enter":      Confirm,
		"esc":        Cancel,
		"tab":        Tab,
		"shift+tab":  ShiftTab,
		"up":         Up,
		"down":       Down,
		"left":       Left,
		"right":      Right,
		"home":       Home,
		"end":        End,
		"ctrl+home":  RangeStart,
		"ctrl+end":   RangeEnd,
		"pgup":       PageUp,
		"pgdown":     PageDown,
		"space":      Space,
		" ":          Space,
		"e":          EditProfile,
		"у":          EditProfile,
		"1":          Tab1,
		"2":          Tab2,
		"3":          Tab3,
		"4":          Tab4,
		"N":          NextMatch,
		"Т":          NextMatch,
		"f1":         Help,
		"ctrl+a":     FetchAll,
		"delete":     DeleteProfile,
		"ctrl+f":     ServerFilter,
		"`":          ExpandJSON,
		"ё":          ExpandJSON,
		"ctrl+space": Completion,
		"ctrl+@":     Completion,
		"ctrl+f1":    ContextHelp,
		"+":          Add,
		"=":          Add,
		"-":          Remove,
		"u":          AddUser,
		"г":          AddUser,
		"ctrl+n":     AddUser,
		"ctrl+т":     AddUser,
		"h":          HomeDatabase,
		"р":          HomeDatabase,
		"r":          MatrixPrivileges,
		"к":          MatrixPrivileges,
		"w":          TreeMode,
		"ц":          TreeMode,
		"insert":     InsertSelected,
		"ctrl+left":  ResizeTreeLeft,
		"ctrl+right": ResizeTreeRight,
		"ctrl+up":    ResizeQueryUp,
		"ctrl+down":  ResizeQueryDown,
		"ctrl+q":     Quit,
		"ctrl+c":     ForceQuit,
	}
	for key, want := range cases {
		got, ok := Lookup(key)
		if !ok || got != want {
			t.Fatalf("Lookup(%q) = %q, %v; want %q", key, got, ok, want)
		}
	}
}

func TestChords(t *testing.T) {
	cases := map[string]Action{
		"ctrl+w": ToggleMode,
		"ctrl+ц": ToggleMode,
		"ц":      ToggleMode,
		"ctrl+e": Edit,
		"ctrl+у": Edit,
		"у":      Edit,
		"ctrl+r": Matrix,
		"ctrl+к": Matrix,
		"к":      Matrix,
		"ctrl+x": Execute,
		"ctrl+ч": Execute,
		"s":      ExactCount,
		"ы":      ExactCount,
		"i":      CaseMode,
		"ш":      CaseMode,
		"ctrl+c": ForceQuit,
		"ctrl+с": ForceQuit,
	}
	for key, want := range cases {
		got, ok := LookupChord(key)
		if !ok || got != want {
			t.Fatalf("LookupChord(%q) = %q, %v; want %q", key, got, ok, want)
		}
	}
	if _, ok := LookupChord("?"); ok {
		t.Fatal("? must not be a chord final")
	}
}
