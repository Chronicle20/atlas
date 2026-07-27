package mapr

import (
	"image"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
)

func layerNames() (map[string]image.Image, []maplayout.Layer) {
	byName := map[string]image.Image{
		"layer-0": image.NewRGBA(image.Rect(0, 0, 1, 1)),
		"layer-1": image.NewRGBA(image.Rect(0, 0, 1, 1)),
	}
	layers := []maplayout.Layer{
		{ID: 0, Name: "layer-0"},
		{ID: 1, Name: "layer-1"},
	}
	return byName, layers
}

// TestRenderOrderCharacterZMapFallsBack is the v12 monolithic regression: the
// ZMap holds the character part order (from the parent Data.wz root's zmap),
// none of which name a map layer. renderOrder must ignore it and fall back to
// the layer-declaration order, or the map renders empty.
func TestRenderOrderCharacterZMapFallsBack(t *testing.T) {
	byName, layers := layerNames()
	characterZMap := []string{"hair", "face", "body", "cape", "weapon", "backHair", "Bd"}
	got := renderOrder(characterZMap, layers, byName)
	if len(got) != 2 || got[0] != "layer-0" || got[1] != "layer-1" {
		t.Fatalf("order = %v, want [layer-0 layer-1] (character zmap must be ignored)", got)
	}
}

// TestRenderOrderNilZMap: no ZMap (split Map.wz, v83) → layer-declaration order.
func TestRenderOrderNilZMap(t *testing.T) {
	byName, layers := layerNames()
	got := renderOrder(nil, layers, byName)
	if len(got) != 2 || got[0] != "layer-0" {
		t.Fatalf("order = %v, want [layer-0 layer-1]", got)
	}
}

// TestRenderOrderValidZMap: a ZMap that DOES name real layers is honored, in
// its own order, and unproduced entries are dropped.
func TestRenderOrderValidZMap(t *testing.T) {
	byName, layers := layerNames()
	got := renderOrder([]string{"layer-1", "layer-9", "layer-0"}, layers, byName)
	if len(got) != 2 || got[0] != "layer-1" || got[1] != "layer-0" {
		t.Fatalf("order = %v, want [layer-1 layer-0]", got)
	}
}
