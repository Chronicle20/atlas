package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_SKILL_DELAY present in v84/v87/v95/jms (dispatcher cases 261/0x10F/303/0x10F).
// VERSION-ABSENT in v83: the v83 CMobPool::OnMobPacket @0x67936d has no skill-delay
// case (switch ends at 0xFF OnMobAttackedByMob). No v83 marker.
// packet-audit:verify packet=monster/clientbound/MonsterMobSkillDelay version=gms_v84 ida=0x688524
// packet-audit:verify packet=monster/clientbound/MonsterMobSkillDelay version=gms_v87 ida=0x6ad0e8
// packet-audit:verify packet=monster/clientbound/MonsterMobSkillDelay version=gms_v95 ida=0x63d560
// packet-audit:verify packet=monster/clientbound/MonsterMobSkillDelay version=jms_v185 ida=0x6ef0d4
func TestMobSkillDelay(t *testing.T) {
	input := NewMobSkillDelay(0x000003E8, 0x0021FF01, 0x00000005, 0x00000002)

	// Golden bytes (v95). CMob::OnMobSkillDelay @0x63d560:
	//   m_delaySkill.tSkillDelayTime = Decode4 -> delay int32 LE
	//   m_delaySkill.nSkillID        = Decode4 -> skillId int32 LE
	//   m_delaySkill.nSLV            = Decode4 -> skillLevel int32 LE
	//   m_delaySkill.nOption         = Decode4 -> option int32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0xE8, 0x03, 0x00, 0x00, // delay int32 LE = 1000
		0x01, 0xFF, 0x21, 0x00, // skillId int32 LE = 0x0021FF01
		0x05, 0x00, 0x00, 0x00, // skillLevel int32 LE = 5
		0x02, 0x00, 0x00, 0x00, // option int32 LE = 2
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobSkillDelay layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
