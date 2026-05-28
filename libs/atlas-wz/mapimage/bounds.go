package mapimage

import (
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"fmt"
	"strconv"
)

// WorldBounds is the world-space rect a map renders into.
// Resolution precedence (mirrors atlas-data's reader for the first two):
//
//  1. VRLeft/VRRight/VRTop/VRBottom — explicit world rect.
//  2. miniMap.width/height with -centerX/-centerY origin.
//  3. Content-derived: bounding box of footholds + per-layer tile/obj anchors,
//     padded to capture sprite extents beyond anchor points.
type WorldBounds struct {
	X, Y, W, H int
}

// contentPadPx is added to each side when bounds are content-derived, so
// sprites whose canvas origin sits well inside the sprite still draw inside
// the rect. Chosen to comfortably cover the largest observed tile/obj origins
// (origin y up to ~300 in e.g. castleWall tile sets).
const contentPadPx = 400

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
	if b, ok := boundsFromContent(root); ok {
		return b, nil
	}
	return WorldBounds{}, fmt.Errorf("no bounds (no VR*, no miniMap, no content)")
}

// boundsFromContent derives a bounding rect from foothold geometry plus
// per-layer tile/obj anchor positions. Returns ok=false when the map has no
// positional content to measure.
func boundsFromContent(root []property.Property) (WorldBounds, bool) {
	const big = int(^uint(0) >> 1) // max int
	minX, minY := big, big
	maxX, maxY := -big, -big
	found := false

	acc := func(x, y int) {
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
		found = true
	}

	// foothold/{group}/{polygon}/{segment}: {x1, y1, x2, y2}
	if fh := findSub(root, "foothold"); fh != nil {
		for _, group := range fh.Children() {
			gsub, ok := group.(*property.SubProperty)
			if !ok {
				continue
			}
			for _, poly := range gsub.Children() {
				psub, ok := poly.(*property.SubProperty)
				if !ok {
					continue
				}
				for _, seg := range psub.Children() {
					ssub, ok := seg.(*property.SubProperty)
					if !ok {
						continue
					}
					ch := ssub.Children()
					acc(intVal(ch, "x1", 0), intVal(ch, "y1", 0))
					acc(intVal(ch, "x2", 0), intVal(ch, "y2", 0))
				}
			}
		}
	}

	// per-layer tile/obj anchors
	for layer := 0; layer < 8; layer++ {
		ls := findSub(root, strconv.Itoa(layer))
		if ls == nil {
			continue
		}
		for _, kind := range []string{"tile", "obj"} {
			sub := findSub(ls.Children(), kind)
			if sub == nil {
				continue
			}
			for _, entry := range sub.Children() {
				esub, ok := entry.(*property.SubProperty)
				if !ok {
					continue
				}
				ch := esub.Children()
				acc(intVal(ch, "x", 0), intVal(ch, "y", 0))
			}
		}
	}

	if !found {
		return WorldBounds{}, false
	}
	minX -= contentPadPx
	minY -= contentPadPx
	maxX += contentPadPx
	maxY += contentPadPx
	return WorldBounds{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}, true
}
