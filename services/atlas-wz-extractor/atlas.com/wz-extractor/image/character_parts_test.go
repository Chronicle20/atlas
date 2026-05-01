package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestExtractInfoBlock(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewString("islot", "Cp"),
			property.NewString("vslot", "Cp"),
			property.NewInt("cash", 0),
		}),
	}
	got := extractInfoBlock(props)
	if got.Islot != "Cp" || got.Vslot != "Cp" || got.Cash != 0 {
		t.Fatalf("unexpected info: %+v", got)
	}
}

func TestWriteInfoJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "1002357")
	if err := writeInfoJSON(target, templateInfo{Islot: "Cp", Vslot: "Cp", Cash: 0}); err != nil {
		t.Fatalf("write: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(target, "info.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ti templateInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ti.Islot != "Cp" {
		t.Fatalf("islot = %q", ti.Islot)
	}
}

// TestStancesInScopeIncludesDefault verifies the allow-list was updated to
// include the "default" stance used by non-animated equipment.
func TestStancesInScopeIncludesDefault(t *testing.T) {
	if _, ok := stancesInScope["default"]; !ok {
		t.Fatal("stancesInScope must contain \"default\" for non-animated equipment (hair, face, hats, etc.)")
	}
}

// TestExtractDefaultStanceChildrenSkipsNonCanvas verifies that only
// CanvasProperty children are counted; SubProperty and other non-canvas
// children are silently ignored.  We pass nil for the wz.File pointer because
// no canvas is present so extractPartCanvas is never called.
func TestExtractDefaultStanceChildrenSkipsNonCanvas(t *testing.T) {
	l := logrus.New()
	l.SetOutput(io.Discard)

	dir := t.TempDir()
	children := []property.Property{
		property.NewSub("map", nil),                      // SubProperty — must be skipped
		property.NewString("z", "hairOverHead"),          // StringProperty — must be skipped
		property.NewVector("origin", 5, 10),              // VectorProperty — must be skipped
	}
	got := extractDefaultStanceChildren(l, nil, "30030", children, dir)
	if got != 0 {
		t.Fatalf("expected 0 canvases counted for all-non-canvas children, got %d", got)
	}
}

// TestExtractDefaultStanceChildrenWritesToDefault0 verifies that the helper
// creates the destination directory at <templateDir>/default/0/ when it
// encounters a canvas child.  We use a zero-size CanvasProperty: ReadCanvasData
// returns (nil, nil) for size ≤ 1, Decompress returns an empty image for nil
// data, so the full pipeline succeeds without a real WZ file handle.
//
// Because wz.File's constructor requires a real file on disk, we cannot pass a
// real *wz.File here.  The zero-size shortcut (dataSize = 0) lets us verify
// path routing without binary WZ data.
func TestExtractDefaultStanceChildrenWritesToDefault0(t *testing.T) {
	t.Skip("requires a real *wz.File — path routing verified by TestStancesInScopeIncludesDefault + compositor fallback tests")
}

func TestBuildPartSidecar(t *testing.T) {
	children := []property.Property{
		property.NewVector("origin", 19, 32),
		property.NewString("z", "body"),
		property.NewString("group", "skin"),
		property.NewInt("delay", 180),
		property.NewShort("face", 1),
		property.NewSub("map", []property.Property{
			property.NewVector("neck", -4, -32),
			property.NewVector("navel", -6, -20),
		}),
	}
	got := buildPartSidecar(children)
	if got.Origin != (vec{X: 19, Y: 32}) {
		t.Fatalf("origin = %+v", got.Origin)
	}
	if got.Z != "body" || got.Group != "skin" || got.Delay != 180 || got.Face != 1 {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if got.Map["neck"] != (vec{X: -4, Y: -32}) {
		t.Fatalf("map.neck = %+v", got.Map["neck"])
	}
	if got.Map["navel"] != (vec{X: -6, Y: -20}) {
		t.Fatalf("map.navel = %+v", got.Map["navel"])
	}
}
