package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v48 ida=0x551c79
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v61 ida=0x5ccf1c
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v72 ida=0x61d358
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v79 ida=0x63d0e6
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v83 ida=0x66e146
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v84 ida=0x684492
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v87 ida=0x6a9061
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v92 ida=0x636320
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=gms_v95 ida=0x640d20
// packet-audit:verify packet=monster/serverbound/MonsterAutoAggro version=jms_v185 ida=0x6eba3c
func TestAutoAggro(t *testing.T) {
	input := NewAutoAggro(0xAABBCCDD, 0x00000027)

	// Golden bytes (v83 baseline). CMob::ApplyControl @0x66e146:
	//   Encode4(_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))  -> mobId  uint32 LE
	//   Encode4(n = |dx|/10 + |dy|/3 [+100])              -> distance uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobId    uint32 LE = 0xAABBCCDD (Encode4 @0x66e146)
		0x27, 0x00, 0x00, 0x00, // distance uint32 LE = 39         (Encode4 @0x66e146)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("AutoAggro layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}

	if input.MobId() != 0xAABBCCDD {
		t.Errorf("MobId() = %#x, want %#x", input.MobId(), uint32(0xAABBCCDD))
	}
	if input.Distance() != 0x27 {
		t.Errorf("Distance() = %d, want %d", input.Distance(), 0x27)
	}
	if input.Operation() != AutoAggroHandle {
		t.Errorf("Operation() = %q, want %q", input.Operation(), AutoAggroHandle)
	}
}
