package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v95 ida=0x52e9e0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v84 ida=0x53bc2a
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v83 ida=0x52fad4
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v87 ida=0x556cfe
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=jms_v185 ida=0x56c859
func TestOperationInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationInvite{targetCharacterId: 12345}
			output := OperationInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}
