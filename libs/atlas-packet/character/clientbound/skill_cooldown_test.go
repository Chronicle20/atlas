package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterSkillCooldown version=gms_v83 ida=0x95be66
// packet-audit:verify packet=character/clientbound/CharacterSkillCooldown version=gms_v87 ida=0x9de54b
// packet-audit:verify packet=character/clientbound/CharacterSkillCooldown version=gms_v95 ida=0x908b90
// packet-audit:verify packet=character/clientbound/CharacterSkillCooldown version=jms_v185 ida=0xa2747f
func TestCharacterSkillCooldown(t *testing.T) {
	input := NewCharacterSkillCooldown(1001003, 30)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
