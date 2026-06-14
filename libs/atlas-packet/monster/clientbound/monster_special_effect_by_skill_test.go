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
	input := NewMonsterSpecialEffectBySkill(0x002F1801, 0x0000A1B2, 0x0190)

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
