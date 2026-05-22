package mapimage

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
)

// TestNewIndexCollectsMapsFromMapWzLayout pins the directory layout the
// real Map.wz exposes: top-level subdirectories Map/Tile/Back/Obj, with
// per-id .img files nested under Map/Map0..Map9. NewIndex MUST surface
// every per-map image in idx.Maps(); a regression here (e.g. a name-match
// drift on the "Map" case in the switch, or skipping the nested level)
// drops every map asset to scanned=0 — exactly what PR-544 saw with the
// real WZ before the imgID fix (42975bb1b).
//
// Using in-memory NewDirectory keeps this test deterministic and
// fixture-free.
func TestNewIndexCollectsMapsFromMapWzLayout(t *testing.T) {
	map0 := wz.NewDirectory("Map0", nil, []*wz.Image{
		wz.NewParsedImage("0100000", nil),
		wz.NewParsedImage("100000000", nil),
	})
	map1 := wz.NewDirectory("Map1", nil, []*wz.Image{
		wz.NewParsedImage("100010000", nil),
	})
	// Sibling Tile/Back/Obj dirs use their own image sets — must NOT bleed
	// into idx.Maps() (the Map field).
	tile := wz.NewDirectory("Tile", nil, []*wz.Image{wz.NewParsedImage("woodMarble", nil)})
	back := wz.NewDirectory("Back", nil, []*wz.Image{wz.NewParsedImage("login", nil)})
	obj := wz.NewDirectory("Obj", nil, []*wz.Image{wz.NewParsedImage("login", nil)})
	mapDir := wz.NewDirectory("Map", []*wz.Directory{map0, map1}, nil)

	root := wz.NewDirectory("Map", []*wz.Directory{mapDir, tile, back, obj}, nil)
	file := wz.NewFileWithRoot("Map", root)

	idx := NewIndex(file)
	maps := idx.Maps()
	if len(maps) != 3 {
		t.Fatalf("idx.Maps() len = %d, want 3 (got names: %v)", len(maps), keysOf(maps))
	}
	for _, want := range []string{"0100000", "100000000", "100010000"} {
		if _, ok := maps[want]; !ok {
			t.Errorf("idx.Maps() missing %q", want)
		}
	}
	// Sanity: Tile/Back/Obj entries must not appear in the Map lookup.
	for _, leaked := range []string{"woodMarble", "login"} {
		if _, ok := maps[leaked]; ok {
			t.Errorf("idx.Maps() leaked %q from Tile/Back/Obj subdir", leaked)
		}
	}
}

func keysOf(m map[string]*wz.Image) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
