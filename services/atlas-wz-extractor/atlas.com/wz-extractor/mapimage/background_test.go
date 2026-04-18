package mapimage

import "testing"

func TestBackgroundTypeHorizontal(t *testing.T) {
	cases := []struct {
		typ  BackgroundType
		want bool
	}{
		{BackgroundNormal, false},
		{BackgroundHTile, true},
		{BackgroundVTile, false},
		{BackgroundBothTile, true},
		{BackgroundHScroll, true},
		{BackgroundVScroll, false},
		{BackgroundBothH, true},
		{BackgroundBothV, true},
	}
	for _, c := range cases {
		if got := c.typ.Horizontal(); got != c.want {
			t.Errorf("BackgroundType(%d).Horizontal() = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestBackgroundTypeVertical(t *testing.T) {
	cases := []struct {
		typ  BackgroundType
		want bool
	}{
		{BackgroundNormal, false},
		{BackgroundHTile, false},
		{BackgroundVTile, true},
		{BackgroundBothTile, true},
		{BackgroundHScroll, false},
		{BackgroundVScroll, true},
		{BackgroundBothH, true},
		{BackgroundBothV, true},
	}
	for _, c := range cases {
		if got := c.typ.Vertical(); got != c.want {
			t.Errorf("BackgroundType(%d).Vertical() = %v, want %v", c.typ, got, c.want)
		}
	}
}
