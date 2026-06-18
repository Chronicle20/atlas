package map_

import "testing"

// TestGroundBelowSnapsToFootholdSurfaceMinus1 pins the Cosmic getGroundBelow
// parity using the real geometry from map 240011000 at x=760: upper ledge fh58
// flat at y=-478 (x 725..805), lower ledge fh71 flat at y=422 (x 675..765). A
// caster standing on fh58 (760, -478) must anchor the door 1px above the
// surface (-479), NOT on the lower ledge.
func TestGroundBelowSnapsToFootholdSurfaceMinus1(t *testing.T) {
	m := Model{footholds: map[uint32]Foothold{
		58: {Id: 58, FirstX: 725, FirstY: -478, SecondX: 805, SecondY: -478},
		71: {Id: 71, FirstX: 675, FirstY: 422, SecondX: 765, SecondY: 422},
	}}
	gx, gy, ok := m.GroundBelow(760, -478)
	if !ok || gx != 760 || gy != -479 {
		t.Fatalf("GroundBelow(760,-478) = (%d,%d,%v), want (760,-479,true)", gx, gy, ok)
	}
}

// TestGroundBelowPicksNearestFootholdBelow: with two stacked ledges, a caster on
// the upper one must snap to the upper ledge, not fall through to the lower.
func TestGroundBelowPicksNearestFootholdBelow(t *testing.T) {
	m := Model{footholds: map[uint32]Foothold{
		1: {Id: 1, FirstX: 250, FirstY: -200, SecondX: 350, SecondY: -200},
		2: {Id: 2, FirstX: 250, FirstY: 100, SecondX: 350, SecondY: 100},
	}}
	if _, gy, ok := m.GroundBelow(300, -200); !ok || gy != -201 {
		t.Fatalf("GroundBelow snapped to (%d,%v), want -201,true (nearest ledge below)", gy, ok)
	}
}

// TestGroundBelowIgnoresWalls: a wall (FirstX==SecondX) at the caster's x must
// be skipped in favor of the floor below.
func TestGroundBelowIgnoresWalls(t *testing.T) {
	m := Model{footholds: map[uint32]Foothold{
		1: {Id: 1, FirstX: 300, FirstY: -300, SecondX: 300, SecondY: 0}, // wall
		2: {Id: 2, FirstX: 250, FirstY: 50, SecondX: 350, SecondY: 50},  // floor
	}}
	if _, gy, ok := m.GroundBelow(300, -100); !ok || gy != 49 {
		t.Fatalf("GroundBelow over a wall = (%d,%v), want 49,true", gy, ok)
	}
}

// TestGroundBelowMissReturnsNotOk: an x with no foothold below returns ok=false
// so the caller keeps the raw caster position (a cast is never blocked).
func TestGroundBelowMissReturnsNotOk(t *testing.T) {
	m := Model{footholds: map[uint32]Foothold{
		1: {Id: 1, FirstX: 0, FirstY: 0, SecondX: 100, SecondY: 0},
	}}
	if _, _, ok := m.GroundBelow(500, 0); ok {
		t.Fatalf("expected ok=false for an x with no foothold below")
	}
}
