package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMessengerInviteDeclinedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerInviteDeclined(7, "TestPlayer", 3)
			output := InviteDeclined{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.DeclineMode() != input.DeclineMode() {
				t.Errorf("declineMode: got %v, want %v", output.DeclineMode(), input.DeclineMode())
			}
		})
	}
}
