package mapimage

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestExtractPortalsDeduplicates pins the F20 fix: two portal entries
// with the same (name, target, x, y) collapse to one. Mirrors the WZ
// shape observed for Henesys 100000000 in the f20 repro log.
func TestExtractPortalsDeduplicates(t *testing.T) {
	mkPortal := func(idx, name string, tm int32) property.Property {
		return property.NewSub(idx, []property.Property{
			property.NewString("pn", name),
			property.NewInt("tm", tm),
			property.NewInt("pt", 2),
			property.NewInt("x", 100),
			property.NewInt("y", 200),
		})
	}
	portalSub := property.NewSub("portal", []property.Property{
		mkPortal("0", "sp", 999999999),
		mkPortal("1", "sp", 999999999), // duplicate (same name, target, x, y)
		mkPortal("2", "east00", 100000001),
	})
	root := []property.Property{portalSub}

	out := extractPortals(root)
	if len(out) != 2 {
		t.Fatalf("got %d portals, want 2 (dedup), entries: %+v", len(out), out)
	}
}
