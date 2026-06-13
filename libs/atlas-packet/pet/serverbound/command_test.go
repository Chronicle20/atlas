package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v83 ida=0x704d5d
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v87 ida=0x748a35
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v95 ida=0x6a3cc0
// packet-audit:verify packet=pet/serverbound/PetCommand version=jms_v185 ida=0x76abe0
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v84 ida=0x7214bf
func TestCommandRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Command{petId: 12345, byName: true, command: 3}
			output := Command{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.ByName() != input.ByName() {
				t.Errorf("byName: got %v, want %v", output.ByName(), input.ByName())
			}
			if output.Command() != input.Command() {
				t.Errorf("command: got %v, want %v", output.Command(), input.Command())
			}
		})
	}
}
