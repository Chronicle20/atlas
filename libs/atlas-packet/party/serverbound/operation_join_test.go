package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/serverbound/PartyOperationJoin version=gms_v95 ida=0x534310
// packet-audit:verify packet=party/serverbound/PartyOperationJoin version=jms_v185 ida=0x56cce9
func TestOperationJoinRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationJoin{partyId: 100}
			output := OperationJoin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
		})
	}
}
