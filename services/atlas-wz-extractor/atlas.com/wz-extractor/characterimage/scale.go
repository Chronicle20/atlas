package characterimage

import "image"

// NearestNeighborUpscale produces an integer-multiple upscale of src using
// nearest-neighbor sampling so each source pixel becomes an N×N block. The
// character renderer's resize parameter is in {1,2,3,4}; resize=1 returns a
// fresh copy.
func NearestNeighborUpscale(src *image.RGBA, resize int) *image.RGBA {
	if resize < 1 {
		resize = 1
	}
	sb := src.Bounds()
	w, h := sb.Dx()*resize, sb.Dy()*resize
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y/resize
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x/resize
			out.Set(x, y, src.At(sx, sy))
		}
	}
	return out
}
