package maplayout

import (
	"bytes"
	"testing"
)

func TestMarshalDeterministic(t *testing.T) {
	l := Layout{
		Version: 1,
		MapID:   100000000,
		Bounds:  Bounds{Left: -1000, Top: -500, Right: 1000, Bottom: 500},
		Layers: []Layer{
			{ID: 0, Name: "back", Z: 0, Source: "back/0.png"},
			{ID: 1, Name: "tile", Z: 1, Source: "tile/1.png"},
			{ID: 2, Name: "obj", Z: 2, Source: "obj/2.png"},
		},
		Footholds: []Foothold{
			{ID: 1, X1: 0, Y1: 0, X2: 100, Y2: 0, Prev: 0, Next: 2},
			{ID: 2, X1: 100, Y1: 0, X2: 200, Y2: 50, Prev: 1, Next: 3},
			{ID: 3, X1: 200, Y1: 50, X2: 300, Y2: 50, Prev: 2, Next: 0},
		},
		Portals: []Portal{
			{Name: "sp", Type: 0, Target: 999999999, X: 0, Y: 0},
			{Name: "out", Type: 2, Target: 100000001, X: 250, Y: 50},
		},
		NPCs: []NPC{
			{ID: 9000000, X: 50, Y: 0, Foothold: 1},
			{ID: 9000001, X: 150, Y: 25, Foothold: 2},
		},
		ZMap: []string{"back", "tile", "obj"},
	}
	a, err := Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		b, err := Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("Marshal not deterministic at iteration %d:\n a=%s\n b=%s", i, a, b)
		}
	}
}
