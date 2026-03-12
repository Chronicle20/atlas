package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestInviteRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InviteRequest{target: "InvitedPlayer"}
			output := InviteRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Target() != input.Target() {
				t.Errorf("target: got %v, want %v", output.Target(), input.Target())
			}
		})
	}
}
