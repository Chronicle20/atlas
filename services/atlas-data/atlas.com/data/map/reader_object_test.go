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

	// The unnamed layer 0 entry is not addressable by SetObjectState and is
	// skipped, so the fixture's three named entries land at indexes 0-2.
	os := getObjects(l, n)
	if len(os) != 3 {
		t.Fatalf("len(os) != 3, got %d", len(os))
	}

	tests := []struct {
		name      string
		index     int
		wantName  string
		wantState uint32
	}{
		{"declared l2 is parsed as the default state", 0, "gate", 1},
		{"non-numeric l2 falls back to state 0", 1, "barricade", 0},
		{"absent l2 falls back to state 0", 2, "lever", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if os[tt.index].Name != tt.wantName {
				t.Fatalf("os[%d].Name != %q, got %q", tt.index, tt.wantName, os[tt.index].Name)
			}
			if os[tt.index].State != tt.wantState {
				t.Fatalf("os[%d].State != %d, got %d", tt.index, tt.wantState, os[tt.index].State)
			}
		})
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
