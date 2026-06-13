package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoMp version=gms_v83 ida=0x58de53
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoMp version=gms_v87 ida=0x5bd318
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoMp version=gms_v95 ida=0x5688f0
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoMp version=jms_v185 ida=0x5e7a49
// packet-audit:verify packet=character/clientbound/CharacterKeyMapAutoMp version=gms_v84 ida=0x59de46
func TestCharacterKeyMapAutoMp(t *testing.T) {
	input := NewCharacterKeyMapAutoMp(2000002)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
