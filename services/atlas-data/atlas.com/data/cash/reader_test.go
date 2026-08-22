package cash

import (
	"atlas-data/xml"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// readCashFixture parses fixture XML and returns a slice of RestModel
func readCashFixture(t *testing.T, xmlData string) []RestModel {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(xmlData)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	// Convert map to slice, maintaining order
	models := make([]RestModel, 0, len(rmm))
	for _, id := range rmm {
		models = append(models, id)
	}
	return models
}

const testXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0524.img">
  <imgdir name="05240000">
    <imgdir name="info">
      <canvas name="icon" width="29" height="29">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="29" height="27">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
      <int name="tradeBlock" value="1"/>
      <int name="only" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000011"/>
      <int name="1" value="5000007"/>
      <int name="2" value="5000013"/>
      <int name="3" value="5000021"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240001">
    <imgdir name="info">
      <canvas name="icon" width="27" height="28">
        <vector name="origin" x="-3" y="28"/>
      </canvas>
      <canvas name="iconRaw" width="27" height="24">
        <vector name="origin" x="-3" y="28"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000017"/>
      <int name="1" value="5000007"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240002">
    <imgdir name="info">
      <canvas name="icon" width="29" height="29">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="29" height="26">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000001"/>
      <int name="1" value="5000006"/>
      <int name="2" value="5000007"/>
      <int name="3" value="5000018"/>
      <int name="4" value="5000037"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240003">
    <imgdir name="info">
      <canvas name="icon" width="30" height="33">
        <vector name="origin" x="0" y="33"/>
      </canvas>
      <canvas name="iconRaw" width="28" height="31">
        <vector name="origin" x="0" y="33"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000008"/>
      <int name="1" value="5000007"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240004">
    <imgdir name="info">
      <canvas name="icon" width="28" height="29">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="28" height="26">
        <vector name="origin" x="-2" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000000"/>
      <int name="1" value="5000004"/>
      <int name="2" value="5000007"/>
      <int name="3" value="5000023"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240005">
    <imgdir name="info">
      <canvas name="icon" width="28" height="30">
        <vector name="origin" x="-2" y="30"/>
      </canvas>
      <canvas name="iconRaw" width="27" height="28">
        <vector name="origin" x="-2" y="30"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000002"/>
      <int name="1" value="5000005"/>
      <int name="2" value="5000007"/>
      <int name="3" value="5000013"/>
      <int name="4" value="5000014"/>
      <int name="5" value="5000015"/>
      <int name="6" value="5000034"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240006">
    <imgdir name="info">
      <canvas name="icon" width="29" height="27">
        <vector name="origin" x="-1" y="27"/>
      </canvas>
      <canvas name="iconRaw" width="29" height="23">
        <vector name="origin" x="-1" y="27"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000003"/>
      <int name="1" value="5000009"/>
      <int name="2" value="5000010"/>
      <int name="3" value="5000007"/>
      <int name="4" value="5000012"/>
      <int name="5" value="5000044"/>
      <int name="6" value="5000101"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240007">
    <imgdir name="info">
      <canvas name="icon" width="27" height="31">
        <vector name="origin" x="-3" y="31"/>
      </canvas>
      <canvas name="iconRaw" width="27" height="29">
        <vector name="origin" x="-3" y="31"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000020"/>
      <int name="1" value="5000007"/>
      <int name="2" value="5000102"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240008">
    <imgdir name="info">
      <canvas name="icon" width="30" height="33">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="33">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000022"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240009">
    <imgdir name="info">
      <canvas name="icon" width="33" height="29">
        <vector name="origin" x="0" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="33" height="26">
        <vector name="origin" x="0" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000024"/>
      <int name="1" value="5000007"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240010">
    <imgdir name="info">
      <canvas name="icon" width="37" height="36">
        <vector name="origin" x="2" y="34"/>
      </canvas>
      <canvas name="iconRaw" width="37" height="36">
        <vector name="origin" x="2" y="34"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000025"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240011">
    <imgdir name="info">
      <canvas name="icon" width="34" height="36">
        <vector name="origin" x="0" y="36"/>
      </canvas>
      <canvas name="iconRaw" width="34" height="38">
        <vector name="origin" x="0" y="37"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000026"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240012">
    <imgdir name="info">
      <canvas name="icon" width="30" height="31">
        <vector name="origin" x="-1" y="31"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="28">
        <vector name="origin" x="-1" y="31"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000029"/>
      <int name="1" value="5000030"/>
      <int name="2" value="5000031"/>
      <int name="3" value="5000032"/>
      <int name="4" value="5000033"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240013">
    <imgdir name="info">
      <canvas name="icon" width="30" height="31">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="30">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000036"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240015">
    <imgdir name="info">
      <canvas name="icon" width="33" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <canvas name="iconRaw" width="33" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000039"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240017">
    <imgdir name="info">
      <canvas name="icon" width="34" height="31">
        <vector name="origin" x="1" y="31"/>
      </canvas>
      <canvas name="iconRaw" width="34" height="30">
        <vector name="origin" x="1" y="31"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000041"/>
      <int name="1" value="5000055"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240018">
    <imgdir name="info">
      <canvas name="icon" width="32" height="29">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="32" height="27">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000042"/>
      <int name="1" value="5000046"/>
      <int name="2" value="5000100"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240020">
    <imgdir name="info">
      <canvas name="icon" width="30" height="28">
        <vector name="origin" x="-3" y="31"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="27">
        <vector name="origin" x="-3" y="31"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000045"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240021">
    <imgdir name="info">
      <canvas name="icon" width="34" height="27">
        <vector name="origin" x="1" y="28"/>
      </canvas>
      <canvas name="iconRaw" width="34" height="27">
        <vector name="origin" x="1" y="28"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000048"/>
      <int name="1" value="5000049"/>
      <int name="2" value="5000050"/>
      <int name="3" value="5000051"/>
      <int name="4" value="5000052"/>
      <int name="5" value="5000053"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240023">
    <imgdir name="info">
      <canvas name="icon" width="29" height="26">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <canvas name="iconRaw" width="29" height="25">
        <vector name="origin" x="-1" y="29"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000058"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240024">
    <imgdir name="info">
      <canvas name="icon" width="30" height="31">
        <vector name="origin" x="-2" y="30"/>
      </canvas>
      <canvas name="iconRaw" width="30" height="28">
        <vector name="origin" x="-2" y="30"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000060"/>
    </imgdir>
  </imgdir>
  <imgdir name="05240027">
    <imgdir name="info">
      <canvas name="icon" width="33" height="32">
        <vector name="origin" x="1" y="32"/>
      </canvas>
      <canvas name="iconRaw" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="slotMax" value="200"/>
      <int name="cash" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="inc" value="100"/>
      <int name="0" value="5000066"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func Identity[M any](m M) M {
	return m
}

const testExpCouponXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0521.img">
  <imgdir name="05211000">
    <imgdir name="info">
      <canvas name="icon" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="rate" value="2"/>
      <imgdir name="time">
        <string name="0" value="MON:18-20"/>
        <string name="1" value="TUE:18-20"/>
        <string name="2" value="WED:18-20"/>
        <string name="3" value="THU:18-20"/>
        <string name="4" value="FRI:18-20"/>
        <string name="5" value="SAT:18-20"/>
        <string name="6" value="SUN:18-20"/>
      </imgdir>
    </imgdir>
    <imgdir name="spec">
      <int name="time" value="2147483647"/>
      <int name="expR" value="2"/>
    </imgdir>
  </imgdir>
  <imgdir name="05211048">
    <imgdir name="info">
      <canvas name="icon" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="rate" value="2"/>
      <imgdir name="time">
        <string name="0" value="MON:00-24"/>
        <string name="1" value="TUE:00-24"/>
        <string name="2" value="WED:00-24"/>
        <string name="3" value="THU:00-24"/>
        <string name="4" value="FRI:00-24"/>
        <string name="5" value="SAT:00-24"/>
        <string name="6" value="SUN:00-24"/>
      </imgdir>
    </imgdir>
    <imgdir name="spec">
      <int name="time" value="2147483647"/>
      <int name="expR" value="3"/>
    </imgdir>
  </imgdir>
  <imgdir name="05211060">
    <imgdir name="info">
      <canvas name="icon" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="rate" value="3"/>
      <imgdir name="time">
        <string name="0" value="MON:00-24"/>
        <string name="1" value="TUE:00-24"/>
        <string name="2" value="WED:00-24"/>
        <string name="3" value="THU:00-24"/>
        <string name="4" value="FRI:00-24"/>
        <string name="5" value="SAT:00-24"/>
        <string name="6" value="SUN:00-24"/>
        <string name="7" value="HOL:00-24"/>
      </imgdir>
    </imgdir>
    <imgdir name="spec">
      <int name="time" value="2147483647"/>
      <int name="expR" value="4"/>
    </imgdir>
  </imgdir>
</imgdir>
`

const testDropCouponXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0536.img">
  <imgdir name="05360000">
    <imgdir name="info">
      <canvas name="icon" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="rate" value="2"/>
      <imgdir name="time">
        <string name="0" value="MON:00-24"/>
        <string name="1" value="TUE:00-24"/>
        <string name="2" value="WED:00-24"/>
        <string name="3" value="THU:00-24"/>
        <string name="4" value="FRI:00-24"/>
        <string name="5" value="SAT:00-24"/>
        <string name="6" value="SUN:00-24"/>
      </imgdir>
    </imgdir>
    <imgdir name="spec">
      <int name="time" value="2147483647"/>
      <int name="drpR" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05360042">
    <imgdir name="info">
      <canvas name="icon" width="32" height="32">
        <vector name="origin" x="0" y="32"/>
      </canvas>
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="rate" value="2"/>
      <imgdir name="time">
        <string name="0" value="MON:00-24"/>
        <string name="1" value="TUE:00-24"/>
        <string name="2" value="WED:00-24"/>
        <string name="3" value="THU:00-24"/>
        <string name="4" value="FRI:00-24"/>
        <string name="5" value="SAT:00-24"/>
        <string name="6" value="SUN:00-24"/>
      </imgdir>
    </imgdir>
    <imgdir name="spec">
      <int name="time" value="2147483647"/>
      <int name="drpR" value="2"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReader(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 22 {
		t.Fatalf("len(rmm) = %d, want 22", len(rmm))
	}

	var rm RestModel
	var ok bool
	var spec int32

	if rm, ok = rmm[strconv.Itoa(5240027)]; !ok {
		t.Fatalf("rmm[5240027] does not exist.")
	}
	if spec, ok = rm.Spec[SpecTypeInc]; !ok {
		t.Fatalf("rmm.Spec[SpecTypeInc] does not exist.")
	}
	if spec != 100 {
		t.Fatalf("rmm.Spec[SpecTypeInc].Spec = %d, want 100", spec)
	}
	if spec, ok = rm.Spec[SpecTypeIndexZero]; !ok {
		t.Fatalf("rmm.Spec[SpecTypeIndexZero] does not exist.")
	}
	if spec != 5000066 {
		t.Fatalf("rmm.Spec[SpecTypeIndexZero].Spec = %d, want 5000066", spec)
	}
}

// TestCashReaderSurfacesTradeBlock pins PRD FR-4.2: tradeBlock must be
// readable for every item family a trade can stage, not just consumable and
// setup. A missing flag must never be read as "tradeable".
func TestCashReaderSurfacesTradeBlock(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	rm, ok := rmm[strconv.Itoa(5240000)]
	if !ok {
		t.Fatalf("rmm[5240000] does not exist.")
	}
	if !rm.TradeBlock {
		t.Error("tradeBlock: got false, want true (fixture sets tradeBlock=1)")
	}
}

// TestCashReaderTradeBlockDefaultsFalse pins PRD FR-4.2: a missing
// tradeBlock node must never be read as "tradeable". 5240001 has no
// tradeBlock node.
func TestCashReaderTradeBlockDefaultsFalse(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	rm, ok := rmm[strconv.Itoa(5240001)]
	if !ok {
		t.Fatalf("rmm[5240001] does not exist.")
	}
	if rm.TradeBlock {
		t.Error("tradeBlock: got true, want false when the WZ node is absent")
	}
}

// TestCashReaderSurfacesOnly pins the DUEY receive fix: info/only must be
// readable for cash items so the recipient-already-holds-it check can be
// conditioned on one-of-a-kind, not mere template co-occurrence.
func TestCashReaderSurfacesOnly(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	rm, ok := rmm[strconv.Itoa(5240000)]
	if !ok {
		t.Fatalf("rmm[5240000] does not exist.")
	}
	if !rm.Only {
		t.Error("only: got false, want true (fixture sets only=1)")
	}
}

// TestCashReaderOnlyDefaultsFalse pins that a missing only node must never
// be read as one-of-a-kind. 5240001 has no only node.
func TestCashReaderOnlyDefaultsFalse(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	rm, ok := rmm[strconv.Itoa(5240001)]
	if !ok {
		t.Fatalf("rmm[5240001] does not exist.")
	}
	if rm.Only {
		t.Error("only: got true, want false when the WZ node is absent")
	}
}

func TestReaderExpCoupons(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testExpCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 3 {
		t.Fatalf("len(rmm) = %d, want 3", len(rmm))
	}

	// Test 5211000 - 2x EXP coupon with restricted time windows (18-20)
	rm, ok := rmm[strconv.Itoa(5211000)]
	if !ok {
		t.Fatalf("rmm[5211000] does not exist")
	}
	if rm.SlotMax != 1 {
		t.Fatalf("rm.SlotMax = %d, want 1", rm.SlotMax)
	}
	// Check info/rate value
	rate, ok := rm.Spec[SpecTypeRate]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeRate] does not exist")
	}
	if rate != 2 {
		t.Fatalf("rm.Spec[SpecTypeRate] = %d, want 2", rate)
	}
	// Check spec/expR value
	expR, ok := rm.Spec[SpecTypeExpR]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeExpR] does not exist")
	}
	if expR != 2 {
		t.Fatalf("rm.Spec[SpecTypeExpR] = %d, want 2", expR)
	}
	specTime, ok := rm.Spec[SpecTypeTime]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeTime] does not exist")
	}
	if specTime != 2147483647 {
		t.Fatalf("rm.Spec[SpecTypeTime] = %d, want 2147483647", specTime)
	}
	if len(rm.TimeWindows) != 7 {
		t.Fatalf("len(rm.TimeWindows) = %d, want 7", len(rm.TimeWindows))
	}
	// Verify first time window
	if rm.TimeWindows[0].Day != "MON" {
		t.Fatalf("rm.TimeWindows[0].Day = %s, want MON", rm.TimeWindows[0].Day)
	}
	if rm.TimeWindows[0].StartHour != 18 {
		t.Fatalf("rm.TimeWindows[0].StartHour = %d, want 18", rm.TimeWindows[0].StartHour)
	}
	if rm.TimeWindows[0].EndHour != 20 {
		t.Fatalf("rm.TimeWindows[0].EndHour = %d, want 20", rm.TimeWindows[0].EndHour)
	}

	// Test 5211048 - 3x EXP coupon with all-day windows
	rm, ok = rmm[strconv.Itoa(5211048)]
	if !ok {
		t.Fatalf("rmm[5211048] does not exist")
	}
	// Check info/rate value
	rate, ok = rm.Spec[SpecTypeRate]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeRate] does not exist for 5211048")
	}
	if rate != 2 {
		t.Fatalf("rm.Spec[SpecTypeRate] = %d, want 2", rate)
	}
	// Check spec/expR value
	expR, ok = rm.Spec[SpecTypeExpR]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeExpR] does not exist for 5211048")
	}
	if expR != 3 {
		t.Fatalf("rm.Spec[SpecTypeExpR] = %d, want 3", expR)
	}
	if len(rm.TimeWindows) != 7 {
		t.Fatalf("len(rm.TimeWindows) = %d, want 7", len(rm.TimeWindows))
	}
	// Verify all-day window
	if rm.TimeWindows[0].StartHour != 0 {
		t.Fatalf("rm.TimeWindows[0].StartHour = %d, want 0", rm.TimeWindows[0].StartHour)
	}
	if rm.TimeWindows[0].EndHour != 24 {
		t.Fatalf("rm.TimeWindows[0].EndHour = %d, want 24", rm.TimeWindows[0].EndHour)
	}

	// Test 5211060 - 4x EXP coupon with 8 time windows (including HOL)
	rm, ok = rmm[strconv.Itoa(5211060)]
	if !ok {
		t.Fatalf("rmm[5211060] does not exist")
	}
	// Check info/rate value
	rate, ok = rm.Spec[SpecTypeRate]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeRate] does not exist for 5211060")
	}
	if rate != 3 {
		t.Fatalf("rm.Spec[SpecTypeRate] = %d, want 3", rate)
	}
	// Check spec/expR value
	expR, ok = rm.Spec[SpecTypeExpR]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeExpR] does not exist for 5211060")
	}
	if expR != 4 {
		t.Fatalf("rm.Spec[SpecTypeExpR] = %d, want 4", expR)
	}
	if len(rm.TimeWindows) != 8 {
		t.Fatalf("len(rm.TimeWindows) = %d, want 8", len(rm.TimeWindows))
	}
	// Verify HOL window
	if rm.TimeWindows[7].Day != "HOL" {
		t.Fatalf("rm.TimeWindows[7].Day = %s, want HOL", rm.TimeWindows[7].Day)
	}
}

func TestReaderDropCoupons(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testDropCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 2 {
		t.Fatalf("len(rmm) = %d, want 2", len(rmm))
	}

	// Test 5360000 - 1x drop rate coupon (base rate, no bonus)
	rm, ok := rmm[strconv.Itoa(5360000)]
	if !ok {
		t.Fatalf("rmm[5360000] does not exist")
	}
	drpR, ok := rm.Spec[SpecTypeDrpR]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeDrpR] does not exist")
	}
	if drpR != 1 {
		t.Fatalf("rm.Spec[SpecTypeDrpR] = %d, want 1", drpR)
	}
	specTime, ok := rm.Spec[SpecTypeTime]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeTime] does not exist")
	}
	if specTime != 2147483647 {
		t.Fatalf("rm.Spec[SpecTypeTime] = %d, want 2147483647", specTime)
	}
	if len(rm.TimeWindows) != 7 {
		t.Fatalf("len(rm.TimeWindows) = %d, want 7", len(rm.TimeWindows))
	}

	// Test 5360042 - 2x drop rate coupon
	rm, ok = rmm[strconv.Itoa(5360042)]
	if !ok {
		t.Fatalf("rmm[5360042] does not exist")
	}
	drpR, ok = rm.Spec[SpecTypeDrpR]
	if !ok {
		t.Fatalf("rm.Spec[SpecTypeDrpR] does not exist for 5360042")
	}
	if drpR != 2 {
		t.Fatalf("rm.Spec[SpecTypeDrpR] = %d, want 2", drpR)
	}
}

func TestParseTimeWindow(t *testing.T) {
	tests := []struct {
		input     string
		wantDay   string
		wantStart int
		wantEnd   int
		wantOk    bool
	}{
		{"MON:18-20", "MON", 18, 20, true},
		{"TUE:00-24", "TUE", 0, 24, true},
		{"HOL:00-24", "HOL", 0, 24, true},
		{"SAT:12-18", "SAT", 12, 18, true},
		{"invalid", "", 0, 0, false},
		{"MON:invalid", "", 0, 0, false},
		{"MON:18", "", 0, 0, false},
		{"", "", 0, 0, false},
	}

	for _, tt := range tests {
		tw, ok := parseTimeWindow(tt.input)
		if ok != tt.wantOk {
			t.Errorf("parseTimeWindow(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			continue
		}
		if !ok {
			continue
		}
		if tw.Day != tt.wantDay {
			t.Errorf("parseTimeWindow(%q).Day = %s, want %s", tt.input, tw.Day, tt.wantDay)
		}
		if tw.StartHour != tt.wantStart {
			t.Errorf("parseTimeWindow(%q).StartHour = %d, want %d", tt.input, tw.StartHour, tt.wantStart)
		}
		if tw.EndHour != tt.wantEnd {
			t.Errorf("parseTimeWindow(%q).EndHour = %d, want %d", tt.input, tw.EndHour, tt.wantEnd)
		}
	}
}

const testSealingLockXML = `<?xml version="1.0" encoding="UTF-8"?>
<imgdir name="0506.img">
  <imgdir name="05061000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="protectTime" value="7"/>
    </imgdir>
  </imgdir>
  <imgdir name="05061001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="protectTime" value="30"/>
    </imgdir>
  </imgdir>
</imgdir>`

func TestReaderProtectTime(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testSealingLockXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if got := rmm[strconv.Itoa(5061000)].ProtectTime; got != 7 {
		t.Fatalf("ProtectTime(5061000) = %d, want 7", got)
	}
	if got := rmm[strconv.Itoa(5061001)].ProtectTime; got != 30 {
		t.Fatalf("ProtectTime(5061001) = %d, want 30", got)
	}
}

const testPetSkillXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0519.img">
  <imgdir name="05190001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeHP" value="1"/>
      <int name="add" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05190006">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeMP" value="1"/>
      <int name="add" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05191001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeHP" value="1"/>
      <int name="add" value="0"/>
    </imgdir>
  </imgdir>
</imgdir>
`

const testRemoteMerchantXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0545.img">
  <imgdir name="5450000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="npc" value="9090000"/>
      <int name="slotMax" value="100"/>
    </imgdir>
  </imgdir>
  <imgdir name="5451000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="100"/>
    </imgdir>
  </imgdir>
</imgdir>
`

// TestRead_ParsesInfoNpcForRemoteMerchantItems pins task-221 FR-1: a cash
// item's info/npc value (the NPC template a remote-merchant item opens, e.g.
// 9090000 for MiuMiu's Travel Store) must be exposed on RestModel, and must
// default to 0 when the item has no info/npc node.
func TestRead_ParsesInfoNpcForRemoteMerchantItems(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testRemoteMerchantXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 2 {
		t.Fatalf("len(rmm) = %d, want 2", len(rmm))
	}

	if got := rmm[strconv.Itoa(5450000)].Npc; got != 9090000 {
		t.Errorf("5450000 Npc = %d, want 9090000", got)
	}
	if got := rmm[strconv.Itoa(5451000)].Npc; got != 0 {
		t.Errorf("5451000 Npc = %d, want 0 (item has no info/npc)", got)
	}
}

func TestReaderPetSkills(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testPetSkillXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 3 {
		t.Fatalf("len(rmm) = %d, want 3", len(rmm))
	}

	cases := []struct {
		id     string
		skills []string
		add    bool
	}{
		{"5190001", []string{"consumeHP"}, true},
		{"5190006", []string{"consumeMP"}, true},
		{"5191001", []string{"consumeHP"}, false},
	}
	for _, c := range cases {
		rm, ok := rmm[c.id]
		if !ok {
			t.Fatalf("rmm[%s] does not exist", c.id)
		}
		if len(rm.PetSkills) != len(c.skills) || rm.PetSkills[0] != c.skills[0] {
			t.Errorf("[%s] PetSkills = %v, want %v", c.id, rm.PetSkills, c.skills)
		}
		if rm.PetSkillAdd != c.add {
			t.Errorf("[%s] PetSkillAdd = %t, want %t", c.id, rm.PetSkillAdd, c.add)
		}
	}
}

const testSandglassXML = `<?xml version="1.0" encoding="UTF-8"?>
<imgdir name="0550.img">
  <imgdir name="05500000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="86400"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="604800"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="1728000"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500005">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="4320000"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500006">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="8553600"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500009">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>`

func TestReaderSandglassAddTimeAndMaxDays(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testSandglassXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		id      int
		addTime uint32
		maxDays uint32
	}{
		{5500000, 86400, 30},
		{5500001, 604800, 30},
		{5500002, 1728000, 30},
		{5500005, 4320000, 30},
		{5500006, 8553600, 30},
		{5500009, 0, 0}, // both fields absent -> default 0
	}
	for _, c := range cases {
		rm := rmm[strconv.Itoa(c.id)]
		if rm.AddTime != c.addTime {
			t.Errorf("AddTime(%d) = %d, want %d", c.id, rm.AddTime, c.addTime)
		}
		if rm.MaxDays != c.maxDays {
			t.Errorf("MaxDays(%d) = %d, want %d", c.id, rm.MaxDays, c.maxDays)
		}
	}
}

// testMesoSackXML mirrors the real GMS 83.1 Item.wz/Cash/0520.img node set:
// icon/iconRaw/meso/cash only — no slotMax, no spec, no tradeBlock. 05200003
// carries an explicit meso of 0 and 05200004 omits the node entirely; both
// must land on Meso == 0 so the handler's fail-closed guard trips (FR-1.2).
const testMesoSackXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0520.img">
  <imgdir name="05200000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="1000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="5000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="10000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200003">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="0"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200004">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="maplepoint" value="10000"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReaderMesoSacks(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMesoSackXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		id   int
		want uint32
	}{
		{5200000, 1000000},
		{5200001, 5000000},
		{5200002, 10000000},
		{5200003, 0}, // explicit zero
		{5200004, 0}, // node absent (Maple Point sack)
	}
	for _, tc := range cases {
		rm, ok := rmm[strconv.Itoa(tc.id)]
		if !ok {
			t.Fatalf("cash item %d missing from read result", tc.id)
		}
		if rm.Meso != tc.want {
			t.Errorf("Meso(%d) = %d, want %d", tc.id, rm.Meso, tc.want)
		}
	}
}

// The award amount is a first-class field, not an effect: folding it into Spec
// would feed it to the consumable pipeline. Spec must gain no "meso" key.
func TestReaderMesoNotFoldedIntoSpec(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMesoSackXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if _, present := rmm[strconv.Itoa(5200000)].Spec[SpecType("meso")]; present {
		t.Fatal(`Spec gained a "meso" key; the award amount must stay a first-class field`)
	}
}

// testMorphCouponXML mirrors Item.wz/Cash/0530.img.xml (transformation coupons,
// classification 530), trimmed of canvas nodes. Values verified against two
// independent local WZ corpora: every item carries spec/hp 50 and
// spec/time 600000, with spec/morph 1, 2, 3 respectively, and no morphRandom.
const testMorphCouponXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0530.img">
  <imgdir name="05300000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05300001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="2"/>
    </imgdir>
  </imgdir>
  <imgdir name="05300002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="3"/>
    </imgdir>
  </imgdir>
</imgdir>`

// TestReaderMorphCoupons pins FR-2.3: all three 0530 items surface morph, hp and
// time. Before this task the reader dropped morph and hp entirely, so the coupon
// was inert no matter what the downstream services did.
func TestReaderMorphCoupons(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMorphCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 3 {
		t.Fatalf("len(rmm) = %d, want 3", len(rmm))
	}

	for id, wantMorph := range map[int]int32{5300000: 1, 5300001: 2, 5300002: 3} {
		rm, ok := rmm[strconv.Itoa(id)]
		if !ok {
			t.Fatalf("rmm[%d] does not exist", id)
		}
		if rm.SlotMax != 200 {
			t.Errorf("[%d] SlotMax = %d, want 200", id, rm.SlotMax)
		}
		if !rm.TradeBlock {
			t.Errorf("[%d] TradeBlock = false, want true", id)
		}
		morph, ok := rm.Spec[SpecTypeMorph]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeMorph] does not exist", id)
		}
		if morph != wantMorph {
			t.Errorf("[%d] Spec[SpecTypeMorph] = %d, want %d", id, morph, wantMorph)
		}
		hp, ok := rm.Spec[SpecTypeHp]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeHp] does not exist", id)
		}
		if hp != 50 {
			t.Errorf("[%d] Spec[SpecTypeHp] = %d, want 50", id, hp)
		}
		specTime, ok := rm.Spec[SpecTypeTime]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeTime] does not exist", id)
		}
		// 600000 is the raw WZ value in MILLISECONDS. atlas-buffs' duration
		// contract is milliseconds, so nothing on this path may rescale it.
		if specTime != 600000 {
			t.Errorf("[%d] Spec[SpecTypeTime] = %d, want 600000", id, specTime)
		}
	}
}

// TestReaderMorphHpAdditiveOnly pins FR-2.4: the two new keys are omit-when-zero,
// so a non-0530 cash item's parse output gains nothing. 5211000 is a 0521 EXP
// coupon, already covered end-to-end by TestReaderExpCoupons; this asserts the
// only thing that could have regressed — spurious keys.
func TestReaderMorphHpAdditiveOnly(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testExpCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	for id := range rmm {
		rm := rmm[id]
		if v, ok := rm.Spec[SpecTypeMorph]; ok {
			t.Errorf("[%s] Spec[SpecTypeMorph] = %d, want absent on a 0521 EXP coupon", id, v)
		}
		if v, ok := rm.Spec[SpecTypeHp]; ok {
			t.Errorf("[%s] Spec[SpecTypeHp] = %d, want absent on a 0521 EXP coupon", id, v)
		}
	}
}

func TestReaderParsesLife(t *testing.T) {
	// 0518.img/05180000/info: slotMax=1, cash=1, life=90. No maxDays, no addTime.
	models := readCashFixture(t, `
		<imgdir name="0518.img">
			<imgdir name="05180000">
				<imgdir name="info">
					<int name="slotMax" value="1"/>
					<int name="cash" value="1"/>
					<int name="life" value="90"/>
				</imgdir>
			</imgdir>
		</imgdir>`)

	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
	if models[0].Life != 90 {
		t.Errorf("Life = %d, want 90", models[0].Life)
	}
	if models[0].MaxDays != 0 {
		t.Errorf("MaxDays = %d, want 0 (0518.img has no maxDays node)", models[0].MaxDays)
	}
}

func TestReaderLifeAbsentIsZeroAndOmitted(t *testing.T) {
	models := readCashFixture(t, `
		<imgdir name="0518.img">
			<imgdir name="05180000">
				<imgdir name="info">
					<int name="slotMax" value="1"/>
				</imgdir>
			</imgdir>
		</imgdir>`)

	if models[0].Life != 0 {
		t.Errorf("Life = %d, want 0", models[0].Life)
	}
	b, err := json.Marshal(models[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"life"`) {
		t.Errorf("absent life must be omitted from JSON, got %s", b)
	}
}
