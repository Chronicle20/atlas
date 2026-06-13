package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v83 ida=0xa1e48c
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v87 ida=0xab57c5
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v95 ida=0x9f5f30
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=jms_v185 ida=0xb04ff3
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v84 ida=0xa6972b
func TestCharacterSkillChange(t *testing.T) {
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
