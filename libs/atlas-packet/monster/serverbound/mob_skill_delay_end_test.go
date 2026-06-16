package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v83 is VERSION-ABSENT (no CMob::Update skill-delay-end send site — feature
// post-dates v83; see structures/applicability.md). Markers cover v84/v87/v95/jms.
// packet-audit:verify packet=monster/serverbound/MonsterMobSkillDelayEnd version=gms_v84 ida=0x67d4ea
// packet-audit:verify packet=monster/serverbound/MonsterMobSkillDelayEnd version=gms_v87 ida=0x6a1c43
// packet-audit:verify packet=monster/serverbound/MonsterMobSkillDelayEnd version=gms_v95 ida=0x654300
// packet-audit:verify packet=monster/serverbound/MonsterMobSkillDelayEnd version=jms_v185 ida=0x6e3d2f
func TestMobSkillDelayEnd(t *testing.T) {
	input := MobSkillDelayEnd{mobCrc: 0xAABBCCDD, skillId: 0x0021FF01, skillLevel: 0x00000005, value: 0x00000190}

	// Golden bytes (v95). CMob::Update skill-delay-end send @0x6543d1:
	//   Encode4(SecureFuse(m_dwMobID)) -> mobCrc uint32 LE
	//   Encode4(m_nSkillID)            -> skillId uint32 LE
	//   Encode4(m_nSkillLevel)         -> skillLevel uint32 LE
	//   Encode4(m_nSkillOption)        -> value uint32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE = 0xAABBCCDD (Encode4 @0x6543d1)
		0x01, 0xFF, 0x21, 0x00, // skillId uint32 LE = 0x0021FF01 (Encode4 @0x6543d1)
		0x05, 0x00, 0x00, 0x00, // skillLevel uint32 LE = 5 (Encode4 @0x6543d1)
		0x90, 0x01, 0x00, 0x00, // value uint32 LE = 400 (Encode4 @0x6543d1)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobSkillDelayEnd layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
