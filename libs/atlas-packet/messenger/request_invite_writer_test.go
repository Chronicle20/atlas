package messenger

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMessengerRequestInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerRequestInvite(5, "TestPlayer", 12345)
			output := RequestInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.MessengerId() != input.MessengerId() {
				t.Errorf("messengerId: got %v, want %v", output.MessengerId(), input.MessengerId())
			}
		})
	}
}
