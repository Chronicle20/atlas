package mapimage

import (
	"atlas-wz-extractor/wz/property"
	"testing"
)

func TestResolveBoundsVR(t *testing.T) {
	info := []property.Property{
		property.NewInt("VRLeft", -100),
		property.NewInt("VRRight", 200),
		property.NewInt("VRTop", -50),
		property.NewInt("VRBottom", 150),
	}
	root := []property.Property{property.NewSub("info", info)}

	b, err := resolveBounds(info, root)
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	if b.X != -100 || b.Y != -50 || b.W != 300 || b.H != 200 {
		t.Errorf("got %+v, want {X:-100 Y:-50 W:300 H:200}", b)
	}
}

func TestResolveBoundsMiniMap(t *testing.T) {
	info := []property.Property{}
	root := []property.Property{
		property.NewSub("miniMap", []property.Property{
			property.NewInt("centerX", 1000),
			property.NewInt("centerY", 250),
			property.NewInt("width", 7444),
			property.NewInt("height", 1391),
		}),
	}
	b, err := resolveBounds(info, root)
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	if b.X != -1000 || b.Y != -250 || b.W != 7444 || b.H != 1391 {
		t.Errorf("got %+v", b)
	}
}

func TestResolveBoundsNone(t *testing.T) {
	info := []property.Property{}
	root := []property.Property{}
	if _, err := resolveBounds(info, root); err == nil {
		t.Fatal("expected error for no VR* and no miniMap")
	}
}

func TestResolveBoundsVRPreferredOverMiniMap(t *testing.T) {
	info := []property.Property{
		property.NewInt("VRLeft", 0),
		property.NewInt("VRRight", 100),
		property.NewInt("VRTop", 0),
		property.NewInt("VRBottom", 100),
	}
	root := []property.Property{
		property.NewSub("info", info),
		property.NewSub("miniMap", []property.Property{
			property.NewInt("width", 9999),
			property.NewInt("height", 9999),
		}),
	}
	b, err := resolveBounds(info, root)
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	if b.W != 100 || b.H != 100 {
		t.Errorf("VR* should win; got %+v", b)
	}
}
