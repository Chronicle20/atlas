package etc

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// karmaTestXML mirrors the shipped v83 layout for an etc item id in this
// package's range (4000000).
const karmaTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0400.img">
  <imgdir name="04000000">
    <imgdir name="info">
      <int name="tradeBlock" value="1"/>
      <int name="tradeAvailable" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="04000001">
    <imgdir name="info">
      <int name="tradeBlock" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReaderTradeAvailablePresent(t *testing.T) {
	l, _ := test.NewNullLogger()

	res, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, want 2", len(res))
	}
	if res[0].Id != 4000000 {
		t.Fatalf("Id = %d, want 4000000", res[0].Id)
	}
	if res[0].TradeAvailable != 1 {
		t.Fatalf("TradeAvailable = %d, want 1", res[0].TradeAvailable)
	}
}

func TestReaderTradeAvailableAbsentDefaultsToZero(t *testing.T) {
	l, _ := test.NewNullLogger()

	res, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if res[1].TradeAvailable != 0 {
		t.Fatalf("TradeAvailable = %d, want 0", res[1].TradeAvailable)
	}
}
