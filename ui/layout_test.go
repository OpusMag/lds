package ui

import "testing"

func TestComputeLayout(t *testing.T) {
	tests := []struct {
		name                                                 string
		w, h                                                 int
		wantBoxWidth, wantHalfBoxHeight, wantIncreasedBoxHeight int
	}{
		{"standard 80x24", 80, 24, 40, 6, 18},
		{"wide 120x40", 120, 40, 60, 10, 30},
		{"tiny 10x10", 10, 10, 5, 2, 7},
		{"odd 81x25", 81, 25, 40, 6, 18},
		{"min 1x1", 1, 1, 0, 0, 0},
		{"zero", 0, 0, 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := ComputeLayout(tc.w, tc.h)
			if l.Width != tc.w || l.Height != tc.h {
				t.Errorf("dims: got (%d,%d) want (%d,%d)", l.Width, l.Height, tc.w, tc.h)
			}
			if l.BoxWidth != tc.wantBoxWidth {
				t.Errorf("BoxWidth: got %d want %d", l.BoxWidth, tc.wantBoxWidth)
			}
			if l.HalfBoxHeight != tc.wantHalfBoxHeight {
				t.Errorf("HalfBoxHeight: got %d want %d", l.HalfBoxHeight, tc.wantHalfBoxHeight)
			}
			if l.IncreasedBoxHeight != tc.wantIncreasedBoxHeight {
				t.Errorf("IncreasedBoxHeight: got %d want %d", l.IncreasedBoxHeight, tc.wantIncreasedBoxHeight)
			}
		})
	}
}

func TestComputeLayoutInvariants(t *testing.T) {
	// IncreasedBoxHeight == h/2 + (h/2)/2
	for h := 0; h < 200; h++ {
		l := ComputeLayout(100, h)
		want := h/2 + (h/2)/2
		if l.IncreasedBoxHeight != want {
			t.Errorf("h=%d: IncreasedBoxHeight got %d want %d", h, l.IncreasedBoxHeight, want)
		}
		if l.HalfBoxHeight != (h/2)/2 {
			t.Errorf("h=%d: HalfBoxHeight got %d want %d", h, l.HalfBoxHeight, (h/2)/2)
		}
	}
}
