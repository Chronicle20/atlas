package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestUseCatchItemBytes pins the wire layout of the serverbound USE_CATCH_ITEM
// request (CWvsContext::SendBridleItemUseRequest). The body is identical on
// every version inspected — Encode4 updateTime, Encode2 nPOS, Encode4 nItemID,
// Encode4 hit-mob object id — so there is deliberately NO version gate here
// (design.md §5.1; PRD FR-1.1).
//
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v48 ida=0x70e0c5
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v61 ida=0x832005
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v72 ida=0x90457d
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v79 ida=0x9558e5
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v83 ida=0xa09bdf
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v84 ida=0xa53fc1
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v87 ida=0xa9f48b
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v92 ida=0x9b5830
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v95 ida=0x9e08c0
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=jms_v185 ida=0xaee887
//
// v83/v84/v87/v92/jms_v185 addresses were resolved for this task by opening
// each IDB (idb_list, matched by binary name), func_query'ing
// SendBridleItemUseRequest (already named on v83/v87/v92/jms_185; unnamed as
// sub_A53FC1 on v84 — renamed live) and decompiling each hit to confirm the
// identical Encode4/Encode2/Encode4/Encode4 shape and the expected COutPacket
// opcode (0x51/81, 0x54/84, 0x58/88, 0x49/73 respectively, matching the
// pre-existing registry opcodes). Recorded in
// docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v92,gms_jms_185}.json.
func TestUseCatchItemBytes(t *testing.T) {
	input := NewUseCatchItem(0x11223344, 0x0005, 2270008, 0x07654321)

	want := []byte{
		0x44, 0x33, 0x22, 0x11, // updateTime      uint32 LE (Encode4 get_update_time)
		0x05, 0x00, // slot            int16  LE (Encode2 nPOS)
		0x38, 0xa3, 0x22, 0x00, // itemId          uint32 LE (Encode4 nItemID)
		0x21, 0x43, 0x65, 0x07, // monsterUniqueId uint32 LE (Encode4 hit-mob id)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("UseCatchItem %s layout mismatch\n got % x\nwant % x", v.Name, got, want)
			}
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestUseCatchItemDecode proves Decode recovers every field (the handler path).
func TestUseCatchItemDecode(t *testing.T) {
	input := NewUseCatchItem(0x11223344, 0x0005, 2270008, 0x07654321)
	ctx := pt.CreateContext("GMS", 83, 1)

	var out UseCatchItem
	pt.RoundTrip(t, ctx, input.Encode, out.Decode, nil)

	if out.UpdateTime() != 0x11223344 || out.Slot() != 5 ||
		out.ItemId() != 2270008 || out.MonsterUniqueId() != 0x07654321 {
		t.Fatalf("decoded %+v, want updateTime=0x11223344 slot=5 itemId=2270008 uniqueId=0x07654321", out)
	}
}
