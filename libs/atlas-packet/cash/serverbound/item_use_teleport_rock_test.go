package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Sub-payload of SendConsumeCashItemUseRequest for teleport rocks (design §1
// Q1): shared RunMapTransferItem target payload + trailing int updateTime on
// ALL versions (v83 tail 0xA0EA53, v95 case 0x9EE059).
//
// task-124 verify pass (live MapleStory_dump.exe v83, port 13342): case 22 of
// the jumptable at CWvsContext::SendConsumeCashItemUseRequest @0xa0a63f
// (@0xa0cab0) computes a flag = (itemId/1000 != 5040) and calls
// CWvsContext::RunMapTransferItem(this, &packet, flag) @0xa0a4aa — the SAME
// helper USE_TELEPORT_ROCK calls — then falls through to the shared send tail
// @0xa0ea53 (Encode4(update_time); SendPacket). candidatesFromFName keys this
// packet as cash/serverbound/CashItemUseTeleportRock (pkg="cash" qualifier,
// matching the ItemUsePointReset/ItemUseVegaScroll sibling convention); the
// marker address is the resolved fname's function entry (0xa0a63f), matching
// the audit report/evidence Address field — not the internal case or tail
// address, mirroring CashItemUsePointReset's convention.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=gms_v83 ida=0xa0a63f
//
// NOTE: no gms_v95 marker is carried here yet (task-124 is a v83-only verify
// pass) — a marker without a v95 audit report/evidence record would itself
// register as an orphan (mirrors the ItemUseVegaScroll convention in
// item_use_vega_scroll_test.go). The 0x9EE059 address noted above is a
// design-time claim, not IDA-reverified this session.
func TestItemUseTeleportRockByMap(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // trailing updateTime = 42
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetMap() != 100000000 || p.UpdateTime() != 42 {
		t.Fatalf("decode: target=%+v updateTime=%d", p.Target(), p.UpdateTime())
	}
}

func TestItemUseTeleportRockByName(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01,       // byName = 1
		0x05, 0x00, // name length
		'A', 'd', 'e', 'l', 'e',
		0x00, 0x00, 0x00, 0x00,
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v", p.Target())
	}
}

func TestItemUseTeleportRockAbsentTarget(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{0x2A, 0x00, 0x00, 0x00} // updateTime only
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if p.Target().Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}

// TestItemUseTeleportRockRoundTrip exercises the Encode path (previously
// untested — review finding, task-124) across pt.Variants for both target
// shapes, matching the sibling round-trip idiom in
// teleportrock/serverbound/use_test.go (TestUseRoundTrip) and
// item_use_pet_consumable_test.go.
func TestItemUseTeleportRockRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/byMap", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseTeleportRock{target: teleportrock.NewTargetByMap(220000000), updateTime: 7}
			output := ItemUseTeleportRock{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Target().Valid() || output.Target().ByName() || output.Target().TargetMap() != 220000000 || output.UpdateTime() != 7 {
				t.Fatalf("round trip: target=%+v updateTime=%d", output.Target(), output.UpdateTime())
			}
		})

		t.Run(v.Name+"/byName", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseTeleportRock{target: teleportrock.NewTargetByName("Adele"), updateTime: 42}
			output := ItemUseTeleportRock{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Target().Valid() || !output.Target().ByName() || output.Target().TargetName() != "Adele" || output.UpdateTime() != 42 {
				t.Fatalf("round trip: target=%+v updateTime=%d", output.Target(), output.UpdateTime())
			}
		})
	}
}
