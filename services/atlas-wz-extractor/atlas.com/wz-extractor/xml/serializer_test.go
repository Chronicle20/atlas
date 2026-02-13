package xml

import (
	"atlas-wz-extractor/wz/property"
	"testing"
)

func TestPropertyToElementNull(t *testing.T) {
	p := property.NewNull("testNull")
	e := propertyToElement(p)
	if e.XMLName.Local != "null" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "null")
	}
	if e.Name != "testNull" {
		t.Errorf("Name = %q, want %q", e.Name, "testNull")
	}
}

func TestPropertyToElementShort(t *testing.T) {
	p := property.NewShort("hp", 100)
	e := propertyToElement(p)
	if e.XMLName.Local != "short" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "short")
	}
	if e.Name != "hp" {
		t.Errorf("Name = %q, want %q", e.Name, "hp")
	}
	if e.Value != "100" {
		t.Errorf("Value = %q, want %q", e.Value, "100")
	}
}

func TestPropertyToElementInt(t *testing.T) {
	p := property.NewInt("mapId", 100000000)
	e := propertyToElement(p)
	if e.XMLName.Local != "int" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "int")
	}
	if e.Value != "100000000" {
		t.Errorf("Value = %q, want %q", e.Value, "100000000")
	}
}

func TestPropertyToElementLong(t *testing.T) {
	p := property.NewLong("exp", 9999999999)
	e := propertyToElement(p)
	if e.XMLName.Local != "long" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "long")
	}
	if e.Value != "9999999999" {
		t.Errorf("Value = %q, want %q", e.Value, "9999999999")
	}
}

func TestPropertyToElementFloat(t *testing.T) {
	p := property.NewFloat("speed", 1.5)
	e := propertyToElement(p)
	if e.XMLName.Local != "float" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "float")
	}
	if e.Value != "1.5" {
		t.Errorf("Value = %q, want %q", e.Value, "1.5")
	}
}

func TestPropertyToElementDouble(t *testing.T) {
	p := property.NewDouble("rate", 0.0)
	e := propertyToElement(p)
	if e.XMLName.Local != "double" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "double")
	}
	if e.Value != "0.0" {
		t.Errorf("Value = %q, want %q", e.Value, "0.0")
	}
}

func TestPropertyToElementString(t *testing.T) {
	p := property.NewString("name", "Mushroom")
	e := propertyToElement(p)
	if e.XMLName.Local != "string" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "string")
	}
	if e.Value != "Mushroom" {
		t.Errorf("Value = %q, want %q", e.Value, "Mushroom")
	}
}

func TestPropertyToElementSub(t *testing.T) {
	children := []property.Property{
		property.NewInt("x", 10),
		property.NewInt("y", 20),
	}
	p := property.NewSub("info", children)
	e := propertyToElement(p)
	if e.XMLName.Local != "imgdir" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "imgdir")
	}
	if e.Name != "info" {
		t.Errorf("Name = %q, want %q", e.Name, "info")
	}
	if len(e.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(e.Children))
	}
	if e.Children[0].XMLName.Local != "int" || e.Children[0].Value != "10" {
		t.Errorf("Children[0] = %+v, want int x=10", e.Children[0])
	}
}

func TestPropertyToElementCanvas(t *testing.T) {
	p := property.NewCanvas("icon", 32, 32, 1, 0, 0, nil)
	e := propertyToElement(p)
	if e.XMLName.Local != "canvas" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "canvas")
	}
	if e.Width != "32" || e.Height != "32" {
		t.Errorf("Width=%q Height=%q, want 32x32", e.Width, e.Height)
	}
}

func TestPropertyToElementVector(t *testing.T) {
	p := property.NewVector("origin", -10, 25)
	e := propertyToElement(p)
	if e.XMLName.Local != "vector" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "vector")
	}
	if e.X != "-10" || e.Y != "25" {
		t.Errorf("X=%q Y=%q, want -10 25", e.X, e.Y)
	}
}

func TestPropertyToElementUOL(t *testing.T) {
	p := property.NewUOL("link", "../stand/0")
	e := propertyToElement(p)
	if e.XMLName.Local != "uol" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "uol")
	}
	if e.Value != "../stand/0" {
		t.Errorf("Value = %q, want %q", e.Value, "../stand/0")
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{1.5, "1.5"},
		{-3.14, "-3.14"},
		{100, "100.0"},
	}
	for _, tt := range tests {
		got := formatFloat(tt.input)
		if got != tt.want {
			t.Errorf("formatFloat(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPropertyToElementConvex(t *testing.T) {
	children := []property.Property{
		property.NewVector("0", 0, 0),
		property.NewVector("1", 10, 20),
	}
	p := property.NewConvex("foothold", children)
	e := propertyToElement(p)
	if e.XMLName.Local != "extended" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "extended")
	}
	if e.Name != "foothold" {
		t.Errorf("Name = %q, want %q", e.Name, "foothold")
	}
	if len(e.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(e.Children))
	}
	if e.Children[0].XMLName.Local != "vector" {
		t.Errorf("Children[0].XMLName.Local = %q, want %q", e.Children[0].XMLName.Local, "vector")
	}
}

func TestPropertyToElementSound(t *testing.T) {
	p := property.NewSound("bgm")
	e := propertyToElement(p)
	if e.XMLName.Local != "sound" {
		t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, "sound")
	}
	if e.Name != "bgm" {
		t.Errorf("Name = %q, want %q", e.Name, "bgm")
	}
}

func TestPropertiesToElementsEmpty(t *testing.T) {
	result := propertiesToElements(nil)
	if result != nil {
		t.Errorf("propertiesToElements(nil) = %v, want nil", result)
	}

	result = propertiesToElements([]property.Property{})
	if result != nil {
		t.Errorf("propertiesToElements([]) = %v, want nil", result)
	}
}

func TestPropertiesToElementsNonEmpty(t *testing.T) {
	props := []property.Property{
		property.NewNull("empty"),
		property.NewInt("id", 42),
		property.NewString("name", "test"),
	}
	result := propertiesToElements(props)
	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}
	if result[0].XMLName.Local != "null" {
		t.Errorf("result[0].XMLName.Local = %q, want %q", result[0].XMLName.Local, "null")
	}
	if result[1].XMLName.Local != "int" {
		t.Errorf("result[1].XMLName.Local = %q, want %q", result[1].XMLName.Local, "int")
	}
	if result[2].XMLName.Local != "string" {
		t.Errorf("result[2].XMLName.Local = %q, want %q", result[2].XMLName.Local, "string")
	}
}
