package mapimage

import (
	"atlas-wz-extractor/wz/property"
	"fmt"
)

// WorldBounds is the world-space rect a map renders into.
// Mirrors the atlas-data reader's precedence: VRLeft/VRRight/VRTop/VRBottom
// first, then miniMap.width/height with -centerX/-centerY origin.
type WorldBounds struct {
	X, Y, W, H int
}

// resolveBounds returns the world-space rect for a map.
func resolveBounds(info, root []property.Property) (WorldBounds, error) {
	vrL := intVal(info, "VRLeft", 0)
	vrR := intVal(info, "VRRight", 0)
	vrT := intVal(info, "VRTop", 0)
	vrB := intVal(info, "VRBottom", 0)
	if vrL != vrR && vrT != vrB {
		return WorldBounds{X: vrL, Y: vrT, W: vrR - vrL, H: vrB - vrT}, nil
	}
	mm := findSub(root, "miniMap")
	if mm != nil {
		cx := intVal(mm.Children(), "centerX", 0)
		cy := intVal(mm.Children(), "centerY", 0)
		w := intVal(mm.Children(), "width", 0)
		h := intVal(mm.Children(), "height", 0)
		if w > 0 && h > 0 {
			return WorldBounds{X: -cx, Y: -cy, W: w, H: h}, nil
		}
	}
	return WorldBounds{}, fmt.Errorf("no bounds (no VR*, no miniMap)")
}
