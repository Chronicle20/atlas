package image

import (
	"atlas-wz-extractor/wz/property"
	"testing"
)

func TestFindSubPresent(t *testing.T) {
	props := []property.Property{
		property.NewInt("x", 10),
		property.NewSub("info", []property.Property{property.NewInt("y", 20)}),
		property.NewString("name", "test"),
	}
	result := findSub(props, "info")
	if result == nil {
		t.Fatal("findSub returned nil, want SubProperty")
	}
	if result.Name() != "info" {
		t.Errorf("Name() = %q, want %q", result.Name(), "info")
	}
}

func TestFindSubMissing(t *testing.T) {
	props := []property.Property{
		property.NewInt("x", 10),
		property.NewString("name", "test"),
	}
	result := findSub(props, "info")
	if result != nil {
		t.Errorf("findSub = %v, want nil", result)
	}
}

func TestFindSubEmpty(t *testing.T) {
	result := findSub(nil, "info")
	if result != nil {
		t.Errorf("findSub(nil) = %v, want nil", result)
	}
}

func TestFindSubCanvasPresent(t *testing.T) {
	props := []property.Property{
		property.NewInt("x", 10),
		property.NewCanvas("icon", 32, 32, 2, 0, 0, nil),
		property.NewCanvas("iconRaw", 16, 16, 1, 0, 0, nil),
	}
	result := findSubCanvas(props, "icon")
	if result == nil {
		t.Fatal("findSubCanvas returned nil, want CanvasProperty")
	}
	if result.Name() != "icon" {
		t.Errorf("Name() = %q, want %q", result.Name(), "icon")
	}
}

func TestFindSubCanvasMissing(t *testing.T) {
	props := []property.Property{
		property.NewInt("x", 10),
		property.NewCanvas("iconRaw", 16, 16, 1, 0, 0, nil),
	}
	result := findSubCanvas(props, "icon")
	if result != nil {
		t.Errorf("findSubCanvas = %v, want nil", result)
	}
}

func TestFindFirstCanvasDirect(t *testing.T) {
	props := []property.Property{
		property.NewInt("delay", 100),
		property.NewCanvas("0", 32, 32, 2, 0, 0, nil),
	}
	result := findFirstCanvas(props)
	if result == nil {
		t.Fatal("findFirstCanvas returned nil")
	}
	if result.Name() != "0" {
		t.Errorf("Name() = %q, want %q", result.Name(), "0")
	}
}

func TestFindFirstCanvasInZeroSub(t *testing.T) {
	props := []property.Property{
		property.NewInt("delay", 100),
		property.NewSub("0", []property.Property{
			property.NewCanvas("frame", 32, 32, 2, 0, 0, nil),
		}),
	}
	result := findFirstCanvas(props)
	if result == nil {
		t.Fatal("findFirstCanvas returned nil")
	}
	if result.Name() != "frame" {
		t.Errorf("Name() = %q, want %q", result.Name(), "frame")
	}
}

func TestFindFirstCanvasEmpty(t *testing.T) {
	props := []property.Property{
		property.NewInt("delay", 100),
		property.NewString("name", "test"),
	}
	result := findFirstCanvas(props)
	if result != nil {
		t.Errorf("findFirstCanvas = %v, want nil", result)
	}
}

func TestFindStandCanvasStandPresent(t *testing.T) {
	props := []property.Property{
		property.NewSub("stand", []property.Property{
			property.NewCanvas("0", 32, 32, 2, 0, 0, nil),
		}),
		property.NewSub("move", []property.Property{
			property.NewCanvas("0", 16, 16, 1, 0, 0, nil),
		}),
	}
	result := findStandCanvas(props)
	if result == nil {
		t.Fatal("findStandCanvas returned nil")
	}
	if result.Width() != 32 {
		t.Errorf("Width() = %d, want 32 (from stand)", result.Width())
	}
}

func TestFindStandCanvasFallbackToMove(t *testing.T) {
	props := []property.Property{
		property.NewSub("stand", []property.Property{
			property.NewInt("delay", 100),
		}),
		property.NewSub("move", []property.Property{
			property.NewCanvas("0", 16, 16, 1, 0, 0, nil),
		}),
	}
	result := findStandCanvas(props)
	if result == nil {
		t.Fatal("findStandCanvas returned nil")
	}
	if result.Width() != 16 {
		t.Errorf("Width() = %d, want 16 (from move)", result.Width())
	}
}

func TestFindStandCanvasFallbackToAnySub(t *testing.T) {
	props := []property.Property{
		property.NewSub("attack", []property.Property{
			property.NewCanvas("0", 64, 64, 2, 0, 0, nil),
		}),
	}
	result := findStandCanvas(props)
	if result == nil {
		t.Fatal("findStandCanvas returned nil")
	}
	if result.Width() != 64 {
		t.Errorf("Width() = %d, want 64 (from attack fallback)", result.Width())
	}
}

func TestFindStandCanvasNone(t *testing.T) {
	props := []property.Property{
		property.NewInt("x", 10),
	}
	result := findStandCanvas(props)
	if result != nil {
		t.Errorf("findStandCanvas = %v, want nil", result)
	}
}

func TestFindReactorCanvasPresent(t *testing.T) {
	props := []property.Property{
		property.NewSub("0", []property.Property{
			property.NewCanvas("0", 32, 32, 2, 0, 0, nil),
		}),
	}
	result := findReactorCanvas(props)
	if result == nil {
		t.Fatal("findReactorCanvas returned nil")
	}
}

func TestFindReactorCanvasMissing(t *testing.T) {
	props := []property.Property{
		property.NewSub("1", []property.Property{
			property.NewCanvas("0", 32, 32, 2, 0, 0, nil),
		}),
	}
	result := findReactorCanvas(props)
	if result != nil {
		t.Errorf("findReactorCanvas = %v, want nil", result)
	}
}

func TestFindInfoIconPresent(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewCanvas("icon", 32, 32, 2, 0, 0, nil),
			property.NewCanvas("iconRaw", 32, 32, 2, 0, 0, nil),
		}),
	}
	result := findInfoIcon(props)
	if result == nil {
		t.Fatal("findInfoIcon returned nil")
	}
	if result.Name() != "icon" {
		t.Errorf("Name() = %q, want %q", result.Name(), "icon")
	}
}

func TestFindInfoIconNoInfo(t *testing.T) {
	props := []property.Property{
		property.NewSub("stand", []property.Property{
			property.NewCanvas("icon", 32, 32, 2, 0, 0, nil),
		}),
	}
	result := findInfoIcon(props)
	if result != nil {
		t.Errorf("findInfoIcon = %v, want nil", result)
	}
}

func TestFindInfoIconNoIcon(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewInt("price", 100),
		}),
	}
	result := findInfoIcon(props)
	if result != nil {
		t.Errorf("findInfoIcon = %v, want nil", result)
	}
}
