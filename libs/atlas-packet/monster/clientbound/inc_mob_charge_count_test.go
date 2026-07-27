package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// INC_MOB_CHARGE_COUNT present in v83/v84/v87/v95 (dispatcher cases 0xFE/260/0x10E/302).
// jms has NO INC_MOB_CHARGE_COUNT dispatcher case → version-absent (no marker).
// packet-audit:verify packet=monster/clientbound/MonsterIncMobChargeCount version=gms_v83 ida=0x6710fc
// packet-audit:verify packet=monster/clientbound/MonsterIncMobChargeCount version=gms_v84 ida=0x687655
// packet-audit:verify packet=monster/clientbound/MonsterIncMobChargeCount version=gms_v87 ida=0x6ac230
// packet-audit:verify packet=monster/clientbound/MonsterIncMobChargeCount version=gms_v95 ida=0x63d500
func TestIncMobChargeCount(t *testing.T) {
	input := NewIncMobChargeCount(0x07654321, 0x0011AABB, 0x00000001)

	// Golden bytes (v83 baseline). CMob::OnIncMobChargeCount @0x6710fc:
	//   m_nMobChargeCount = Decode4 -> chargeCount int32 LE
	//   m_bAttackReady    = Decode4 -> attackReady int32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xBB, 0xAA, 0x11, 0x00, // chargeCount int32 LE = 0x0011AABB
		0x01, 0x00, 0x00, 0x00, // attackReady int32 LE = 1
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("IncMobChargeCount layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestIncMobChargeCountBytesV79 pins the exact wire bytes against the v79 client
// read order. INC_MOB_CHARGE_COUNT (op 232) is a per-mob OnMobPacket case:
// CMobPool::OnMobPacket @0x646d46 reads a uniqueId (Decode4 @0x646d50) -> GetMob,
// THEN dispatches to CMob::OnIncMobChargeCount (sub_640081 @0x640081,
// GMS_v79_1_DEVM.exe, port 13340) which reads:
//
//	Decode4 @0x640091 — m_nMobChargeCount (chargeCount int32)
//	Decode4 @0x640097 — m_bAttackReady    (attackReady int32)
//
// So the v79 wire is [uniqueId int32][chargeCount int32][attackReady int32]. The
// leading uniqueId is the universal CMobPool::OnMobPacket prefix (see
// legacyMobPoolPrefix); written for the pre-v83 legacy range, gated off for v83+.
//
// packet-audit:verify packet=monster/clientbound/MonsterIncMobChargeCount version=gms_v79 ida=0x640081
func TestIncMobChargeCountBytesV79(t *testing.T) {
	input := NewIncMobChargeCount(0x07654321, 0x0011AABB, 0x00000001)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x646d50)
		0xBB, 0xAA, 0x11, 0x00, // chargeCount int32 LE = 0x0011AABB (Decode4 @0x640091)
		0x01, 0x00, 0x00, 0x00, // attackReady int32 LE = 1 (Decode4 @0x640097)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 incMobChargeCount bytes:\n got % x\nwant % x", got, want)
	}
}
