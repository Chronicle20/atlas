package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v83 ida=0x670c63
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v84 ida=0x6871bc
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v87 ida=0x6abd95
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v95 ida=0x64b260
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=jms_v185 ida=0x6edce8
func TestMobDamageMob(t *testing.T) {
	input := MobDamageMob{
		attackerMobId: 0x11223344,
		characterId:   0x0010F447,
		mobId:         0xAABBCCDD,
		attackIndex:   0x03,
		damage:        0x000003E7,
		reflect:       0x01,
		x:             0x0102,
		y:             0x0304,
	}

	// Golden bytes (v83 baseline). CMob::SetDamagedByMob @0x670c63 (opcode 0xC2):
	//   Encode4(GetMobID(attacker)) -> attackerMobId; Encode4(characterId);
	//   Encode4(GetMobID(this)) -> mobId; Encode1(nAttackIdx); Encode4(damage);
	//   Encode1(nDir<0) -> reflect; Encode2(xCenter); Encode2(yCenter)
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerMobId uint32 LE = 0x11223344
		0x47, 0xF4, 0x10, 0x00, // characterId uint32 LE = 0x0010F447
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE = 0xAABBCCDD
		0x03,                   // attackIndex byte = 3 (Encode1)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE = 999 (Encode4)
		0x01,       // reflect byte = 1 (Encode1)
		0x02, 0x01, // x uint16 LE = 0x0102 (Encode2)
		0x04, 0x03, // y uint16 LE = 0x0304 (Encode2)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobDamageMob layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMobDamageMobBytesV79 pins the exact wire bytes against the v79 client send
// order. CMob::SetDamagedByMob is the unnamed sub_63FBE8 @0x63fbe8
// (GMS_v79_1_DEVM.exe, port 13340), opcode 186:
//
//	COutPacket(186) @0x63fe07
//	Encode4 @0x63fe2b — fused attacker mob id (sub_4DC1C0(a5+95, a5[97])) -> attackerMobId
//	Encode4 @0x63fe3e — *(g_pWvsContext+8352)                            -> characterId
//	Encode4 @0x63fe5b — fused victim mob id (sub_4DC1C0(this+95, this[97])) -> mobId
//	Encode1 @0x63fe66 — a6 (nAttackIdx)                                  -> attackIndex
//	Encode4 @0x63fe71 — damage (v45)                                     -> damage
//	Encode1 @0x63fe81 — (a7 < 0)                                         -> reflect
//	Encode2 @0x63fe8a — xCenter (v23)                                    -> x
//	Encode2 @0x63fe95 — yCenter (v40)                                    -> y
//
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v79 ida=0x63fbe8
func TestMobDamageMobBytesV79(t *testing.T) {
	input := MobDamageMob{
		attackerMobId: 0x11223344,
		characterId:   0x0010F447,
		mobId:         0xAABBCCDD,
		attackIndex:   0x03,
		damage:        0x000003E7,
		reflect:       0x01,
		x:             0x0102,
		y:             0x0304,
	}
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerMobId uint32 LE (Encode4 @0x63fe2b)
		0x47, 0xF4, 0x10, 0x00, // characterId uint32 LE (Encode4 @0x63fe3e)
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x63fe5b)
		0x03,                   // attackIndex byte (Encode1 @0x63fe66)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE (Encode4 @0x63fe71)
		0x01,       // reflect byte (Encode1 @0x63fe81)
		0x02, 0x01, // x uint16 LE (Encode2 @0x63fe8a)
		0x04, 0x03, // y uint16 LE (Encode2 @0x63fe95)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 mobDamageMob bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobDamageMobBytesV72 pins the v72 wire. MOB_DAMAGE_MOB is sub_61F2AB
// @0x61f2ab (GMS_v72.1_U_DEVM.exe, port 13339), opcode 184:
//
//	COutPacket(184) @0x61f4ca
//	Encode4 @0x61f4ee — fused attacker mob id -> attackerMobId
//	Encode4 @0x61f501 — *(g_pWvsContext+8352)  -> characterId
//	Encode4 @0x61f51e — fused victim mob id    -> mobId
//	Encode1 @0x61f529 — a6 (nAttackIdx)        -> attackIndex
//	Encode4 @0x61f534 — damage (v45)           -> damage
//	Encode1 @0x61f544 — (a7 < 0)               -> reflect
//	Encode2 @0x61f54d — xCenter                -> x
//	Encode2 @0x61f558 — yCenter                -> y
//
// Byte-identical to v79.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v72 ida=0x61f2ab
func TestMobDamageMobBytesV72(t *testing.T) {
	input := MobDamageMob{
		attackerMobId: 0x11223344,
		characterId:   0x0010F447,
		mobId:         0xAABBCCDD,
		attackIndex:   0x03,
		damage:        0x000003E7,
		reflect:       0x01,
		x:             0x0102,
		y:             0x0304,
	}
	ctx := pt.CreateContext("GMS", 72, 1)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerMobId uint32 LE (Encode4 @0x61f4ee)
		0x47, 0xF4, 0x10, 0x00, // characterId uint32 LE (Encode4 @0x61f501)
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x61f51e)
		0x03,                   // attackIndex byte (Encode1 @0x61f529)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE (Encode4 @0x61f534)
		0x01,       // reflect byte (Encode1 @0x61f544)
		0x02, 0x01, // x uint16 LE (Encode2 @0x61f54d)
		0x04, 0x03, // y uint16 LE (Encode2 @0x61f558)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 mobDamageMob bytes:\n got % x\nwant % x", got, want)
	}
}
