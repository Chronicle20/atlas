package wztoxml

import (
	stdxml "encoding/xml"
	"os"
	"path/filepath"
	"testing"

	atlasxml "atlas-data/xml"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

func TestPropertyToElementShort(t *testing.T) {
	p := property.NewShort("hp", 100)
	e := propertyToElement(p)
	if e.XMLName.Local != "short" || e.Name != "hp" || e.Value != "100" {
		t.Errorf("unexpected element: %+v", e)
	}
}

func TestPropertyToElementInt(t *testing.T) {
	p := property.NewInt("mapId", 100000000)
	e := propertyToElement(p)
	if e.XMLName.Local != "int" || e.Value != "100000000" {
		t.Errorf("unexpected element: %+v", e)
	}
}

func TestPropertyToElementSub(t *testing.T) {
	children := []property.Property{
		property.NewInt("x", 10),
		property.NewInt("y", 20),
	}
	p := property.NewSub("info", children)
	e := propertyToElement(p)
	if e.XMLName.Local != "imgdir" || e.Name != "info" {
		t.Errorf("unexpected element: %+v", e)
	}
	if len(e.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(e.Children))
	}
	if e.Children[0].XMLName.Local != "int" || e.Children[0].Value != "10" {
		t.Errorf("Children[0] = %+v", e.Children[0])
	}
}

func TestPropertyToElementVector(t *testing.T) {
	p := property.NewVector("origin", -10, 25)
	e := propertyToElement(p)
	if e.XMLName.Local != "vector" || e.X != "-10" || e.Y != "25" {
		t.Errorf("unexpected element: %+v", e)
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{{0, "0.0"}, {1, "1.0"}, {1.5, "1.5"}, {-3.14, "-3.14"}, {100, "100.0"}}
	for _, tc := range tests {
		if got := formatFloat(tc.in); got != tc.want {
			t.Errorf("formatFloat(%v)=%q want %q", tc.in, got, tc.want)
		}
	}
}

// TestRoundTripImage verifies an in-memory wz.Image can be serialized to XML
// and then re-parsed by atlas-data/xml into a Node with the expected shape.
func TestRoundTripImage(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewInt("id", 100000),
			property.NewString("name", "Mushroom"),
		}),
	}
	// We can't easily build a wz.Image directly without exporting more APIs;
	// instead test the inner serializer by writing the XML manually and
	// verifying it parses back into atlas-data/xml.Node.
	root := xmlElement{
		XMLName:  stdxml.Name{Local: "imgdir"},
		Name:     "0100000.img",
		Children: propertiesToElements(props),
	}
	path := filepath.Join(dir, "0100000.img.xml")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(stdxml.Header); err != nil {
		t.Fatal(err)
	}
	enc := stdxml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Parse via atlas-data xml reader.
	n, err := atlasxml.FromPathProvider(path)()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n.Name != "0100000.img" {
		t.Errorf("Name=%q", n.Name)
	}
	info, err := n.ChildByName("info")
	if err != nil {
		t.Fatalf("ChildByName: %v", err)
	}
	if info.GetIntegerWithDefault("id", -1) != 100000 {
		t.Errorf("id mismatch")
	}
	if info.GetString("name", "") != "Mushroom" {
		t.Errorf("name mismatch")
	}
}
