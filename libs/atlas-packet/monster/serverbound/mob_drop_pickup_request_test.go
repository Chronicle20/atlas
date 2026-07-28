package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v84 sender CMob::SendDropPickUpRequest was unnamed in the v84 IDB; task-092
// Stage 4 located + named it (@0x684cdf, COutPacket(0xC3) — opcode corrected from
// the csv-stale 0xBE; Encode4(fused mobId)+Encode4(dropObjId)) and pinned v84.
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v83 ida=0x66e91f
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v84 ida=0x684cdf
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v87 ida=0x6a98ae
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v95 ida=0x644450
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=jms_v185 ida=0x6ec289
func TestMobDropPickupRequest(t *testing.T) {
	input := MobDropPickupRequest{mobCrc: 0xAABBCCDD, dropId: 0x01020304}

	// Golden bytes (v83 baseline). CMob::SendDropPickUpRequest @0x66e91f:
	//   Encode4(_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))  -> mobCrc uint32 LE
	//   Encode4(dwDropID)                                 -> dropId uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x66e91f)
		0x04, 0x03, 0x02, 0x01, // dropId uint32 LE = 0x01020304 (Encode4 @0x66e91f)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobDropPickupRequest layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMobDropPickupRequestBytesV79 pins the exact wire bytes against the v79
// client send order. CMob::SendDropPickUpRequest is the unnamed sub_63D8BF
// @0x63d8bf (GMS_v79_1_DEVM.exe, port 13340), opcode 182:
//
//	COutPacket(182) @0x63d939
//	Encode4 @0x63d95a — fused mob id (sub_4DC1C0(this+380, m_dwMobID)) -> mobCrc
//	Encode4 @0x63d963 — a2 (dwDropID)                                 -> dropId
//
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v79 ida=0x63d8bf
func TestMobDropPickupRequestBytesV79(t *testing.T) {
	input := MobDropPickupRequest{mobCrc: 0xAABBCCDD, dropId: 0x01020304}
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE (Encode4 @0x63d95a)
		0x04, 0x03, 0x02, 0x01, // dropId uint32 LE (Encode4 @0x63d963)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 mobDropPickupRequest bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobDropPickupRequestBytesV72 pins the v72 wire. MOB_DROP_PICKUP_REQUEST is
// sub_61DB0D @0x61db0d (GMS_v72.1_U_DEVM.exe, port 13339), opcode 180:
//
//	COutPacket(180) @0x61db87
//	Encode4 @0x61dba8 — fused mob id -> mobCrc
//	Encode4 @0x61dbb1 — a2 (dwDropID) -> dropId
//
// Byte-identical to v79.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v72 ida=0x61db0d
func TestMobDropPickupRequestBytesV72(t *testing.T) {
	input := MobDropPickupRequest{mobCrc: 0xAABBCCDD, dropId: 0x01020304}
	ctx := pt.CreateContext("GMS", 72, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE (Encode4 @0x61dba8)
		0x04, 0x03, 0x02, 0x01, // dropId uint32 LE (Encode4 @0x61dbb1)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 mobDropPickupRequest bytes:\n got % x\nwant % x", got, want)
	}
}
