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
	templateId := "2000"
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

func writeSyntheticHat(t *testing.T, root string, hatId int) {
	t.Helper()
	tmpl := "10000" // matches request equipment id 10000 (stripped form per normalizeId)
	if hatId != 0 {
		// caller chose an explicit id — currently unused
		_ = hatId
	}
	frameDir := filepath.Join(root, "character-parts", tmpl, "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			img.SetRGBA(x, y, color.RGBA{B: 200, G: 50, A: 255})
		}
	}
	f, _ := os.Create(filepath.Join(frameDir, "default.png"))
	defer f.Close()
	_ = png.Encode(f, img)
	_ = os.WriteFile(filepath.Join(frameDir, "default.json"),
		[]byte(`{"origin":{"x":3,"y":3},"map":{"neck":{"x":0,"y":0}},"z":"cap"}`), 0o644)
	_ = os.WriteFile(filepath.Join(root, "character-parts", tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644)
}

// writeSyntheticDefaultHair creates a synthetic hair sprite at
// <root>/character-parts/30030/default/0/hair.{png,json}.
// This mirrors equipment that has no animated stances and only exists under
// the `default` stance directory.
func writeSyntheticDefaultHair(t *testing.T, root string) {
	t.Helper()
	templateId := "30030"
	frameDir := filepath.Join(root, "character-parts", templateId, "default", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 50, G: 200, B: 50, A: 255})
		}
	}
	f, err := os.Create(filepath.Join(frameDir, "hair.png"))
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	// hair anchors at neck joint; origin=(1,1), map.neck=(0,0) so the hair's
	// neck joint coincides with the body's neck canvas point.
	if err := os.WriteFile(filepath.Join(frameDir, "hair.json"),
		[]byte(`{"origin":{"x":1,"y":1},"map":{"neck":{"x":0,"y":0}},"z":"hairOverHead"}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "character-parts", templateId, "info.json"),
		[]byte(`{"islot":"Ha","vslot":"Ha","cash":0}`), 0o644); err != nil {
		t.Fatalf("write info: %v", err)
	}
}

// TestCompositeFallsBackToDefault verifies that when compositing a character
// with a hair template that only has a `default/0` asset directory (not the
// requested `stand1`), the compositor falls back gracefully and the hair pixel
// appears in the output canvas.
func TestCompositeFallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	// zmap: hairOverHead sorts after body.
	if err := os.MkdirAll(filepath.Join(root, "character-meta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(root, "character-meta", "zmap.json"),
		[]byte(`["body","arm","hairOverHead"]`), 0o644)
	_ = os.WriteFile(filepath.Join(root, "character-meta", "smap.json"), []byte(`{}`), 0o644)

	writeSyntheticBody(t, root)
	writeSyntheticDefaultHair(t, root)

	c := NewCompositor()
	res, err := c.Composite(CompositeRequest{
		AssetsRoot: root,
		Skin:       0,
		Hair:       30030,
		Stance:     "stand1",
		Frame:      0,
		Resize:     1,
		Equipment:  map[int]int{},
	})
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}

	// Body origin (2,3) lands at canvas (48,120).  body.neck = (48+0, 120-3) = (48,117).
	// hair origin (1,1), map.neck=(0,0) → anchor = (48-0, 117-0) = (48,117).
	// drawPart places top-left at (anchor.X - origin.X, anchor.Y - origin.Y) = (47,116).
	// The 4x4 hair sprite occupies x:[47..50], y:[116..119].
	hairPixel := res.Image.RGBAAt(47, 116)
	if hairPixel.A == 0 || hairPixel.G == 0 {
		t.Fatalf("expected hair (green) pixel at (47,116), got %+v", hairPixel)
	}
	if res.EquippedSlotCount != 0 {
		t.Fatalf("hair is not counted as equipment slot; expected 0, got %d", res.EquippedSlotCount)
	}
}

// TestResolveTemplateStanceFallback exercises the three cases in
// resolveTemplateStance: present (no fallback), only default, and neither.
func TestResolveTemplateStanceFallback(t *testing.T) {
	root := t.TempDir()

	// Case 1: body sprite at 2000/stand1/0 — should resolve to stand1/0.
	bodyDir := filepath.Join(root, "character-parts", "2000", "stand1", "0")
	if err := os.MkdirAll(bodyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bodyDir, "body.png"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	gotStance, gotFrame, err := resolveTemplateStance(root, "2000", "stand1", 0)
	if err != nil {
		t.Fatalf("case 1: unexpected error: %v", err)
	}
	if gotStance != "stand1" || gotFrame != 0 {
		t.Fatalf("case 1: want stand1/0, got %s/%d", gotStance, gotFrame)
	}

	// Case 2: hair at 30030/default/0 only — should fall back to default/0.
	hairDir := filepath.Join(root, "character-parts", "30030", "default", "0")
	if err := os.MkdirAll(hairDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hairDir, "hair.png"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	gotStance, gotFrame, err = resolveTemplateStance(root, "30030", "stand1", 0)
	if err != nil {
		t.Fatalf("case 2: unexpected error: %v", err)
	}
	if gotStance != "default" || gotFrame != 0 {
		t.Fatalf("case 2: want default/0, got %s/%d", gotStance, gotFrame)
	}

	// Case 3: template 99999 has neither stand1/0 nor default/0 — should error.
	_, _, err = resolveTemplateStance(root, "99999", "stand1", 0)
	if err == nil {
		t.Fatal("case 3: expected error for missing template, got nil")
	}

	// Case 4: weapon at 1452000/stand1/0 only (typical of crossbows / guns /
	// knuckles whose WZ source omits stand2). Two-handed override forces
	// stand2 lookup; we should fall back to stand1 instead of skipping the
	// part outright, otherwise Bowmasters / Marksmen / Buccaneers / Corsairs
	// render without their weapons.
	xbowDir := filepath.Join(root, "character-parts", "1452000", "stand1", "0")
	if err := os.MkdirAll(xbowDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xbowDir, "weapon.png"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gotStance, gotFrame, err = resolveTemplateStance(root, "1452000", "stand2", 0)
	if err != nil {
		t.Fatalf("case 4: unexpected error: %v", err)
	}
	if gotStance != "stand1" || gotFrame != 0 {
		t.Fatalf("case 4: want stand1/0 fallback, got %s/%d", gotStance, gotFrame)
	}
}

func TestCompositeWithHatBlitsAboveBody(t *testing.T) {
	root := t.TempDir()
	// zmap places "cap" above "body".
	if err := os.MkdirAll(filepath.Join(root, "character-meta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(root, "character-meta", "zmap.json"),
		[]byte(`["body","arm","cap"]`), 0o644)
	_ = os.WriteFile(filepath.Join(root, "character-meta", "smap.json"), []byte(`{}`), 0o644)

	writeSyntheticBody(t, root)
	writeSyntheticHat(t, root, 10000)

	c := NewCompositor()
	res, err := c.Composite(CompositeRequest{
		AssetsRoot: root,
		Skin:       0,
		Stance:     "stand1",
		Frame:      0,
		Resize:     1,
		Equipment:  map[int]int{-1: 10000},
	})
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	// The hat sprite should land near the body's neck on canvas. With body
	// origin (2,3) at (48,120), body.neck = (48,117). Hat origin (3,3) over
	// joint neck (0,0) means hat anchor = (45, 114). So pixel (45,114) is hat.
	c1 := res.Image.RGBAAt(45, 114)
	if c1.B == 0 {
		t.Fatalf("hat pixel missing at (45,114): %+v", c1)
	}
}

// TestSolveViaSharedJointMultiStep verifies that a part with only a `brow`
// joint anchors via the head (whose map has `brow`), not via the body
// (whose map only has `neck`). This proves the algorithm walks placed
// parts most-recent-first and picks the closest matching joint owner.
func TestSolveViaSharedJointMultiStep(t *testing.T) {
	bodyAnchor := Anchor{X: 48, Y: 96}
	body := PartMeta{Map: map[string]Vec{"neck": {X: 0, Y: -30}}}
	headAnchor := Anchor{X: 48, Y: 51} // (48,96)+(0,-30)-(0,15)
	head := PartMeta{Map: map[string]Vec{"neck": {X: 0, Y: 15}, "brow": {X: 0, Y: -10}}}
	hair := PartMeta{Map: map[string]Vec{"brow": {X: 0, Y: 0}}}

	placed := []placement{
		{partName: "body", meta: body, anchor: bodyAnchor},
		{partName: "head", meta: head, anchor: headAnchor},
	}
	got, ok := solveViaSharedJoint(placed, hair)
	if !ok {
		t.Fatal("hair should attach via head.brow")
	}
	want := Anchor{X: 48, Y: 41} // (48,51)+(0,-10)-(0,0)
	if got != want {
		t.Fatalf("hair anchor = %+v, want %+v", got, want)
	}
}

// TestSolveViaSharedJointWeaponViaArm verifies that a weapon (with `hand`
// joint) attaches to the arm part — not to body — because arm is the most
// recent placement that exposes `hand`. Body has only `navel`.
func TestSolveViaSharedJointWeaponViaArm(t *testing.T) {
	bodyAnchor := Anchor{X: 50, Y: 100}
	body := PartMeta{Map: map[string]Vec{"navel": {X: -8, Y: -21}}}
	armAnchor := Anchor{X: 45, Y: 80}
	arm := PartMeta{Map: map[string]Vec{
		"navel": {X: -13, Y: -1},
		"hand":  {X: -1, Y: 5},
	}}
	weapon := PartMeta{Map: map[string]Vec{"hand": {X: -20, Y: -2}}}

	placed := []placement{
		{partName: "body", meta: body, anchor: bodyAnchor},
		{partName: "arm", meta: arm, anchor: armAnchor},
	}
	got, ok := solveViaSharedJoint(placed, weapon)
	if !ok {
		t.Fatal("weapon should attach via arm.hand")
	}
	// arm anchor (45,80) + arm.hand (-1,5) - weapon.hand (-20,-2) = (64, 87)
	want := Anchor{X: 64, Y: 87}
	if got != want {
		t.Fatalf("weapon anchor = %+v, want %+v", got, want)
	}
}

// TestSolveViaSharedJointOrphanReturnsFalse verifies that a part with no
// shared joint with any placed part returns ok=false (the caller skips).
func TestSolveViaSharedJointOrphanReturnsFalse(t *testing.T) {
	body := PartMeta{Map: map[string]Vec{"neck": {X: 0, Y: -30}}}
	orphan := PartMeta{Map: map[string]Vec{"weirdJoint": {X: 0, Y: 0}}}
	placed := []placement{
		{partName: "body", meta: body, anchor: Anchor{X: 48, Y: 96}},
	}
	if _, ok := solveViaSharedJoint(placed, orphan); ok {
		t.Fatal("expected orphan part to return ok=false")
	}
}

// TestSolveViaSharedJointEmptyPlaced returns false (no parents to match).
func TestSolveViaSharedJointEmptyPlaced(t *testing.T) {
	part := PartMeta{Map: map[string]Vec{"neck": {X: 0, Y: 0}}}
	if _, ok := solveViaSharedJoint(nil, part); ok {
		t.Fatal("expected empty placed list to return ok=false")
	}
}

// TestHeadTemplateId verifies the head template name calculation: skin 0
// should map to "12000" (after stripping the leading zero from "00012000"),
// matching the on-disk dir for 0001{wzSkin}.img.
func TestHeadTemplateId(t *testing.T) {
	cases := []struct {
		wzSkin int
		want   string
	}{
		{2000, "12000"},
		{2005, "12005"},
		{2009, "12009"},
	}
	for _, c := range cases {
		got := headTemplateId(c.wzSkin)
		if got != c.want {
			t.Errorf("headTemplateId(%d) = %q, want %q", c.wzSkin, got, c.want)
		}
	}
}
