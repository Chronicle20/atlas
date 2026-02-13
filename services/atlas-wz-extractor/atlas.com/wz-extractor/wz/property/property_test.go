package property

import "testing"

func TestNullProperty(t *testing.T) {
	p := NewNull("empty")
	if p.Name() != "empty" {
		t.Errorf("Name() = %q, want %q", p.Name(), "empty")
	}
	if p.Type() != TypeNull {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeNull)
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestShortProperty(t *testing.T) {
	p := NewShort("hp", 30000)
	if p.Name() != "hp" {
		t.Errorf("Name() = %q, want %q", p.Name(), "hp")
	}
	if p.Type() != TypeShort {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeShort)
	}
	if p.Value() != 30000 {
		t.Errorf("Value() = %d, want 30000", p.Value())
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestShortPropertyNegative(t *testing.T) {
	p := NewShort("damage", -100)
	if p.Value() != -100 {
		t.Errorf("Value() = %d, want -100", p.Value())
	}
}

func TestIntProperty(t *testing.T) {
	p := NewInt("mapId", 100000000)
	if p.Name() != "mapId" {
		t.Errorf("Name() = %q, want %q", p.Name(), "mapId")
	}
	if p.Type() != TypeInt {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeInt)
	}
	if p.Value() != 100000000 {
		t.Errorf("Value() = %d, want 100000000", p.Value())
	}
}

func TestLongProperty(t *testing.T) {
	p := NewLong("exp", 9999999999)
	if p.Name() != "exp" {
		t.Errorf("Name() = %q, want %q", p.Name(), "exp")
	}
	if p.Type() != TypeLong {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeLong)
	}
	if p.Value() != 9999999999 {
		t.Errorf("Value() = %d, want 9999999999", p.Value())
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestFloatProperty(t *testing.T) {
	p := NewFloat("speed", 1.5)
	if p.Name() != "speed" {
		t.Errorf("Name() = %q, want %q", p.Name(), "speed")
	}
	if p.Type() != TypeFloat {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeFloat)
	}
	if p.Value() != 1.5 {
		t.Errorf("Value() = %f, want 1.5", p.Value())
	}
}

func TestDoubleProperty(t *testing.T) {
	p := NewDouble("rate", 0.123456789)
	if p.Name() != "rate" {
		t.Errorf("Name() = %q, want %q", p.Name(), "rate")
	}
	if p.Type() != TypeDouble {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeDouble)
	}
	if p.Value() != 0.123456789 {
		t.Errorf("Value() = %f, want 0.123456789", p.Value())
	}
}

func TestStringProperty(t *testing.T) {
	p := NewString("name", "Mushroom")
	if p.Name() != "name" {
		t.Errorf("Name() = %q, want %q", p.Name(), "name")
	}
	if p.Type() != TypeString {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeString)
	}
	if p.Value() != "Mushroom" {
		t.Errorf("Value() = %q, want %q", p.Value(), "Mushroom")
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestSubProperty(t *testing.T) {
	children := []Property{
		NewInt("x", 10),
		NewInt("y", 20),
	}
	p := NewSub("info", children)
	if p.Name() != "info" {
		t.Errorf("Name() = %q, want %q", p.Name(), "info")
	}
	if p.Type() != TypeSub {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeSub)
	}
	if len(p.Children()) != 2 {
		t.Fatalf("len(Children()) = %d, want 2", len(p.Children()))
	}
	if p.Children()[0].Name() != "x" {
		t.Errorf("Children()[0].Name() = %q, want %q", p.Children()[0].Name(), "x")
	}
}

func TestSubPropertyEmpty(t *testing.T) {
	p := NewSub("empty", nil)
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestCanvasProperty(t *testing.T) {
	children := []Property{NewVector("origin", -5, -10)}
	p := NewCanvas("icon", 32, 32, 2, 1024, 512, children)
	if p.Name() != "icon" {
		t.Errorf("Name() = %q, want %q", p.Name(), "icon")
	}
	if p.Type() != TypeCanvas {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeCanvas)
	}
	if p.Width() != 32 {
		t.Errorf("Width() = %d, want 32", p.Width())
	}
	if p.Height() != 32 {
		t.Errorf("Height() = %d, want 32", p.Height())
	}
	if p.Format() != 2 {
		t.Errorf("Format() = %d, want 2", p.Format())
	}
	if p.DataOffset() != 1024 {
		t.Errorf("DataOffset() = %d, want 1024", p.DataOffset())
	}
	if p.DataSize() != 512 {
		t.Errorf("DataSize() = %d, want 512", p.DataSize())
	}
	if len(p.Children()) != 1 {
		t.Fatalf("len(Children()) = %d, want 1", len(p.Children()))
	}
}

func TestVectorProperty(t *testing.T) {
	p := NewVector("origin", -10, 25)
	if p.Name() != "origin" {
		t.Errorf("Name() = %q, want %q", p.Name(), "origin")
	}
	if p.Type() != TypeVector {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeVector)
	}
	if p.X() != -10 {
		t.Errorf("X() = %d, want -10", p.X())
	}
	if p.Y() != 25 {
		t.Errorf("Y() = %d, want 25", p.Y())
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestConvexProperty(t *testing.T) {
	children := []Property{
		NewVector("0", 0, 0),
		NewVector("1", 10, 10),
	}
	p := NewConvex("foothold", children)
	if p.Name() != "foothold" {
		t.Errorf("Name() = %q, want %q", p.Name(), "foothold")
	}
	if p.Type() != TypeConvex {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeConvex)
	}
	if len(p.Children()) != 2 {
		t.Fatalf("len(Children()) = %d, want 2", len(p.Children()))
	}
}

func TestSoundProperty(t *testing.T) {
	p := NewSound("bgm")
	if p.Name() != "bgm" {
		t.Errorf("Name() = %q, want %q", p.Name(), "bgm")
	}
	if p.Type() != TypeSound {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeSound)
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}

func TestUOLProperty(t *testing.T) {
	p := NewUOL("link", "../stand/0")
	if p.Name() != "link" {
		t.Errorf("Name() = %q, want %q", p.Name(), "link")
	}
	if p.Type() != TypeUOL {
		t.Errorf("Type() = %d, want %d", p.Type(), TypeUOL)
	}
	if p.Value() != "../stand/0" {
		t.Errorf("Value() = %q, want %q", p.Value(), "../stand/0")
	}
	if p.Children() != nil {
		t.Errorf("Children() = %v, want nil", p.Children())
	}
}
