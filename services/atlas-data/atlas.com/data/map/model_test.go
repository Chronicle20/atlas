package _map

import (
	"atlas-data/point"
	"testing"
)

func buildSampleTree() *FootholdTreeRestModel {
	tree := NewFootholdTree(-1000, -1000, 1000, 1000)
	footholds := []FootholdRestModel{
		{Id: 1, First: &point.RestModel{X: -100, Y: 100}, Second: &point.RestModel{X: 100, Y: 100}},   // flat
		{Id: 2, First: &point.RestModel{X: 100, Y: 100}, Second: &point.RestModel{X: 300, Y: 200}},    // down-slope
		{Id: 3, First: &point.RestModel{X: -300, Y: 200}, Second: &point.RestModel{X: -100, Y: 100}},  // up-slope
		{Id: 4, First: &point.RestModel{X: 500, Y: 0}, Second: &point.RestModel{X: 500, Y: 200}},      // wall
	}
	return tree.Insert(footholds)
}

func TestFootholdFindById(t *testing.T) {
	tree := buildSampleTree()

	if fh := tree.findById(1); fh == nil || fh.Id != 1 {
		t.Fatalf("findById(1) = %v, want id=1", fh)
	}
	if fh := tree.findById(4); fh == nil || fh.Id != 4 {
		t.Fatalf("findById(4) = %v, want id=4 (wall)", fh)
	}
	if fh := tree.findById(999); fh != nil {
		t.Fatalf("findById(999) = %v, want nil", fh)
	}
}

func TestCalcYOnFoothold(t *testing.T) {
	flat := &FootholdRestModel{Id: 1, First: &point.RestModel{X: -100, Y: 100}, Second: &point.RestModel{X: 100, Y: 100}}
	downSlope := &FootholdRestModel{Id: 2, First: &point.RestModel{X: 100, Y: 100}, Second: &point.RestModel{X: 300, Y: 200}}
	upSlope := &FootholdRestModel{Id: 3, First: &point.RestModel{X: -300, Y: 200}, Second: &point.RestModel{X: -100, Y: 100}}
	wall := &FootholdRestModel{Id: 4, First: &point.RestModel{X: 500, Y: 0}, Second: &point.RestModel{X: 500, Y: 200}}

	tests := []struct {
		name     string
		fh       *FootholdRestModel
		x        int16
		wantOK   bool
		wantYMin int16
		wantYMax int16
	}{
		{name: "flat_returns_y1", fh: flat, x: 0, wantOK: true, wantYMin: 100, wantYMax: 100},
		{name: "down_slope_midpoint", fh: downSlope, x: 200, wantOK: true, wantYMin: 145, wantYMax: 155},
		{name: "up_slope_midpoint", fh: upSlope, x: -200, wantOK: true, wantYMin: 145, wantYMax: 155},
		{name: "wall_unwalkable", fh: wall, x: 500, wantOK: false},
		{name: "out_of_span_right", fh: flat, x: 500, wantOK: false},
		{name: "out_of_span_left", fh: flat, x: -500, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y, ok := calcYOnFoothold(tt.fh, tt.x)
			if ok != tt.wantOK {
				t.Fatalf("ok=%t, want %t", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if y < tt.wantYMin || y > tt.wantYMax {
				t.Fatalf("y=%d, want in [%d, %d]", y, tt.wantYMin, tt.wantYMax)
			}
		})
	}
}
