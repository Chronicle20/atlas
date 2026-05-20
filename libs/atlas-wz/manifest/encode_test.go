package manifest

import (
	"bytes"
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
			Z: 1,
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
