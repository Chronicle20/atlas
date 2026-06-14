package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v84 sender CMob::TryFirstSelfDestruction was unnamed in the v84 IDB; task-092
// Stage 4 located + named it (@0x6849ee, COutPacket(0xC6), single Encode4 of the
// fused mob id — v83-identical wire shape) and pinned v84 evidence.
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v83 ida=0x66e636
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v84 ida=0x6849ee
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v87 ida=0x6a95bd
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v95 ida=0x640ee0
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=jms_v185 ida=0x6ebf98
func TestMonsterBomb(t *testing.T) {
	input := MonsterBomb{mobId: 0xAABBCCDD}

	// Golden bytes (v83 baseline). CMob::TryFirstSelfDestruction @0x66e636 (opcode 0xC1):
	//   Encode4(SecureFuse(m_dwMobID)) -> mobId uint32 LE — the only wire field.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE = 0xAABBCCDD (Encode4 @0x66e636)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MonsterBomb layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
