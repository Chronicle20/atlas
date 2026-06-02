package charparts

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestExtractZmapPreservesOrder builds a synthetic Base.wz with a zmap.img and
// confirms ExtractZmap returns the child names in WZ declaration order — the
// order IS the render order, so it must not be sorted or reordered.
func TestExtractZmapPreservesOrder(t *testing.T) {
	// Deliberately not alphabetical: render order is front-to-back and bears
	// no relation to lexical order.
	zmapImg := wz.NewParsedImage("zmap.img", []property.Property{
		property.NewString("backWeapon", ""),
		property.NewString("capeBelowBody", ""),
		property.NewString("body", ""),
		property.NewString("weapon", ""),
		property.NewString("hairOverHead", ""),
		property.NewString("capOverHair", ""),
	})
	root := wz.NewDirectory("Base", nil, []*wz.Image{zmapImg})
	f := wz.NewFileWithRoot("Base", root)

	got, err := ExtractZmap(f)
	if err != nil {
		t.Fatalf("ExtractZmap err: %v", err)
	}
	want := []string{"backWeapon", "capeBelowBody", "body", "weapon", "hairOverHead", "capOverHair"}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("zmap[%d] = %q, want %q (order must be preserved)", i, got[i], want[i])
		}
	}
}

// TestExtractZmapMissing returns ErrZmapMissing when Base.wz lacks a zmap.img.
func TestExtractZmapMissing(t *testing.T) {
	root := wz.NewDirectory("Base", nil, []*wz.Image{
		wz.NewParsedImage("smap.img", nil),
	})
	f := wz.NewFileWithRoot("Base", root)

	if _, err := ExtractZmap(f); !errors.Is(err, ErrZmapMissing) {
		t.Fatalf("expected ErrZmapMissing, got %v", err)
	}
}

// TestExtractZmapNilFile guards against caller misuse.
func TestExtractZmapNilFile(t *testing.T) {
	if _, err := ExtractZmap(nil); err == nil {
		t.Fatal("expected error for nil wz.File")
	}
}

// TestMarshalZmapPreservesOrder confirms the JSON encoding keeps order (a JSON
// array, not an object — order is significant) and round-trips.
func TestMarshalZmapPreservesOrder(t *testing.T) {
	z := []string{"backWeapon", "body", "weapon", "hairOverHead"}
	b, err := MarshalZmap(z)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `["backWeapon","body","weapon","hairOverHead"]` {
		t.Fatalf("unexpected marshal: %s", b)
	}
	var rt []string
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatal(err)
	}
	for i := range z {
		if rt[i] != z[i] {
			t.Fatalf("round-trip order mismatch at %d: %q vs %q", i, rt[i], z[i])
		}
	}
}

// TestMarshalZmapNil treats a nil slice as an empty array, producing "[]" so
// the downstream PUT never writes "null".
func TestMarshalZmapNil(t *testing.T) {
	b, err := MarshalZmap(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Errorf("nil zmap marshal = %s, want []", b)
	}
}
