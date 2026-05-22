package mapimage

import (
	"image"
	"image/draw"
)

// mirrorX returns a horizontally flipped copy of `src`.
func mirrorX(src *image.NRGBA) *image.NRGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcX := w - 1 - x
			di := dst.PixOffset(x, y)
			si := src.PixOffset(srcX, y)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// blit draws a sprite at anchor (ex, ey) in world coords. The sprite's `origin`
// maps to (ex, ey), so the top-left blit is (ex - ox - world.X, ey - oy - world.Y).
func blit(canvas *image.RGBA, src *image.NRGBA, ex, ey, ox, oy int, world WorldBounds, w, h int) {
	dx := ex - ox - world.X
	dy := ey - oy - world.Y
	dr := image.Rect(dx, dy, dx+w, dy+h)
	draw.Draw(canvas, dr, src, image.Point{}, draw.Over)
}
