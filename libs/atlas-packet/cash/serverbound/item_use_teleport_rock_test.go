package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
// NOTE: no gms_v95 marker yet in this fix commit — it lands in the follow-up
// verification commit alongside the pinned evidence record and audit report
// (VERIFYING_A_PACKET.md §4: wire divergence is its own commit before the
// verification commit).
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
