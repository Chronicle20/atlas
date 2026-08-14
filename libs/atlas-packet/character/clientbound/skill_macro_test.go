package clientbound

import (
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// fixture builds the shared two-entry model used by every macro fixture in
// task-226. Kept identical to baseline-bytes.md's fixture so the v83/v84 rows
// are directly comparable to the pre-change capture.
func fixture() SkillMacro {
	return NewSkillMacro(
		NewSkillMacroEntry("Buff", true, skill.Id(1001003), skill.Id(1001004), skill.Id(0)),
		NewSkillMacroEntry("Attack", false, skill.Id(1001005), skill.Id(0), skill.Id(0)),
	)
}

// TestSkillMacroByteOutput verifies MACRO_SYS_DATA_INIT byte output against the
// client's CWvsContext::OnMacroSysDataInit read order (see layout-derivation.md).
//
// Wire layout: count(1) + per entry { name(2+len) + shout(1) + skillId1(4) +
// skillId2(4) + skillId3(4) }.
// Fixture: 1 + (2+4+1+12) + (2+6+1+12) = 41 bytes.
//
// Shout polarity is INVERTED (wire 0 = shout on) per layout-derivation.md's
// "Shout polarity" verdict, independently re-derived across all nine
// populated versions with zero deviation; no version gate is required.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v61 ida=0x849bce
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v72 ida=0x92126b
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v79 ida=0x97311a
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v83 ida=0xa290f8
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v84 ida=0xa748bb
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v87 ida=0xac0d6e
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v92 ida=0x9c5390
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=gms_v95 ida=0x9f0c70
// packet-audit:verify packet=character/clientbound/CharacterSkillMacro version=jms_v185 ida=0xb10384
//
// count=2
// entry0: len=0x0004 "Buff"   shout=00 (INVERTED: true->0) 1001003=0x000f462b 1001004=0x000f462c 0
// entry1: len=0x0006 "Attack" shout=01 (INVERTED: false->1) 1001005=0x000f462d 0 0
const wantUniform = "02" +
	"0400" + "42756666" + "00" + "2b460f00" + "2c460f00" + "00000000" +
	"0600" + "41747461636b" + "01" + "2d460f00" + "00000000" + "00000000"

func TestSkillMacroByteOutput(t *testing.T) {
	cases := []struct {
		variant pt.TenantVariant
		want    string
	}{
		{pt.Variants[8], wantUniform},  // GMS v61
		{pt.Variants[9], wantUniform},  // GMS v72
		{pt.Variants[10], wantUniform}, // GMS v79
		{pt.Variants[1], wantUniform},  // GMS v83
		{pt.Variants[5], wantUniform},  // GMS v84
		{pt.Variants[2], wantUniform},  // GMS v87
		{pt.Variants[11], wantUniform}, // GMS v92
		{pt.Variants[3], wantUniform},  // GMS v95
		{pt.Variants[4], wantUniform},  // JMS v185
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			got := hex.EncodeToString(fixture().Encode(nil, ctx)(nil))
			if got != tc.want {
				t.Errorf("bytes:\n got %s\nwant %s", got, tc.want)
			}
		})
	}
}
