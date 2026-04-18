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
		t.Fatal("expected error for no VR*, no miniMap, no content")
	}
}

func TestResolveBoundsFromFootholds(t *testing.T) {
	root := []property.Property{
		property.NewSub("foothold", []property.Property{
			property.NewSub("0", []property.Property{
				property.NewSub("1", []property.Property{
					property.NewSub("1", []property.Property{
						property.NewInt("x1", 100),
						property.NewInt("y1", -50),
						property.NewInt("x2", 500),
						property.NewInt("y2", -50),
					}),
					property.NewSub("2", []property.Property{
						property.NewInt("x1", 500),
						property.NewInt("y1", -50),
						property.NewInt("x2", 800),
						property.NewInt("y2", 100),
					}),
				}),
			}),
		}),
	}
	b, err := resolveBounds(nil, root)
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	// x range [100, 800] padded by 400 each side → [-300, 1200], W=1500.
	// y range [-50, 100] padded → [-450, 500], H=950.
	if b.X != -300 || b.W != 1500 {
		t.Errorf("x=%d w=%d, want -300/1500", b.X, b.W)
	}
	if b.Y != -450 || b.H != 950 {
		t.Errorf("y=%d h=%d, want -450/950", b.Y, b.H)
	}
}

func TestResolveBoundsFromTileObj(t *testing.T) {
	root := []property.Property{
		property.NewSub("0", []property.Property{
			property.NewSub("tile", []property.Property{
				property.NewSub("0", []property.Property{
					property.NewInt("x", 0),
					property.NewInt("y", 0),
				}),
				property.NewSub("1", []property.Property{
					property.NewInt("x", 1000),
					property.NewInt("y", 500),
				}),
			}),
			property.NewSub("obj", []property.Property{
				property.NewSub("0", []property.Property{
					property.NewInt("x", -200),
					property.NewInt("y", -100),
				}),
			}),
		}),
	}
	b, err := resolveBounds(nil, root)
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	// x in [-200, 1000] padded → [-600, 1400]; y in [-100, 500] padded → [-500, 900].
	if b.W != 2000 || b.H != 1400 {
		t.Errorf("W=%d H=%d, want 2000/1400", b.W, b.H)
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
