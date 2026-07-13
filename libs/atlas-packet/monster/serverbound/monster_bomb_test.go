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

// TestMonsterBombBytesV79 pins the exact wire bytes against the v79 client send
// order. CMob::TryFirstSelfDestruction is the unnamed sub_63D5D6 @0x63d5d6
// (GMS_v79_1_DEVM.exe, port 13340), opcode 185:
//
//	COutPacket(185) @0x63d6ff
//	Encode4 @0x63d720 — fused mob id (sub_4DC1C0(this+95, m_dwMobID)) -> mobId
//
// Exactly one wire field. Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v79 ida=0x63d5d6
func TestMonsterBombBytesV79(t *testing.T) {
	input := MonsterBomb{mobId: 0xAABBCCDD}
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x63d720)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 monsterBomb bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterBombBytesV72 pins the v72 wire. MONSTER_BOMB is sub_61D837
// @0x61d837 (GMS_v72.1_U_DEVM.exe, port 13339), opcode 183:
//
//	COutPacket(183) @0x61d95d
//	Encode4 @0x61d97e — fused mob id -> mobId
//
// Exactly one wire field. Byte-identical to v79.
//
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v72 ida=0x61d837
func TestMonsterBombBytesV72(t *testing.T) {
	input := MonsterBomb{mobId: 0xAABBCCDD}
	ctx := pt.CreateContext("GMS", 72, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x61d97e)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 monsterBomb bytes:\n got % x\nwant % x", got, want)
	}
}
