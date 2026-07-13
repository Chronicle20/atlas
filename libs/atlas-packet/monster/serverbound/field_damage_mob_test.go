package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v83 ida=0x6675a8
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v84 ida=0x67d4ea
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v87 ida=0x6a1c43
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v95 ida=0x654300
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=jms_v185 ida=0x6e3d2f
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v72 ida=0x616dd0
func TestFieldDamageMob(t *testing.T) {
	input := FieldDamageMob{mobCrc: 0xAABBCCDD, damage: 0x000003E7}

	// Golden bytes (v83 baseline). CMob::Update field-damage send @0x667d39:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	//   Encode4(nFieldDamage)          -> damage uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x667d39)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE = 999 (Encode4 @0x667d39)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FieldDamageMob layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestFieldDamageMobV72 pins the FIELD_DAMAGE_MOB (send op 181) wire against
// CMob::Update (v72 sub_616DD0 @0x616dd0), field-damage send site @0x6174d0:
//
//	COutPacket(181)                                                 /*0x6174d0*/
//	Encode4(SecureFuse(this.m_dwMobID))    // mobCrc                 /*0x6174f4*/
//	Encode4(nFieldDamage)                  // damage (from sub_5F3CD7) /*0x6174ff*/
//
// Two Encode4, no version gate — byte-identical to the v83 golden fixture.
// op 181 = v79 op 183 - 2 (deep-cluster Δ-2). The shared CMob::Update export
// entry captures the sibling MOB_DAMAGE_MOB_FRIENDLY 3-int send, so this codec's
// 2-int layout diffs the flat report (advisory ❌); this byte fixture is the
// ground truth per the tier-1 rule.
func TestFieldDamageMobV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := FieldDamageMob{mobCrc: 0xAABBCCDD, damage: 0x000003E7}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x6174f4)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE = 999 (Encode4 @0x6174ff)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 FieldDamageMob layout mismatch\n got % x\nwant % x", got, want)
	}
}
