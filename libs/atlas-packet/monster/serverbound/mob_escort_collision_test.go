package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_COLLISION present in v95 (opcode 236) + jms (opcode 0xCB). Absent in
// v83/v84/v87 (no escort family; the v87 registry row is stale — removed).
// packet-audit:verify packet=monster/serverbound/MonsterMobEscortCollision version=gms_v95 ida=0x641150
// packet-audit:verify packet=monster/serverbound/MonsterMobEscortCollision version=jms_v185 ida=0x6efeb7
func TestMobEscortCollision(t *testing.T) {
	input := MobEscortCollision{mobCrc: 0xAABBCCDD, dest: 0x00000005}

	// Golden bytes (v95). CMob::SendCollisionEscort @0x641150:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	//   Encode4(nDest)                 -> dest   uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD
		0x05, 0x00, 0x00, 0x00, // dest uint32 LE = 5
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortCollision layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
