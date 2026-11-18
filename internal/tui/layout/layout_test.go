package layout

import "testing"

func TestComputeAtFloor(t *testing.T) {
	r := Compute(80, 24)
	if r.TooSmall {
		t.Fatal("80x24 must not be too small")
	}
	if r.TopHeight != 1 || r.BottomHeight != 1 || r.QueryHeight != 1 {
		t.Fatalf("unexpected chrome: %+v", r)
	}
	if r.BodyHeight != 21 {
		t.Fatalf("body height = %d, want 21", r.BodyHeight)
	}
	if r.TreeWidth != 28 || r.MainWidth != 51 {
		t.Fatalf("tree/main = %d/%d, want 28/51", r.TreeWidth, r.MainWidth)
	}
}

func TestComputeTooSmall(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{79, 24},
		{80, 23},
		{72, 20},
	} {
		if r := Compute(tc.w, tc.h); !r.TooSmall {
			t.Fatalf("%dx%d should be too small", tc.w, tc.h)
		}
	}
}

func TestComputeWide(t *testing.T) {
	r := Compute(100, 30)
	if r.TooSmall {
		t.Fatal("100x30 must fit")
	}
	if r.BodyHeight != 27 {
		t.Fatalf("body height = %d, want 27", r.BodyHeight)
	}
	if r.TreeWidth != 28 || r.MainWidth != 71 {
		t.Fatalf("tree/main = %d/%d, want 28/71", r.TreeWidth, r.MainWidth)
	}
}
