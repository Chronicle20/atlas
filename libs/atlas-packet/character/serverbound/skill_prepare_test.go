package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// buildSampleSkillPrepare mirrors model.sampleSkillPrepareInfo: a standard keydown
// skill (Hurricane 3121004) so the swallowMobId branch (skillId == 33101005) stays
// quiet and the wire structure is driven purely by tenant version.
func buildSampleSkillPrepare() model.SkillPrepareInfo {
	m := model.NewSkillPrepareInfo()
	m.SetSkillId(3121004)
	m.SetLevel(10)
	m.SetAction(0x0142)
	m.SetActionSpeed(4)
	return *m
}

// TestSkillPrepareRoundTrip pins that the serverbound SkillPrepare wrapper delegates
// symmetrically to the shared model.SkillPrepareInfo codec across all tenant variants.
// The model itself (incl. the swallowMobId branch) is production-tested in
// model/skill_prepare_info_test.go.
func TestSkillPrepareRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := SkillPrepare{info: buildSampleSkillPrepare()}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// TestSkillPrepareByteFixture asserts the exact encoded bytes of the serverbound
// SkillPrepare wrapper match wire-spec §1 field order:
// skillId u32 LE, level u8, action u16 LE, actionSpeed u8. The sample skill
// (Hurricane 3121004) is NOT the swallow skill (33101005), so the trailing
// conditional swallowMobId u32 is absent on every version — all five encode
// identically. Bytes trace to CUserLocal::DoActiveSkill_Prepare's COutPacket
// writes (skillId/level/action/actionSpeed) per wire-spec.md (IDB-verified).
//
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v83 ida=0x96a86e
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v84 ida=0x9a9761
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v87 ida=0x9ee1e6
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v95 ida=0x941710
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=jms_v185 ida=0xa39cfd
func TestSkillPrepareByteFixture(t *testing.T) {
	// skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00)
	// level=10 (0x0A)
	// action=0x0142 LE = 42 01
	// actionSpeed=4 (0x04)
	expected := []byte{
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
		0x0A,       // level=10
		0x42, 0x01, // action=0x0142 LE
		0x04, // actionSpeed=4
	}

	versions := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}

	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			m := SkillPrepare{info: buildSampleSkillPrepare()}
			got := pt.Encode(t, ctx, m.Encode, nil)
			if len(got) != len(expected) {
				t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X",
					len(got), len(expected), got, expected)
			}
			for i := range expected {
				if got[i] != expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X",
						i, got[i], expected[i], got, expected)
					break
				}
			}
		})
	}
}
