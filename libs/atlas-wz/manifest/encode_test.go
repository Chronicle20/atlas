package manifest

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshalSortsMapKeys(t *testing.T) {
	m := Manifest{
		Version: 1, ID: 1040002, PartClass: "coat",
		Sheet: Size{256, 256},
		Sprites: []Sprite{{
			Stance: "stand1", Frame: 0, Part: "arm",
			Rect: Rect{0, 0, 32, 48}, Origin: Point{16, 32},
			Anchors: map[string]Point{
				"neck":  {16, 8},
				"navel": {16, 32},
				"armor": {1, 2},
				"head":  {3, 4},
			},
			Z: "armBelowHead",
		}},
	}
	a, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		b, err := Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("Marshal not deterministic at iteration %d:\n a=%s\n b=%s", i, a, b)
		}
	}
}

// TestMarshalVslotOmitemptyAbsentByDefault locks in the requirement that
// adding the Vslot field does not change manifests where vslot is unset:
// existing per-id manifests remain byte-identical to their pre-Vslot encoding.
func TestMarshalVslotOmitemptyAbsentByDefault(t *testing.T) {
	m := Manifest{
		Version: 1, ID: 1, PartClass: "coat",
		Sheet:   Size{Width: 4, Height: 4},
		Sprites: []Sprite{},
	}
	b, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "vslot") {
		t.Fatalf("vslot key should be omitted when empty: %s", b)
	}
}

// TestMarshalVslotPresentDeterministic exercises Marshal across many runs with
// vslot populated to guard against any future map-iteration noise around the
// new field.
func TestMarshalVslotPresentDeterministic(t *testing.T) {
	m := Manifest{
		Version: 1, ID: 1002000, PartClass: "Cap", Vslot: "CpHnHd",
		Sheet:   Size{Width: 8, Height: 8},
		Sprites: []Sprite{},
	}
	a, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(a), `"vslot":"CpHnHd"`) {
		t.Fatalf("vslot value missing from encoded form: %s", a)
	}
	for i := 0; i < 32; i++ {
		b, err := Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("Marshal not deterministic at iteration %d", i)
		}
	}
}
