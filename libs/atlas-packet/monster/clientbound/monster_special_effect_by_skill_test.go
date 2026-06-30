package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v83 ida=0x66d8e7
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v84 ida=0x683be9
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v87 ida=0x6a87b3
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v95 ida=0x6540b0
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=jms_v185 ida=0x6eb08d
func TestMonsterSpecialEffectBySkill(t *testing.T) {
	input := NewMonsterSpecialEffectBySkill(0x07654321, 0x002F1801, 0x0000A1B2, 0x0190)

	// Golden bytes (v83 baseline). CMob::OnSpecialEffectBySkill @0x66d8e7 reads a
	// single Decode4 (skillId); the special UOL is resolved client-side from the
	// skill entry, so v83/v84/v87/jms carry only the one wire field.
	gotV83 := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	wantV83 := []byte{
		0x01, 0x18, 0x2F, 0x00, // skillId int32 LE = 0x002F1801 (Decode4 @0x66d8e7)
	}
	if !bytes.Equal(gotV83, wantV83) {
		t.Fatalf("MonsterSpecialEffectBySkill v83 layout mismatch\n got % x\nwant % x", gotV83, wantV83)
	}

	// Golden bytes (v95). CMob::OnSpecialEffectBySkill @0x6540b0:
	//   v4 = Decode4 -> skillId; v6 = Decode4 -> characterId (GetUser); v7 = Decode2 -> delay (tDelay).
	gotV95 := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	wantV95 := []byte{
		0x01, 0x18, 0x2F, 0x00, // skillId int32 LE = 0x002F1801 (Decode4 @0x6540b0)
		0xB2, 0xA1, 0x00, 0x00, // characterId int32 LE = 0x0000A1B2 (Decode4 @0x6540b0)
		0x90, 0x01, // delay uint16 LE = 0x0190 (Decode2 @0x6540b0)
	}
	if !bytes.Equal(gotV95, wantV95) {
		t.Fatalf("MonsterSpecialEffectBySkill v95 layout mismatch\n got % x\nwant % x", gotV95, wantV95)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterSpecialEffectBySkillBytesV79 pins the exact wire bytes against the
// v79 client read order. MONSTER_SPECIAL_EFFECT_BY_SKILL (op 225) is a per-mob
// OnMobPacket case: CMobPool::OnMobPacket @0x646d46 reads a uniqueId (Decode4
// @0x646d50) -> GetMob, THEN dispatches to CMob::OnSpecialEffectBySkill @0x63c887
// (GMS_v79_1_DEVM.exe, port 13340) which reads:
//
//	Decode4 @0x63c8a3 — skillId (special UOL resolved client-side from the skill
//	                    entry; no further wire reads). No characterId/delay (v95+).
//
// So the v79 wire is [uniqueId int32][skillId int32]. The leading uniqueId is the
// universal CMobPool::OnMobPacket prefix (see legacyMobPoolPrefix); written for the
// pre-v83 legacy range, gated off for v83+ (frozen per campaign).
//
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v79 ida=0x63c887
func TestMonsterSpecialEffectBySkillBytesV79(t *testing.T) {
	input := NewMonsterSpecialEffectBySkill(0x07654321, 0x002F1801, 0x0000A1B2, 0x0190)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x646d50)
		0x01, 0x18, 0x2F, 0x00, // skillId int32 LE = 0x002F1801 (Decode4 @0x63c8a3)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 specialEffectBySkill bytes:\n got % x\nwant % x", got, want)
	}
}
