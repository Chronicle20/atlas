package cash

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// karmaTestXML mirrors the shipped v83 layout for the scissors themselves
// (05520000), plus a sibling carrying tradeAvailable/tradeBlock only.
const karmaTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0552.img">
  <imgdir name="05520000">
    <imgdir name="info">
      <int name="tradeBlock" value="1"/>
      <int name="tradeAvailable" value="1"/>
      <int name="karma" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05520001">
    <imgdir name="info">
      <int name="cash" value="1"/>
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
	if res[0].Id != 5520000 {
		t.Fatalf("Id = %d, want 5520000", res[0].Id)
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

// TestReaderKarmaPresent pins the scissors' own info/karma type.
func TestReaderKarmaPresent(t *testing.T) {
	l, _ := test.NewNullLogger()

	res, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Karma != 1 {
		t.Fatalf("Karma = %d, want 1", res[0].Karma)
	}
}

// TestReaderKarmaAbsentDefaultsToZero pins the real v83 shape: the shipped
// cash corpus carries no `karma` node at all (only `cash`), so Karma must
// default to 0 rather than requiring a classification filter.
func TestReaderKarmaAbsentDefaultsToZero(t *testing.T) {
	l, _ := test.NewNullLogger()

	res, err := Read(l)(xml.FromByteArrayProvider([]byte(karmaTestXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if res[1].Karma != 0 {
		t.Fatalf("Karma = %d, want 0", res[1].Karma)
	}
}
