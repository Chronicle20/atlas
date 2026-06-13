package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v83 ida=0x9832e3
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v87 ida=0xa08d57
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v95 ida=0x954c50
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v84 ida=0x9c3681
func TestCharacterDamagePhysical(t *testing.T) {
	input := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
