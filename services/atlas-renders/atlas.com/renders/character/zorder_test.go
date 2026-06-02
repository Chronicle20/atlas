package character

import (
	"sort"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
)

func TestZIndex(t *testing.T) {
	// Front-to-back order (index 0 = most frontward), mirroring the shape of
	// Base.wz/zmap.img.
	zmap := []string{"capOverHair", "hairOverHead", "face", "head", "weapon", "body", "capeBelowBody", "backWeapon"}

	cases := []struct {
		layer string
		want  int
	}{
		{"capOverHair", 0},
		{"face", 2},
		{"weapon", 4},
		{"backWeapon", 7},
		{"HairOverHead", 1}, // case-insensitive
		{"WEAPON", 4},
		{"notInZmap", len(zmap)}, // unknown sorts to the back
	}
	for _, c := range cases {
		if got := zIndex(zmap, c.layer); got != c.want {
			t.Errorf("zIndex(%q) = %d, want %d", c.layer, got, c.want)
		}
	}

	// Empty zmap: every layer collapses to 0 so the stable sort preserves
	// insertion order (the graceful fallback when the sidecar is missing).
	if got := zIndex(nil, "weapon"); got != 0 {
		t.Errorf("zIndex(nil, weapon) = %d, want 0", got)
	}
}

// TestPlacementSortByZmap exercises the exact comparator Composite uses:
// sort.SliceStable descending by zIndex(zmap, sprite.Z). zmap is front-to-back, so
// after the sort the back-most layer must be first (drawn first) and the
// front-most layer last (drawn last, on top).
func TestPlacementSortByZmap(t *testing.T) {
	zmap := []string{
		"weaponOverHand", // 0 frontmost
		"hairOverHead",   // 1
		"face",           // 2
		"head",           // 3
		"arm",            // 4
		"body",           // 5
		"weapon",         // 6
		"capeBelowBody",  // 7 backmost
	}

	// Insertion order deliberately scrambled relative to render order.
	placements := []placement{
		{sprite: manifest.Sprite{Z: "body"}},
		{sprite: manifest.Sprite{Z: "weaponOverHand"}},
		{sprite: manifest.Sprite{Z: "capeBelowBody"}},
		{sprite: manifest.Sprite{Z: "head"}},
		{sprite: manifest.Sprite{Z: "weapon"}},
		{sprite: manifest.Sprite{Z: "face"}},
		{sprite: manifest.Sprite{Z: "arm"}},
		{sprite: manifest.Sprite{Z: "hairOverHead"}},
	}

	sort.SliceStable(placements, func(i, j int) bool {
		return zIndex(zmap, string(placements[i].sprite.Z)) > zIndex(zmap, string(placements[j].sprite.Z))
	})

	// Expected draw order = zmap reversed (backmost first, frontmost last).
	want := []string{
		"capeBelowBody", "weapon", "body", "arm", "head", "face", "hairOverHead", "weaponOverHand",
	}
	for i, p := range placements {
		if string(p.sprite.Z) != want[i] {
			got := make([]string, len(placements))
			for k, pp := range placements {
				got[k] = string(pp.sprite.Z)
			}
			t.Fatalf("draw order = %v, want %v", got, want)
		}
	}
}

// TestPlacementSortUnknownLayerToBack confirms a part whose layer is absent
// from zmap draws first (back-most), never on top of a mapped part.
func TestPlacementSortUnknownLayerToBack(t *testing.T) {
	zmap := []string{"face", "head", "body"}
	placements := []placement{
		{sprite: manifest.Sprite{Z: "face"}},
		{sprite: manifest.Sprite{Z: "mysteryLayer"}}, // unknown -> len(zmap)=3, backmost
		{sprite: manifest.Sprite{Z: "body"}},
	}
	sort.SliceStable(placements, func(i, j int) bool {
		return zIndex(zmap, string(placements[i].sprite.Z)) > zIndex(zmap, string(placements[j].sprite.Z))
	})
	if string(placements[0].sprite.Z) != "mysteryLayer" {
		t.Fatalf("unknown layer should draw first (backmost); got first=%q", string(placements[0].sprite.Z))
	}
	if string(placements[len(placements)-1].sprite.Z) != "face" {
		t.Fatalf("frontmost mapped layer should draw last; got last=%q", string(placements[len(placements)-1].sprite.Z))
	}
}
