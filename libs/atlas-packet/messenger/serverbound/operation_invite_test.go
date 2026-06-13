package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/serverbound/MessengerOperationInvite version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/serverbound/MessengerOperationInvite version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/serverbound/MessengerOperationInvite version=gms_v95 ida=0x7f5820
// packet-audit:verify packet=messenger/serverbound/MessengerOperationInvite version=jms_v185 ida=0x8e4e8a
func TestOperationInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationInvite{targetCharacter: "TestPlayer"}
			output := OperationInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != input.TargetCharacter() {
				t.Errorf("targetCharacter: got %v, want %v", output.TargetCharacter(), input.TargetCharacter())
			}
		})
	}
}
