package model

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// sampleAttackInfo builds a representative client->server attack request. skillId
// is 0 (a plain weapon swing) so the keydown/charging and special-skill
// (NightWalker/ThunderBreaker) branches stay quiet and the structure is driven
// purely by attackType + tenant version.
func sampleAttackInfo(at AttackType) *AttackInfo {
	ai := NewAttackInfo(at)
	ai.SetHits(2)
	ai.SetDamage(1)
	ai.SetSkillId(0)
	ai.SetOption(0x10)
	ai.SetLeft(true)
	ai.SetAttackAction(0x05)
	ai.SetActionSpeed(4)
	di := NewDamageInfo(2)
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{1000, 2000})
	ai.AddDamageInfo(*di)
	if at == AttackTypeRanged {
		ai.SetBulletPosition(100, 200)
	}
	return ai
}

// TestAttackInfoRoundTrip pins Encode/Decode symmetry for every attack type
// across all tenant variants. RoundTrip fails if any byte is left unconsumed,
// which is exactly what a version-gate drift between Encode and Decode produces
// (e.g. the primary dr-block must be present on BOTH sides for GMS v84+).
func TestAttackInfoRoundTrip(t *testing.T) {
	types := []struct {
		name string
		at   AttackType
	}{
		{"Melee", AttackTypeMelee},
		{"Ranged", AttackTypeRanged},
		{"Magic", AttackTypeMagic},
		{"Energy", AttackTypeEnergy},
	}
	for _, v := range pt.Variants {
		for _, ty := range types {
			t.Run(v.Name+"/"+ty.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				ai := sampleAttackInfo(ty.at)
				pt.RoundTrip(t, ctx, ai.Encode, ai.Decode, nil)
			})
		}
	}
}

// TestAttackInfoVersionBoundary pins the corrected dr-block boundary. The
// primary damage-randomizer block (dr0/dr1/dr2/dr3/randomDr/crc32 = 6x uint32)
// is present GMS v84+, NOT v95+ (the bug this fixes was a >=95 gate that left
// v84 attacks reading skillId from the wrong offset -> 0xFFFFFFFF). v84..v94 are
// identical; v95 adds skillLevel(1) + anotherCrc(4) + a per-type int(4) = +9.
func TestAttackInfoVersionBoundary(t *testing.T) {
	enc := func(major uint16, at AttackType) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		ai := sampleAttackInfo(at)
		return pt.Encode(t, ctx, ai.Encode, nil)
	}

	v83 := enc(83, AttackTypeMelee)
	v84 := enc(84, AttackTypeMelee)
	if len(v84) != len(v83)+24 {
		t.Errorf("v84 melee (%d) must be v83 (%d) + 24 bytes (primary dr-block: 6x uint32)", len(v84), len(v83))
	}
	for _, major := range []uint16{85, 86, 87, 94} {
		if got := enc(major, AttackTypeMelee); len(got) != len(v84) {
			t.Errorf("v%d melee (%d) must equal v84 (%d): no structure change until v95", major, len(got), len(v84))
		}
	}
	if v95 := enc(95, AttackTypeMelee); len(v95) != len(v84)+9 {
		t.Errorf("v95 melee (%d) must be v84 (%d) + 9 bytes (skillLevel + anotherCrc + per-type int)", len(v95), len(v84))
	}
}
