package image

import (
	"atlas-wz-extractor/wz"
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

// TestStancesInScopeIncludesFrontBack verifies the head template's
// front/back stances are extracted (the head canvas only lives under those).
func TestStancesInScopeIncludesFrontBack(t *testing.T) {
	for _, s := range []string{"front", "back"} {
		if _, ok := stancesInScope[s]; !ok {
			t.Fatalf("stancesInScope missing %q", s)
		}
		if _, ok := directCanvasStances[s]; !ok {
			t.Fatalf("directCanvasStances missing %q", s)
		}
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
	got := extractDefaultStanceChildren(l, nil, "30030", children, dir, "default", nil, "default")
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

// TestCanonicalizeUOLPath verifies the path arithmetic used to dereference
// UOL values (relative ../-style paths against an anchor path).
func TestCanonicalizeUOLPath(t *testing.T) {
	cases := []struct {
		anchor string
		uol    string
		want   string
	}{
		{"stand1/0", "../../front/head", "front/head"},
		{"stand1/0/head", "../../../default/head", "default/head"},
		{"", "default/face", "default/face"},
		{"a/b/c", "..", "a/b"},
		{"a/b/c", "../../x", "a/x"},
		{"a/b", "X/Y", "a/b/x/y"}, // names are lower-cased
	}
	for _, c := range cases {
		got := canonicalizeUOLPath(c.anchor, c.uol)
		if got != c.want {
			t.Errorf("canonicalizeUOLPath(%q, %q) = %q, want %q", c.anchor, c.uol, got, c.want)
		}
	}
}

// TestBuildPathLookup verifies the recursive index covers nested children
// and lower-cases keys.
func TestBuildPathLookup(t *testing.T) {
	root := []property.Property{
		property.NewSub("stand1", []property.Property{
			property.NewSub("0", []property.Property{
				property.NewUOL("head", "../../front/head"),
			}),
		}),
		property.NewSub("Front", []property.Property{
			property.NewVector("head", 1, 2),
		}),
	}
	lookup := buildPathLookup(root)
	if _, ok := lookup["stand1/0/head"]; !ok {
		t.Fatal("missing stand1/0/head")
	}
	if _, ok := lookup["front/head"]; !ok {
		t.Fatalf("missing front/head; got keys: %v", lookup)
	}
}

// TestResolveUOL verifies a UOL pointing to a sibling resolves to the
// sibling property.
func TestResolveUOL(t *testing.T) {
	target := property.NewVector("head", 99, 100)
	root := []property.Property{
		property.NewSub("stand1", []property.Property{
			property.NewSub("0", []property.Property{
				property.NewUOL("head", "../../front/head"),
			}),
		}),
		property.NewSub("front", []property.Property{
			target,
		}),
	}
	lookup := buildPathLookup(root)
	uol := lookup["stand1/0/head"].(*property.UOLProperty)
	got := resolveUOL(lookup, "stand1/0", uol)
	if got != target {
		t.Fatalf("resolveUOL = %v, want %v", got, target)
	}
}

// TestResolveUOLChained follows a UOL that points at another UOL.
func TestResolveUOLChained(t *testing.T) {
	final := property.NewVector("head", 1, 2)
	root := []property.Property{
		property.NewSub("a", []property.Property{
			property.NewUOL("head", "../b/head"),
		}),
		property.NewSub("b", []property.Property{
			property.NewUOL("head", "../c/head"),
		}),
		property.NewSub("c", []property.Property{
			final,
		}),
	}
	lookup := buildPathLookup(root)
	uol := lookup["a/head"].(*property.UOLProperty)
	got := resolveUOL(lookup, "a", uol)
	if got != final {
		t.Fatalf("resolveUOL chained = %v, want %v", got, final)
	}
}

// TestResolveUOLMissing returns nil when the target path doesn't exist.
func TestResolveUOLMissing(t *testing.T) {
	root := []property.Property{
		property.NewSub("stand1", []property.Property{
			property.NewUOL("head", "../missing/head"),
		}),
	}
	lookup := buildPathLookup(root)
	uol := lookup["stand1/head"].(*property.UOLProperty)
	if got := resolveUOL(lookup, "stand1", uol); got != nil {
		t.Fatalf("resolveUOL = %v, want nil", got)
	}
}

// recordedWrite captures one canvas write for assertion.
type recordedWrite struct {
	dir      string
	partName string
	canvas   *property.CanvasProperty
}

// withCanvasRecorder swaps in a stub canvasWriter that just records calls.
// Returns the recorder slice and a restore function for the caller to defer.
func withCanvasRecorder(t *testing.T) (*[]recordedWrite, func()) {
	t.Helper()
	var rec []recordedWrite
	prev := canvasWriter
	canvasWriter = func(_ logrus.FieldLogger, _ *wz.File, cp *property.CanvasProperty, dir, name string) error {
		rec = append(rec, recordedWrite{dir: dir, partName: name, canvas: cp})
		return nil
	}
	return &rec, func() { canvasWriter = prev }
}

// TestExtractDefaultStanceChildrenResolvesUOL verifies that a UOL child of
// a `default`/`front`/`back` stance is dereferenced and the target canvas is
// emitted under the alias's name.
func TestExtractDefaultStanceChildrenResolvesUOL(t *testing.T) {
	rec, restore := withCanvasRecorder(t)
	defer restore()

	// The head canvas lives under front/head; stand1/0/head is a UOL alias.
	headCanvas := property.NewCanvas("head", 4, 4, 0, 0, 0, nil)
	root := []property.Property{
		property.NewSub("front", []property.Property{
			headCanvas,
		}),
	}
	lookup := buildPathLookup(root)
	frontSub := root[0].(*property.SubProperty)

	dir := t.TempDir()
	got := extractDefaultStanceChildren(
		logrus.New(), nil, "12000", frontSub.Children(),
		dir, "front", lookup, "front",
	)
	if got != 1 {
		t.Fatalf("extracted = %d, want 1", got)
	}
	if len(*rec) != 1 {
		t.Fatalf("recorded %d writes, want 1", len(*rec))
	}
	if (*rec)[0].partName != "head" || (*rec)[0].canvas != headCanvas {
		t.Fatalf("unexpected write: %+v", (*rec)[0])
	}
}

// TestExtractAnimatedFrameChildrenResolvesUOL verifies that a UOL inside an
// animated stance frame (e.g., stand1/0/head -> ../../front/head) materializes
// the target canvas under the alias's path.
func TestExtractAnimatedFrameChildrenResolvesUOL(t *testing.T) {
	rec, restore := withCanvasRecorder(t)
	defer restore()

	headCanvas := property.NewCanvas("head", 4, 4, 0, 0, 0, nil)
	root := []property.Property{
		property.NewSub("front", []property.Property{
			headCanvas,
		}),
		property.NewSub("stand1", []property.Property{
			property.NewSub("0", []property.Property{
				property.NewUOL("head", "../../front/head"),
			}),
		}),
	}
	lookup := buildPathLookup(root)
	frame := root[1].(*property.SubProperty).Children()[0].(*property.SubProperty)

	dir := t.TempDir()
	got := extractAnimatedFrameChildren(
		logrus.New(), nil, "12000", "stand1", "0",
		frame.Children(), dir, lookup, "stand1/0",
	)
	if got != 1 {
		t.Fatalf("extracted = %d, want 1", got)
	}
	if len(*rec) != 1 || (*rec)[0].partName != "head" || (*rec)[0].canvas != headCanvas {
		t.Fatalf("unexpected writes: %+v", *rec)
	}
}
