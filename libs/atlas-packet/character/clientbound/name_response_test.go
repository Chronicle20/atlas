package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v83 ida=0x5f9c72
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v84 ida=0x60eca7
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v87 ida=0x63153b
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v95 ida=0x5d5790
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=jms_v185 ida=0x66f957
func TestCharacterNameResponseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterNameResponse{name: "TestChar", code: 0}
			output := CharacterNameResponse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
