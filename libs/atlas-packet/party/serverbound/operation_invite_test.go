package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/serverbound/PartyOperationInvite version=gms_v95 ida=0x534310
// packet-audit:verify packet=party/serverbound/PartyOperationInvite version=jms_v185 ida=0x56cce9
// packet-audit:verify packet=party/serverbound/PartyOperationInvite version=gms_v87 ida=0x5570df
// packet-audit:verify packet=party/serverbound/PartyOperationInvite version=gms_v83 ida=0x52fecf
func TestOperationInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationInvite{name: "InviteTarget"}
			output := OperationInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
