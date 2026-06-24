package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v83 ida=0x9724f9
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v87 ida=0x9f74de
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v95 ida=0x949240
func TestCharacterChairShow(t *testing.T) {
	input := NewCharacterChairShow(1234, 3010000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
