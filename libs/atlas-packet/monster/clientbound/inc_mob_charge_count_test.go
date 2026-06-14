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
	input := NewIncMobChargeCount(0x0011AABB, 0x00000001)

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
