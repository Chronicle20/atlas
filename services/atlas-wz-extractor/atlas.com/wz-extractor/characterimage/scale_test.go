package characterimage

import (
	"image"
	"image/color"
	"testing"
)

func TestNearestNeighborUpscale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{G: 255, A: 255})
	src.Set(0, 1, color.RGBA{B: 255, A: 255})
	src.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	got := NearestNeighborUpscale(src, 2)

	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("dims = %v", got.Bounds())
	}
	if r, _, _, _ := got.At(0, 0).RGBA(); r != 0xffff {
		t.Fatalf("(0,0) red expected, got %v", got.At(0, 0))
	}
	if r, _, _, _ := got.At(3, 3).RGBA(); r != 0xffff {
		t.Fatalf("(3,3) white expected, got %v", got.At(3, 3))
	}
}

func TestNearestNeighborUpscaleResize1Identity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	src.Set(1, 1, color.RGBA{A: 255})
	got := NearestNeighborUpscale(src, 1)
	if got.Bounds() != src.Bounds() {
		t.Fatalf("resize=1 should be identity dims, got %v", got.Bounds())
	}
}
