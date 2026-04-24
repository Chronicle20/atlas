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

func TestCalcYOnFootholdFlat(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     1,
		First:  &point.RestModel{X: -100, Y: 100},
		Second: &point.RestModel{X: 100, Y: 100},
	}
	y, ok := calcYOnFoothold(fh, 0)
	if !ok {
		t.Fatalf("calcYOnFoothold flat: ok=false, want true")
	}
	if y != 100 {
		t.Fatalf("calcYOnFoothold flat: y=%d, want 100", y)
	}
}

func TestCalcYOnFootholdDownSlope(t *testing.T) {
	// 200px wide, descends 100px: at x=200 (midpoint), y should be ~150
	fh := &FootholdRestModel{
		Id:     2,
		First:  &point.RestModel{X: 100, Y: 100},
		Second: &point.RestModel{X: 300, Y: 200},
	}
	y, ok := calcYOnFoothold(fh, 200)
	if !ok {
		t.Fatalf("calcYOnFoothold down-slope: ok=false, want true")
	}
	if y < 145 || y > 155 {
		t.Fatalf("calcYOnFoothold down-slope mid: y=%d, want ~150", y)
	}
}

func TestCalcYOnFootholdUpSlope(t *testing.T) {
	// First.Y=200, Second.Y=100 — going right, y decreases
	fh := &FootholdRestModel{
		Id:     3,
		First:  &point.RestModel{X: -300, Y: 200},
		Second: &point.RestModel{X: -100, Y: 100},
	}
	y, ok := calcYOnFoothold(fh, -200)
	if !ok {
		t.Fatalf("calcYOnFoothold up-slope: ok=false, want true")
	}
	if y < 145 || y > 155 {
		t.Fatalf("calcYOnFoothold up-slope mid: y=%d, want ~150", y)
	}
}

func TestCalcYOnFootholdWall(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     4,
		First:  &point.RestModel{X: 500, Y: 0},
		Second: &point.RestModel{X: 500, Y: 200},
	}
	if _, ok := calcYOnFoothold(fh, 500); ok {
		t.Fatalf("calcYOnFoothold wall: ok=true, want false")
	}
}

func TestCalcYOnFootholdOutOfSpan(t *testing.T) {
	fh := &FootholdRestModel{
		Id:     1,
		First:  &point.RestModel{X: -100, Y: 100},
		Second: &point.RestModel{X: 100, Y: 100},
	}
	if _, ok := calcYOnFoothold(fh, 500); ok {
		t.Fatalf("calcYOnFoothold out-of-span (right): ok=true, want false")
	}
	if _, ok := calcYOnFoothold(fh, -500); ok {
		t.Fatalf("calcYOnFoothold out-of-span (left): ok=true, want false")
	}
}
