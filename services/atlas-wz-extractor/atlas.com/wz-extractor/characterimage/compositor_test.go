package characterimage

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeSyntheticBody creates a 4x4 colored body sprite under the assets root.
func writeSyntheticBody(t *testing.T, root string) string {
	t.Helper()
	templateId := "00002000"
	frameDir := filepath.Join(root, "character-parts", templateId, "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	pngFile, err := os.Create(filepath.Join(frameDir, "body.png"))
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer pngFile.Close()
	if err := png.Encode(pngFile, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frameDir, "body.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{"neck":{"x":0,"y":-3}},"z":"body","group":"skin"}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "character-parts", templateId, "info.json"),
		[]byte(`{"islot":"Bd","vslot":"Bd","cash":0}`), 0o644); err != nil {
		t.Fatalf("write info: %v", err)
	}
	return templateId
}

func writeSyntheticMaps(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "character-meta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zmap.json"),
		[]byte(`["body","arm","head","hairOverHead"]`), 0o644); err != nil {
		t.Fatalf("zmap: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smap.json"),
		[]byte(`{}`), 0o644); err != nil {
		t.Fatalf("smap: %v", err)
	}
}

func TestCompositeBareBody(t *testing.T) {
	root := t.TempDir()
	writeSyntheticMaps(t, root)
	writeSyntheticBody(t, root)

	c := NewCompositor()
	res, err := c.Composite(CompositeRequest{
		AssetsRoot: root,
		Skin:       0,
		Stance:     "stand1",
		Frame:      0,
		Resize:     1,
		Equipment:  map[int]int{},
	})
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if res.Image.Bounds().Dx() != CanvasWidth || res.Image.Bounds().Dy() != CanvasHeight {
		t.Fatalf("dims = %v", res.Image.Bounds())
	}
	if res.EquippedSlotCount != 0 {
		t.Fatalf("expected 0 equipped, got %d", res.EquippedSlotCount)
	}
	// Body origin (2,3) must land at canvas (48, 120), so the body sprite
	// occupies x:[46..49], y:[117..120].
	checkColored(t, res.Image, 46, 117)
	checkColored(t, res.Image, 49, 120)
	// Outside the sprite, pixels are transparent.
	if a := res.Image.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("pixel (0,0) alpha = %d, want 0", a)
	}
}

func checkColored(t *testing.T, img *image.RGBA, x, y int) {
	t.Helper()
	c := img.RGBAAt(x, y)
	if c.A == 0 || c.R == 0 {
		t.Fatalf("pixel (%d,%d) = %+v — expected body color", x, y, c)
	}
}
