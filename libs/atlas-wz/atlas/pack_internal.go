package atlas

import (
	"image"
	"image/draw"

	"github.com/Chronicle20/atlas/libs/atlas-wz/manifest"
)

// freeRect is a free rectangle in the bin.
type freeRect struct{ x, y, w, h int }

// tryPack attempts to lay out sprites into a size×size sheet. Returns
// (sheet, manifest, true) on success.
func tryPack(sorted []Input, size int) (image.Image, manifest.Manifest, bool) {
	free := []freeRect{{0, 0, size, size}}
	placements := make([]image.Rectangle, len(sorted))

	for i, sp := range sorted {
		w, h := sp.Img.Bounds().Dx(), sp.Img.Bounds().Dy()
		bestIdx := -1
		bestShort := 1 << 30
		bestLong := 1 << 30
		for j, fr := range free {
			if fr.w < w || fr.h < h {
				continue
			}
			leftoverW := fr.w - w
			leftoverH := fr.h - h
			shortSide := minInt(leftoverW, leftoverH)
			longSide := maxInt(leftoverW, leftoverH)
			if shortSide < bestShort || (shortSide == bestShort && longSide < bestLong) {
				bestShort = shortSide
				bestLong = longSide
				bestIdx = j
			}
		}
		if bestIdx == -1 {
			return nil, manifest.Manifest{}, false
		}
		chosen := free[bestIdx]
		placement := image.Rect(chosen.x, chosen.y, chosen.x+w, chosen.y+h)
		placements[i] = placement
		free = splitFree(free, bestIdx, placement)
		free = pruneFree(free)
	}

	sheet := image.NewNRGBA(image.Rect(0, 0, size, size))
	sprites := make([]manifest.Sprite, len(sorted))
	for i, sp := range sorted {
		draw.Draw(sheet, placements[i], sp.Img, sp.Img.Bounds().Min, draw.Src)
		anchors := make(map[string]manifest.Point, len(sp.Anchors))
		for k, p := range sp.Anchors {
			anchors[k] = manifest.Point{X: p.X, Y: p.Y}
		}
		sprites[i] = manifest.Sprite{
			// Stance/Frame/Part are derived by the caller from sp.Name;
			// pack only sets geometric fields.
			Part: sp.Name,
			Rect: manifest.Rect{
				X: placements[i].Min.X, Y: placements[i].Min.Y,
				W: placements[i].Dx(), H: placements[i].Dy(),
			},
			Origin:  manifest.Point{X: sp.Origin.X, Y: sp.Origin.Y},
			Anchors: anchors,
			Z:       manifest.ZOrder(sp.Z),
		}
	}
	return sheet, manifest.Manifest{
		Version: manifest.SchemaVersion,
		Sheet:   manifest.Size{Width: size, Height: size},
		Sprites: sprites,
	}, true
}

func splitFree(free []freeRect, idx int, used image.Rectangle) []freeRect {
	target := free[idx]
	// Remove target, append up to four child rects.
	out := free[:idx:idx]
	out = append(out, free[idx+1:]...)
	// Right of used
	if used.Max.X < target.x+target.w {
		out = append(out, freeRect{used.Max.X, target.y, target.x + target.w - used.Max.X, target.h})
	}
	// Below used
	if used.Max.Y < target.y+target.h {
		out = append(out, freeRect{target.x, used.Max.Y, target.w, target.y + target.h - used.Max.Y})
	}
	return out
}

func pruneFree(free []freeRect) []freeRect {
	// Remove rectangles fully contained inside another.
	out := free[:0]
	for i, a := range free {
		contained := false
		for j, b := range free {
			if i == j {
				continue
			}
			if a.x >= b.x && a.y >= b.y && a.x+a.w <= b.x+b.w && a.y+a.h <= b.y+b.h {
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, a)
		}
	}
	return out
}

func minInt(a, b int) int { if a < b { return a }; return b }
func maxInt(a, b int) int { if a > b { return a }; return b }
