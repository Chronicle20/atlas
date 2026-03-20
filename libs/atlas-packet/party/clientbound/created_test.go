package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCreatedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCreated(7, 12345)
			output := Created{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
		})
	}
}
