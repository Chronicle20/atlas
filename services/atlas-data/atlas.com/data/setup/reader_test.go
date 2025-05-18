package setup

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
		t.Fatalf("Expected 3 setup items, got %d", len(res))
	}

	// Test the first setup item
	setup1 := res[0]
	if setup1.Id != 3010000 {
		t.Fatalf("Expected ID 3010000, got %d", setup1.Id)
	}
	if setup1.Price != 100 {
		t.Fatalf("Expected price 100, got %d", setup1.Price)
	}
	if setup1.SlotMax != 1 {
		t.Fatalf("Expected slotMax 1, got %d", setup1.SlotMax)
	}
	if setup1.RecoveryHP != 50 {
		t.Fatalf("Expected recoveryHP 50, got %d", setup1.RecoveryHP)
	}
	if !setup1.TradeBlock {
		t.Fatalf("Expected tradeBlock true, got false")
	}
	if !setup1.NotSale {
		t.Fatalf("Expected notSale true, got false")
	}

	// Test the second setup item
	setup2 := res[1]
	if setup2.Id != 3010001 {
		t.Fatalf("Expected ID 3010001, got %d", setup2.Id)
	}
	if setup2.Price != 500 {
		t.Fatalf("Expected price 500, got %d", setup2.Price)
	}
	if setup2.SlotMax != 1 {
		t.Fatalf("Expected slotMax 1, got %d", setup2.SlotMax)
	}
	if setup2.RecoveryHP != 35 {
		t.Fatalf("Expected recoveryHP 35, got %d", setup2.RecoveryHP)
	}
	if setup2.ReqLevel != 6 {
		t.Fatalf("Expected reqLevel 6, got %d", setup2.ReqLevel)
	}

	// Test the third setup item
	setup3 := res[2]
	if setup3.Id != 3012011 {
		t.Fatalf("Expected ID 3012011, got %d", setup3.Id)
	}
	if setup3.Price != 1 {
		t.Fatalf("Expected price 1, got %d", setup3.Price)
	}
	if setup3.SlotMax != 1 {
		t.Fatalf("Expected slotMax 1, got %d", setup3.SlotMax)
	}
	if setup3.RecoveryHP != 50 {
		t.Fatalf("Expected recoveryHP 50, got %d", setup3.RecoveryHP)
	}
	if setup3.DistanceX != 100 {
		t.Fatalf("Expected distanceX 100, got %d", setup3.DistanceX)
	}
	if setup3.DistanceY != 0 {
		t.Fatalf("Expected distanceY 0, got %d", setup3.DistanceY)
	}
	if setup3.MaxDiff != 5 {
		t.Fatalf("Expected maxDiff 5, got %d", setup3.MaxDiff)
	}
	if setup3.Direction != 21 {
		t.Fatalf("Expected direction 21, got %d", setup3.Direction)
	}
}

const testXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0301.img">
  <imgdir name="03010000">
    <imgdir name="info">
      <canvas name="icon" width="33" height="28">
        <vector name="origin" x="1" y="30"/>
      </canvas>
      <canvas name="iconRaw" width="33" height="26">
        <vector name="origin" x="1" y="30"/>
      </canvas>
      <int name="price" value="100"/>
      <int name="slotMax" value="1"/>
      <int name="recoveryHP" value="50"/>
      <int name="tradeBlock" value="1"/>
      <int name="notSale" value="1"/>
    </imgdir>
    <imgdir name="effect">
      <canvas name="0" width="58" height="36">
        <vector name="origin" x="21" y="-16"/>
      </canvas>
      <int name="z" value="-1"/>
      <int name="pos" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="03010001">
    <imgdir name="info">
      <canvas name="icon" width="29" height="29">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="29" height="27">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <int name="price" value="500"/>
      <int name="slotMax" value="1"/>
      <int name="recoveryHP" value="35"/>
      <int name="reqLevel" value="6"/>
    </imgdir>
    <imgdir name="effect">
      <canvas name="0" width="41" height="36">
        <vector name="origin" x="16" y="-15"/>
      </canvas>
      <int name="z" value="-1"/>
      <int name="pos" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="03012011">
    <imgdir name="info">
      <canvas name="icon" width="34" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <canvas name="iconRaw" width="34" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="distanceX" value="100"/>
      <int name="distanceY" value="0"/>
      <int name="maxDiff" value="5"/>
      <int name="direction" value="21"/>
      <int name="price" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="recoveryHP" value="50"/>
    </imgdir>
    <imgdir name="effect">
      <canvas name="0" width="66" height="39">
        <vector name="origin" x="28" y="-9"/>
        <int name="z" value="0"/>
        <int name="delay" value="144"/>
      </canvas>
      <int name="z" value="-1"/>
      <int name="pos" value="1"/>
    </imgdir>
    <imgdir name="effect2">
      <canvas name="0" width="26" height="12">
        <vector name="origin" x="22" y="-19"/>
        <int name="z" value="0"/>
        <int name="delay" value="144"/>
      </canvas>
      <int name="z" value="3"/>
      <int name="pos" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>`