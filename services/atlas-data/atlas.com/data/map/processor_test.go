package _map

import (
	"atlas-data/map/monster"
	monstertpl "atlas-data/monster"
	"atlas-data/point"
	"errors"
	"testing"
)

var errTestMissing = errors.New("test: template missing")

func snapTestTree() FootholdTreeRestModel {
	tree := NewFootholdTree(-2000, -2000, 2000, 2000)
	footholds := []FootholdRestModel{
		{Id: 10, First: &point.RestModel{X: -200, Y: 100}, Second: &point.RestModel{X: 200, Y: 100}}, // flat
		{Id: 11, First: &point.RestModel{X: 200, Y: 100}, Second: &point.RestModel{X: 400, Y: 200}},  // down-slope
	}
	return *tree.Insert(footholds)
}

func groundLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: false, Swimming: false}, nil
}
func flyingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: true}, nil
}
func swimmingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Swimming: true}, nil
}
func errLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{}, errTestMissing
}

func TestSnapToGround_FhSet_Flat_CorrectsY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 10}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 100 {
		t.Fatalf("flat fh snap: Y=%d, want 100", out.Y)
	}
}

func TestSnapToGround_FhSet_Slope_CorrectsY(t *testing.T) {
	tree := snapTestTree()
	// midpoint of foothold 11 (x=300) is y≈150
	sp := monster.RestModel{Id: 0, Template: 100100, X: 300, Y: 80, FH: 11}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y < 145 || out.Y > 155 {
		t.Fatalf("slope fh snap: Y=%d, want ~150", out.Y)
	}
}

func TestSnapToGround_FhSet_Missing_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 9999}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 80 {
		t.Fatalf("missing fh: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_FlyingMob_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 2230000, X: 0, Y: -300, FH: 0}
	out := snapToGround(tree, sp, flyingLookup)
	if out.Y != -300 {
		t.Fatalf("flying mob: Y=%d, want -300 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_SwimmingMob_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 2230100, X: 0, Y: 50, FH: 0}
	out := snapToGround(tree, sp, swimmingLookup)
	if out.Y != 50 {
		t.Fatalf("swimming mob: Y=%d, want 50 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_GroundMob_FindsBelow(t *testing.T) {
	tree := snapTestTree()
	// X=0 is over flat foothold (Id=10) at Y=100. Spawn at Y=80, expect snap to 99 (Y-1 offset).
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 0}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 99 {
		t.Fatalf("ground mob fh=0 findBelow: Y=%d, want 99", out.Y)
	}
}

func TestSnapToGround_FhZero_NoFootholdBelow_LeavesY(t *testing.T) {
	tree := snapTestTree()
	// X=9999 is well outside any foothold span
	sp := monster.RestModel{Id: 0, Template: 100100, X: 9999, Y: 80, FH: 0}
	out := snapToGround(tree, sp, groundLookup)
	if out.Y != 80 {
		t.Fatalf("no foothold below: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_FhZero_TemplateLookupErr_LeavesY(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 0}
	out := snapToGround(tree, sp, errLookup)
	if out.Y != 80 {
		t.Fatalf("template lookup err: Y=%d, want 80 (unchanged)", out.Y)
	}
}

func TestSnapToGround_Idempotent(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Id: 0, Template: 100100, X: 0, Y: 80, FH: 10}
	once := snapToGround(tree, sp, groundLookup)
	twice := snapToGround(tree, once, groundLookup)
	if once.Y != twice.Y {
		t.Fatalf("idempotency broken: once=%d, twice=%d", once.Y, twice.Y)
	}
}
