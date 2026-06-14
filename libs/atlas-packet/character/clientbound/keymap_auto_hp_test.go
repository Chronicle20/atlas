package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoHp version=gms_v83 ida=0x58de2d
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoHp version=gms_v87 ida=0x5bd2f2
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoHp version=gms_v95 ida=0x5688c0
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoHp version=jms_v185 ida=0x5e7a23
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoHp version=gms_v84 ida=0x59de20
func TestCharacterKeyMapAutoHp(t *testing.T) {
	input := NewCharacterKeyMapAutoHp(2000001)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
