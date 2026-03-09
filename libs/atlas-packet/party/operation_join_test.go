package party

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

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
