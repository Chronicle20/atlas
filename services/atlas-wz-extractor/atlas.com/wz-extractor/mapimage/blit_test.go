package mapimage

import (
	"image"
	"testing"
)

func TestMirrorX(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	// Pixel (0,0) red, (1,0) green.
	src.Pix = []uint8{
		255, 0, 0, 255,
		0, 255, 0, 255,
	}
	dst := mirrorX(src)
	// Expect flipped: (0,0) green, (1,0) red.
	want := []uint8{
		0, 255, 0, 255,
		255, 0, 0, 255,
	}
	for i, w := range want {
		if dst.Pix[i] != w {
			t.Errorf("pixel byte %d = %d, want %d", i, dst.Pix[i], w)
		}
	}
}

func TestBlitOrigin(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 10, 10))
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	// anchor (5,5) with origin (1,1) in a world rooted at (0,0) places blit at (4,4).
	blit(canvas, src, 5, 5, 1, 1, WorldBounds{X: 0, Y: 0, W: 10, H: 10}, 2, 2)
	// Pixel (4,4) should now be opaque white.
	off := canvas.PixOffset(4, 4)
	if canvas.Pix[off+3] != 255 {
		t.Errorf("expected opaque blit at (4,4), got alpha=%d", canvas.Pix[off+3])
	}
	if canvas.Pix[canvas.PixOffset(3, 3)+3] != 0 {
		t.Errorf("expected no blit at (3,3)")
	}
}

func TestBlitAppliesWorldOrigin(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 10, 10))
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.Pix = []uint8{255, 255, 255, 255}
	// anchor (5,5) with world.X=2, world.Y=2 → image (3,3).
	blit(canvas, src, 5, 5, 0, 0, WorldBounds{X: 2, Y: 2, W: 10, H: 10}, 1, 1)
	if canvas.Pix[canvas.PixOffset(3, 3)+3] != 255 {
		t.Errorf("expected opaque at (3,3)")
	}
}
