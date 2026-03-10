package party

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestDisbandWRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewDisbandW(11, 5000, 300)
			output := DisbandW{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
		})
	}
}
