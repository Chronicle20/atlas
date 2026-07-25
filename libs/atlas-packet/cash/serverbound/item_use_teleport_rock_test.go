package serverbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Sub-payload of SendConsumeCashItemUseRequest for teleport rocks: shared
// RunMapTransferItem target payload, then a trailing int updateTime ONLY on
// MajorVersion()<87 (v83/v84 — updateTimeFirst=false). On MajorVersion()>=87
// (v87/v95/jms) update_time is already consumed by the parent ItemUse's
// leading header int32, so the sub-body is the target payload alone.
//
// task-124 v83 verify pass (live MapleStory_dump.exe v83, port 13342): case 22
// of the jumptable at CWvsContext::SendConsumeCashItemUseRequest @0xa0a63f
// (@0xa0cab0) computes a flag = (itemId/1000 != 5040) and calls
// CWvsContext::RunMapTransferItem(this, &packet, flag) @0xa0a4aa — the SAME
// helper USE_TELEPORT_ROCK calls — then falls through to the shared send tail
// @0xa0ea53 (Encode4(update_time); SendPacket): a genuine TRAILING updateTime
// on v83. candidatesFromFName keys this packet as
// cash/serverbound/CashItemUseTeleportRock (pkg="cash" qualifier, matching the
// ItemUsePointReset/ItemUseVegaScroll sibling convention); the marker address
// is the resolved fname's function entry (0xa0a63f), matching the audit
// report/evidence Address field — not the internal case or tail address,
// mirroring CashItemUsePointReset's convention.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=gms_v83 ida=0xa0a63f
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

// task-124 v84 verify pass (live GMS_v84.1_U_DEVM.exe, port 13345): case 22
// of the jumptable at CWvsContext::SendConsumeCashItemUseRequest @0xa54a2f
// (case body @0xa56e9d, reached via byte_A58F60[22-12]=8 ->
// jpt_A54ADD[8]=0xa56e9d) computes a flag = (itemId/1000 != 5040) @0xa56ea9
// and calls CWvsContext::RunMapTransferItem(this, &v_pkt, flag) @0xa56ebb —
// the SAME helper USE_TELEPORT_ROCK calls (0xa547ab, both renamed live from
// sub_A5489A/sub_A547AB during this pass) — then on success falls through to
// the shared send tail @0xa58e47: Encode4(get_update_time()) @0xa58e50;
// SendPacket @0xa58e59 — a genuine TRAILING updateTime. No leading
// updateTime precedes the switch dispatch (confirmed by disassembling the
// function prologue from 0xa54a2f), consistent with updateTimeFirst :=
// MajorVersion()>=87 (84 < 87) in character_cash_item_use.go — i.e. v84
// behaves exactly like v83 here (updateTimeFirst=false, trailing updateTime),
// NOT like v95's leading-header case below.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=gms_v84 ida=0xa54a2f
func TestItemUseTeleportRockByMapV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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

// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::SendConsumeCashItemUseRequest @0x9eb3e0, opcode 0x55. The
// header encodes update_time FIRST (Encode4 @0x9eb4b7, BEFORE
// Encode2(nPOS)/Encode4(nItemID) and the switch on
// get_consume_cash_item_type(nItemID)) — already the parent ItemUse's leading
// updateTime (updateTimeFirst := MajorVersion()>=87 in
// character_cash_item_use.go). Case 22 ($LN84_16 @0x9ee059) computes
// bCanTransferContinent = (nItemID / 5040 != 5040, via the 0x10624DD3
// magic-multiply division) and calls
// CWvsContext::RunMapTransferItem(this, &oPacket, flag) @0x9ee080 — the SAME
// helper USE_TELEPORT_ROCK calls (0x9e11c0, byte-identical target payload
// logic to v83) — then falls straight to the shared send tail
// ($LN232_14 @0x9f063c: CanSendExclRequest + SendPacket) with NO further
// Encode4 anywhere in that path (confirmed by disassembling through to
// SendPacket @0x9f066b). So the v95 sub-body is EXACTLY the target payload —
// nothing trails it. This CORRECTS the original task-124 design hypothesis
// (§1 Q1, which assumed a v95 trailing updateTime at the case-22 tail
// 0x9EE059) and the bug this pass fixed in item_use_teleport_rock.go / the
// shared teleportrock.Target codec (which previously reserved a phantom
// trailing 4-byte budget unconditionally, corrupting map-target decodes on
// this exact MajorVersion()>=87 path: a genuine 5-byte by-map payload with
// nothing following was misread as "no selection").
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=gms_v95 ida=0x9eb3e0
func TestItemUseTeleportRockByMapV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		// nothing else — v95 has no trailing update-time budget in this sub-body
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("decode: target=%+v", p.Target())
	}
}

// task-124 v87 verify pass (live GMSv87_4GB.exe, port 13343): v87 is the
// FIRST MajorVersion()>=87 version in the coverage set. The function prologue
// of CWvsContext::SendConsumeCashItemUseRequest @0xa9fef9 writes
// COutPacket::COutPacket(&pkt, 0x52) @0xa9ff67 (opcode 82, matches registry
// USE_CASH_ITEM), then IMMEDIATELY Encode4(get_update_time()) @0xa9ff7c —
// BEFORE Encode2(nPOS) @0xa9ff87 and Encode4(nItemID) @0xa9ff93 and the
// switch on get_consume_cash_item_type(nItemID) @0xa9ff99 — i.e. a leading
// common-header updateTime, exactly mirroring v95's 0x9eb3e0 shape. Case 22
// (jumptable 00A9FFB5, label loc_AA240C @0xaa240c) computes
// flag = (itemId/1000 != 5040) @0xaa2418-241d and calls
// CWvsContext::RunMapTransferItem(this, &oPacket, flag) @0xaa242a — the SAME
// helper USE_TELEPORT_ROCK calls (renamed live this pass from sub_A9FD64) —
// then on success falls to the shared tail (CanSendExclRequest @0xaa01c9,
// then on success SendPacketThunk @0xaa43ac, renamed live from sub_556185 and
// confirmed via decompile to be a pure CClientSocket::SendPacket wrapper with
// NO Encode call inside it): CONFIRMED there is NO further Encode4 anywhere
// between the case-22 RunMapTransferItem call and the final SendPacket. This
// CONFIRMS the v95 fix (item_use_teleport_rock.go's `!m.updateTimeFirst`
// gate) generalizes to v87: like v95, the sub-body is ONLY the target
// payload — no trailing updateTime, because update_time was already consumed
// by the common header before the switch.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=gms_v87 ida=0xa9fef9
func TestItemUseTeleportRockByMapV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		// nothing else — v87 has no trailing update-time budget in this sub-body
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("decode: target=%+v", p.Target())
	}
}

// task-124 jms_v185 verify pass (live MapleStory_dump_SCY.exe, port 13344):
// jms_v185 MajorVersion() is 185, so >=87 — like v87/v95, the sub-body has no
// trailing update-time budget. CWvsContext::SendConsumeCashItemUseRequest
// @0xaef2f5 writes COutPacket::COutPacket(&pkt,0x47) @0xaef363 (opcode 71,
// matches registry USE_CASH_ITEM), then IMMEDIATELY Encode4(get_update_time())
// @0xaef36c — BEFORE Encode2(nPOS) @0xaef37a and Encode4(nItemID) @0xaef385
// and the switch on get_consume_cash_item_type(nItemID) @0xaef393 — i.e. a
// leading common-header updateTime, exactly mirroring v87/v95's shape. Case 22
// (jumptable 00AEF3A8, label loc_AF14BA @0xaf14ba) calls a validity check
// (sub_AF2D2B @0xaf14bd), then on the success path computes
// flag = (itemId/1000 != 5040) @0xaf14e6-14f0 and calls
// CWvsContext::RunMapTransferItem(this,&oPacket,flag) @0xaf14f9 — the SAME
// helper USE_TELEPORT_ROCK calls (renamed live this pass from sub_AEF160,
// shared with the jms Use codec's helper at the same address 0xaef160) — then
// on success falls to the shared tail (loc_AF2AD2: CanSendExclRequest
// @0xaf2adb, then on success sub_56BC92 @0xaf2afc, decompiled and confirmed to
// be a pure CClientSocket::SendPacket wrapper with NO Encode call inside it):
// CONFIRMED there is no further Encode4 anywhere between the case-22
// RunMapTransferItem call and the final SendPacket. This CONFIRMS the v87/v95
// fix (item_use_teleport_rock.go's `!m.updateTimeFirst` gate) generalizes to
// jms_v185: the sub-body is ONLY the target payload — no trailing updateTime.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTeleportRock version=jms_v185 ida=0xaef2f5
func TestItemUseTeleportRockByMapJms(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		// nothing else — jms has no trailing update-time budget in this sub-body
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("decode: target=%+v", p.Target())
	}
}

func TestItemUseTeleportRockByName(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01,       // byName = 1
		0x05, 0x00, // name length
		'A', 'd', 'e', 'l', 'e',
		// nothing else — v95 has no trailing update-time budget in this sub-body
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || !p.Target().ByName() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v", p.Target())
	}
}

func TestItemUseTeleportRockAbsentTargetV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{} // nothing at all — v95 has no trailing budget to fall back on
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if p.Target().Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}

// TestItemUseTeleportRockRoundTrip exercises the Encode path across both
// updateTimeFirst configurations (matching TestItemUsePointResetRoundTrip's
// convention in item_use_point_reset_test.go — utf=false models v83/v84's
// trailing tail, utf=true models v87+/v95/jms's leading-header consumption).
func TestItemUseTeleportRockRoundTrip(t *testing.T) {
	for _, utf := range []bool{true, false} {
		name := "trailingUpdateTime"
		if utf {
			name = "updateTimeFirst"
		}
		t.Run(name+"/byMap", func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := ItemUseTeleportRock{target: teleportrock.NewTargetByMap(220000000), updateTime: 7, updateTimeFirst: utf}
			output := ItemUseTeleportRock{updateTimeFirst: utf}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Target().Valid() || output.Target().ByName() || output.Target().TargetMap() != 220000000 {
				t.Fatalf("round trip target: %+v", output.Target())
			}
			if !utf && output.UpdateTime() != 7 {
				t.Fatalf("round trip updateTime: got %d want 7", output.UpdateTime())
			}
		})

		t.Run(name+"/byName", func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := ItemUseTeleportRock{target: teleportrock.NewTargetByName("Adele"), updateTime: 42, updateTimeFirst: utf}
			output := ItemUseTeleportRock{updateTimeFirst: utf}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Target().Valid() || !output.Target().ByName() || output.Target().TargetName() != "Adele" {
				t.Fatalf("round trip target: %+v", output.Target())
			}
			if !utf && output.UpdateTime() != 42 {
				t.Fatalf("round trip updateTime: got %d want 42", output.UpdateTime())
			}
		})
	}
}

// TestItemUseTeleportRockRoundTripAcrossVariants keeps the original
// cross-tenant-variant coverage (previously untested Encode path — review
// finding, task-124) with a fixed updateTimeFirst=false shape, matching the
// sibling round-trip idiom in teleportrock/serverbound/use_test.go
// (TestUseRoundTrip) and item_use_pet_consumable_test.go.
func TestItemUseTeleportRockRoundTripAcrossVariants(t *testing.T) {
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
