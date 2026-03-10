package party

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestInviteWRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInviteW(16, 5000, "PartyLeader")
			output := InviteW{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.OriginatorName() != input.OriginatorName() {
				t.Errorf("originatorName: got %v, want %v", output.OriginatorName(), input.OriginatorName())
			}
		})
	}
}
