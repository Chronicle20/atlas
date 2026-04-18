package image

import (
	"atlas-wz-extractor/wz/property"
	"testing"
)

func TestFindMinimapCanvasPresent(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{property.NewInt("version", 1)}),
		property.NewSub("miniMap", []property.Property{
			property.NewCanvas("canvas", 100, 50, 2, 1024, 512, nil),
			property.NewInt("width", 7444),
			property.NewInt("height", 1391),
			property.NewInt("centerX", 1000),
			property.NewInt("centerY", 250),
			property.NewInt("mag", 4),
		}),
	}
	cp := findMinimapCanvas(props)
	if cp == nil {
		t.Fatal("findMinimapCanvas returned nil, want CanvasProperty")
	}
	if cp.Width() != 100 || cp.Height() != 50 {
		t.Errorf("canvas dims = %dx%d, want 100x50", cp.Width(), cp.Height())
	}
}

func TestFindMinimapCanvasMissingSub(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{property.NewInt("version", 1)}),
	}
	if cp := findMinimapCanvas(props); cp != nil {
		t.Errorf("findMinimapCanvas = %v, want nil for missing miniMap", cp)
	}
}

func TestFindMinimapCanvasMissingCanvas(t *testing.T) {
	props := []property.Property{
		property.NewSub("miniMap", []property.Property{
			property.NewInt("width", 7444),
			property.NewInt("height", 1391),
		}),
	}
	if cp := findMinimapCanvas(props); cp != nil {
		t.Errorf("findMinimapCanvas = %v, want nil when canvas child missing", cp)
	}
}
