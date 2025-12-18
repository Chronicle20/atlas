package etc

import (
	"atlas-data/xml"
	"github.com/sirupsen/logrus/hooks/test"
	"testing"
)

func Identity[M any](m M) M {
	return m
}

func TestReader(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	res, err := rms()
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 3 {
		t.Fatalf("Expected 3 etc items, got %d", len(res))
	}

	// Test the first etc item
	etc1 := res[0]
	if etc1.Id != 4000000 {
		t.Fatalf("Expected ID 4000000, got %d", etc1.Id)
	}
	if etc1.Price != 3 {
		t.Fatalf("Expected price 3, got %d", etc1.Price)
	}
	if etc1.SlotMax != 200 {
		t.Fatalf("Expected slotMax 200, got %d", etc1.SlotMax)
	}

	// Test the second etc item
	etc2 := res[1]
	if etc2.Id != 4000001 {
		t.Fatalf("Expected ID 4000001, got %d", etc2.Id)
	}
	if etc2.Price != 4 {
		t.Fatalf("Expected price 4, got %d", etc2.Price)
	}
	if etc2.SlotMax != 200 {
		t.Fatalf("Expected slotMax 200, got %d", etc2.SlotMax)
	}

	// Test the third etc item
	etc3 := res[2]
	if etc3.Id != 4001434 {
		t.Fatalf("Expected ID 4001434, got %d", etc3.Id)
	}
	if etc3.Price != 1 {
		t.Fatalf("Expected price 1, got %d", etc3.Price)
	}
	if etc3.SlotMax != 100 {
		t.Fatalf("Expected slotMax 100, got %d", etc3.SlotMax)
	}
}

const testXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0400.img">
  <imgdir name="04000000">
    <imgdir name="info">
      <canvas name="icon" width="26" height="27">
        <vector name="origin" x="-4" y="27"/>
      </canvas>
      <canvas name="iconRaw" width="25" height="25">
        <vector name="origin" x="-4" y="27"/>
      </canvas>
      <int name="price" value="3"/>
      <int name="slotMax" value="200"/>
    </imgdir>
  </imgdir>
  <imgdir name="04000001">
    <imgdir name="info">
      <canvas name="icon" width="30" height="28">
        <vector name="origin" x="-1" y="28"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="26">
        <vector name="origin" x="-1" y="28"/>
      </canvas>
      <int name="price" value="4"/>
      <int name="slotMax" value="200"/>
    </imgdir>
  </imgdir>
  <imgdir name="04001434">
    <imgdir name="info">
      <canvas name="icon" width="31" height="30">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="31" height="26">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <int name="price" value="1"/>
      <int name="notSale" value="1"/>
      <int name="slotMax" value="100"/>
    </imgdir>
  </imgdir>
</imgdir>`
