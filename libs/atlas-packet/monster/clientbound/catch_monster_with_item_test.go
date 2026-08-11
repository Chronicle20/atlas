package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v83 ida=0x66d997
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v84 ida=0x683c9f
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v87 ida=0x6a886e
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v92 ida=0x630c50
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v95 ida=0x63cd40
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=jms_v185 ida=0x6eb148
func TestCatchMonsterWithItem(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)

	// v83 baseline: CMobPool::OnMobPacket @0x67936d Decode4 -> GetMob, then
	// CMob::OnEffectByItem @0x66d997 reads Decode4 itemId + Decode1 result.
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x67936d)
		0x38, 0xa3, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x66d997)
		0x01, // result   byte  (Decode1 @0x66d997)
	}
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, want) {
		t.Fatalf("CatchMonsterWithItem v83 layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestCatchMonsterWithItemBytesV48 pins the v48 SHORT body. CMobPool::OnMobPacket
// @0x559390 Decode4 -> GetMob dispatches case 173 to sub_551481, which reads
// ONLY Decode4 (the item id) and passes it to sub_54E82D — there is no trailing
// result byte on v48. Every later version reads Decode4 + Decode1 (v61
// @0x5cc793, v79 @0x63c937, v92 @0x630c50). design.md §2 F-2.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v48 ida=0x551481
func TestCatchMonsterWithItemBytesV48(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 48, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x559390)
		0x38, 0xa3, 0x22, 0x00, // itemId   int32 LE (Decode4, sub_551481)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterWithItemBytesV61 pins the first version that DOES carry the
// trailing result byte — the boundary the v48CatchByItemNoResult gate encodes.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v61 ida=0x5cc793
func TestCatchMonsterWithItemBytesV61(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 61, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x5d48f3)
		0x38, 0xa3, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x5cc793)
		0x01, // result   byte  (Decode1 @0x5cc793)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterWithItemBytesV79 pins the v79 cell (fixture promotion).
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v79 ida=0x63c937
func TestCatchMonsterWithItemBytesV79(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x646d46)
		0x38, 0xa3, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x63c937)
		0x01, // result   byte  (Decode1 @0x63c937)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterWithItemBytesV72 pins the v72 cell (fixture promotion,
// design.md §3). CMobPool::OnMobPacket @0x62560d Decode4 @0x625617 -> GetMob,
// then dispatches to CMob::OnEffectByItem @0x61cbcc: Decode4 (itemId) +
// Decode1 (result) — same shape as v79/v61, byte-identical.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v72 ida=0x61cbcc
func TestCatchMonsterWithItemBytesV72(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 72, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x625617)
		0x38, 0xa3, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x61cbcc)
		0x01, // result   byte  (Decode1 @0x61cbcc)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}
