package _map

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// objectTestXML mirrors the shape of Map.wz/Map/Map1/103000800.img: obj
// entries live under numbered layers, so a parser must scan every layer. The
// layer 4 entry named "gate" with l2=1 is verbatim from that map.
const objectTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="103000800.img">
  <imgdir name="info"><int name="version" value="10"/></imgdir>
  <imgdir name="0">
    <imgdir name="obj">
      <imgdir name="1"><string name="oS" value="acc1"/><string name="l0" value="grassySoil"/><string name="l1" value="nature"/><string name="l2" value="6"/><int name="x" value="121"/><int name="y" value="204"/><int name="z" value="0"/><int name="f" value="0"/><int name="zM" value="0"/></imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="4">
    <imgdir name="obj">
      <imgdir name="34"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="gate"/><string name="l2" value="1"/><int name="x" value="715"/><int name="y" value="34"/><int name="z" value="5"/><int name="f" value="0"/><int name="zM" value="0"/><string name="name" value="gate"/></imgdir>
      <imgdir name="35"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="barricade"/><string name="l2" value="on"/><int name="x" value="120"/><int name="y" value="60"/><int name="z" value="5"/><int name="f" value="0"/><int name="zM" value="0"/><string name="name" value="barricade"/></imgdir>
      <imgdir name="36"><string name="oS" value="effect"/><string name="l0" value="quest"/><string name="l1" value="lever"/><int name="x" value="220"/><int name="y" value="60"/><int name="z" value="5"/><int name="f" value="0"/><int name="zM" value="0"/><string name="name" value="lever"/></imgdir>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestGetObjects(t *testing.T) {
	n, err := xml.FromByteArrayProvider([]byte(objectTestXML))()
	if err != nil {
		t.Fatal(err)
	}
	l, _ := test.NewNullLogger()

	os := getObjects(l, n)
	if len(os) != 3 {
		t.Fatalf("len(os) != 3, got %d", len(os))
	}

	// The unnamed layer 0 entry is not addressable by SetObjectState and is skipped.
	if os[0].Name != "gate" {
		t.Fatalf("os[0].Name != \"gate\", got %q", os[0].Name)
	}
	if os[0].State != 1 {
		t.Fatalf("os[0].State != 1, got %d", os[0].State)
	}

	// A non-numeric l2 falls back to state 0 rather than failing the parse.
	if os[1].Name != "barricade" {
		t.Fatalf("os[1].Name != \"barricade\", got %q", os[1].Name)
	}
	if os[1].State != 0 {
		t.Fatalf("os[1].State for non-numeric l2 != 0, got %d", os[1].State)
	}

	// An absent l2 also falls back to state 0.
	if os[2].Name != "lever" {
		t.Fatalf("os[2].Name != \"lever\", got %q", os[2].Name)
	}
	if os[2].State != 0 {
		t.Fatalf("os[2].State for absent l2 != 0, got %d", os[2].State)
	}
}

func TestGetObjectsWithoutObjNode(t *testing.T) {
	n, err := xml.FromByteArrayProvider([]byte(reactorTestXML))()
	if err != nil {
		t.Fatal(err)
	}
	l, _ := test.NewNullLogger()

	os := getObjects(l, n)
	if len(os) != 0 {
		t.Fatalf("len(os) != 0, got %d", len(os))
	}
}
