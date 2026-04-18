package mapimage

import (
	"atlas-wz-extractor/wz/property"
	"testing"
)

func TestNormalizeMapId(t *testing.T) {
	cases := map[string]string{
		"100000000": "100000000",
		"000000000": "0",
		"010000000": "10000000",
		"0":         "0",
	}
	for in, want := range cases {
		if got := normalizeMapId(in); got != want {
			t.Errorf("normalizeMapId(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmptyMapTrue(t *testing.T) {
	root := []property.Property{
		property.NewSub("info", nil),
	}
	if !emptyMap(root) {
		t.Error("expected emptyMap=true for map with no back[] and no layers")
	}
}

func TestEmptyMapFalseWithBack(t *testing.T) {
	root := []property.Property{
		property.NewSub("back", []property.Property{
			property.NewSub("0", []property.Property{property.NewString("bS", "grassySoil")}),
		}),
	}
	if emptyMap(root) {
		t.Error("expected emptyMap=false when back[] has entries")
	}
}

func TestEmptyMapFalseWithLayer(t *testing.T) {
	root := []property.Property{
		property.NewSub("0", []property.Property{
			property.NewSub("tile", []property.Property{
				property.NewSub("0", nil),
			}),
		}),
	}
	if emptyMap(root) {
		t.Error("expected emptyMap=false when layer 0 has tiles")
	}
}
