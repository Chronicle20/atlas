package characterimage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStanceForcesStand2OnTwoHanded(t *testing.T) {
	// Empty assetsRoot bypasses the on-disk check and preserves the pre-fix
	// behaviour: any IsTwoHanded weapon forces stand2.
	got, override := ResolveStance("", "stand1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm forces stand2: got %q override=%v", got, override)
	}
	got, override = ResolveStance("", "stand1", map[int]int{-11: 1302000})
	if got != "stand1" || override {
		t.Fatalf("sword keeps stand1: got %q override=%v", got, override)
	}
	got, override = ResolveStance("", "walk1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm + walk1: got %q override=%v", got, override)
	}
}

func TestResolveStanceSkipsOverrideWhenWeaponLacksStand2(t *testing.T) {
	// A bow/knuckle/gun-shaped scenario: weapon is two-handed (1452000 = bow)
	// but its on-disk template only has stand1, no stand2. The override must
	// stay disabled so the body doesn't go into a pose the weapon can't fill.
	root := t.TempDir()
	weapon := 1452000
	stand1 := filepath.Join(root, "character-parts", "1452000", "stand1", "0")
	if err := os.MkdirAll(stand1, 0o755); err != nil {
		t.Fatalf("mkdir stand1: %v", err)
	}
	got, override := ResolveStance(root, "stand1", map[int]int{-11: weapon})
	if got != "stand1" || override {
		t.Fatalf("bow without stand2 should keep stand1: got %q override=%v", got, override)
	}
}

func TestResolveStanceAppliesOverrideWhenWeaponHasStand2(t *testing.T) {
	// Two-handed sword scenario: weapon ships stand2 in WZ → override fires.
	root := t.TempDir()
	weapon := 1402072
	stand2 := filepath.Join(root, "character-parts", "1402072", "stand2", "0")
	if err := os.MkdirAll(stand2, 0o755); err != nil {
		t.Fatalf("mkdir stand2: %v", err)
	}
	got, override := ResolveStance(root, "stand1", map[int]int{-11: weapon})
	if got != "stand2" || !override {
		t.Fatalf("2H sword with stand2 should force stand2: got %q override=%v", got, override)
	}
}
