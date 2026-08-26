package wztoxml

import (
	stdxml "encoding/xml"
	"os"
	"path/filepath"
	"testing"

	atlasxml "atlas-data/xml"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/wzxml"
)

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
	root := wzxml.Element{
		XMLName:  stdxml.Name{Local: "imgdir"},
		Name:     "0100000.img",
		Children: wzxml.PropertiesToElements(props),
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
