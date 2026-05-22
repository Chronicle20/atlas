package mapimage

import (
	"image"
	"image/draw"
)

// BackgroundType is the tile-mode flag from a `back/i/type` property.
// Verified from WzComparerR2's BackItem.GetBackTileMode.
type BackgroundType int

const (
	BackgroundNormal   BackgroundType = 0
	BackgroundHTile    BackgroundType = 1
	BackgroundVTile    BackgroundType = 2
	BackgroundBothTile BackgroundType = 3
	BackgroundHScroll  BackgroundType = 4 // collapsed to HTile for static
	BackgroundVScroll  BackgroundType = 5 // collapsed to VTile for static
	BackgroundBothH    BackgroundType = 6 // collapsed to BothTile
	BackgroundBothV    BackgroundType = 7 // collapsed to BothTile
)

// Horizontal reports whether the background tiles along X.
func (b BackgroundType) Horizontal() bool {
	switch b {
	case BackgroundHTile, BackgroundBothTile, BackgroundHScroll, BackgroundBothH, BackgroundBothV:
		return true
	}
	return false
}

// Vertical reports whether the background tiles along Y.
func (b BackgroundType) Vertical() bool {
	switch b {
	case BackgroundVTile, BackgroundBothTile, BackgroundVScroll, BackgroundBothH, BackgroundBothV:
		return true
	}
	return false
}

// drawBackground tiles `src` over the output canvas per the background type.
// Parallax (rx/ry) is intentionally collapsed for the static composite render.
// The caller passes `src` already mirrored when b.f != 0.
func drawBackground(canvas *image.RGBA, src *image.NRGBA, s *sprite, b backEntry, world WorldBounds) {
	stepX := b.cx
	if stepX <= 0 {
		stepX = s.w
	}
	stepY := b.cy
	if stepY <= 0 {
		stepY = s.h
	}
	baseX := b.x - s.ox - world.X
	baseY := b.y - s.oy - world.Y

	typ := BackgroundType(b.typ)
	horiz := typ.Horizontal()
	vert := typ.Vertical()

	var xs, ys []int
	if horiz && stepX > 0 {
		start := baseX
		for start > 0 {
			start -= stepX
		}
		for x := start; x < world.W; x += stepX {
			xs = append(xs, x)
		}
	} else {
		xs = []int{baseX}
	}
	if vert && stepY > 0 {
		start := baseY
		for start > 0 {
			start -= stepY
		}
		for y := start; y < world.H; y += stepY {
			ys = append(ys, y)
		}
	} else {
		ys = []int{baseY}
	}

	for _, y := range ys {
		for _, x := range xs {
			dr := image.Rect(x, y, x+s.w, y+s.h)
			draw.Draw(canvas, dr, src, image.Point{}, draw.Over)
		}
	}
}
