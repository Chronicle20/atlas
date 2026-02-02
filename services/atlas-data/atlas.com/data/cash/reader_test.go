package cash

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
	"strconv"
	"testing"
)

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
