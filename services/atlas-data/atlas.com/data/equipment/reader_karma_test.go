package equipment

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// karmaTestXML mirrors the shipped v83 layout verbatim: 01002357 is the Zakum
// Helmet (Character.wz/Cap/01002357.img), which carries
// <int name="tradeAvailable" value="1"/> beside tradeBlock under info.
const karmaTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01002357.img">
  <imgdir name="info">
    <int name="only" value="1"/>
    <int name="tradeBlock" value="1"/>
    <int name="tradeAvailable" value="1"/>
  </imgdir>
</imgdir>
`

// karmaAbsentTestXML is the same node with no tradeAvailable child.
const karmaAbsentTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01002358.img">
  <imgdir name="info">
    <int name="tradeBlock" value="1"/>
  </imgdir>
</imgdir>
`

// TestReaderTradeAvailablePresent is the FR-3.5 real-item case: the Zakum Helmet.
func TestReaderTradeAvailablePresent(t *testing.T) {
	l, _ := test.NewNullLogger()

	rm, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if rm.Id != 1002357 {
		t.Fatalf("Id = %d, want 1002357", rm.Id)
	}
	if rm.TradeAvailable != 1 {
		t.Fatalf("TradeAvailable = %d, want 1", rm.TradeAvailable)
	}
}

func TestReaderTradeAvailableAbsentDefaultsToZero(t *testing.T) {
	l, _ := test.NewNullLogger()

	rm, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaAbsentTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if rm.TradeAvailable != 0 {
		t.Fatalf("TradeAvailable = %d, want 0", rm.TradeAvailable)
	}
}
