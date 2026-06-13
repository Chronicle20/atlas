package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/serverbound/PartyOperationChangeLeader version=gms_v95 ida=0x530370
// packet-audit:verify packet=party/serverbound/PartyOperationChangeLeader version=jms_v185 ida=0x56d0cc
// packet-audit:verify packet=party/serverbound/PartyOperationChangeLeader version=gms_v87 ida=0x5574c9
func TestOperationChangeLeaderRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationChangeLeader{targetCharacterId: 300}
			output := OperationChangeLeader{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}
