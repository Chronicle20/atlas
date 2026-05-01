package characterimage

import "testing"

func TestResolveStanceForcesStand2OnTwoHanded(t *testing.T) {
	// Polearm 1442024 is two-handed. Sword 1302000 is one-handed.
	got, override := ResolveStance("stand1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm forces stand2: got %q override=%v", got, override)
	}
	got, override = ResolveStance("stand1", map[int]int{-11: 1302000})
	if got != "stand1" || override {
		t.Fatalf("sword keeps stand1: got %q override=%v", got, override)
	}
	// walk1 must also be overridden when two-handed weapon equipped.
	got, override = ResolveStance("walk1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm + walk1: got %q override=%v", got, override)
	}
}
