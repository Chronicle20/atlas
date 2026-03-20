package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBuddyInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyInvite(9, 1000, 2000, "TestPlayer")
			output := Invite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.OriginatorId() != input.OriginatorId() {
				t.Errorf("originatorId: got %v, want %v", output.OriginatorId(), input.OriginatorId())
			}
			if output.OriginatorName() != input.OriginatorName() {
				t.Errorf("originatorName: got %v, want %v", output.OriginatorName(), input.OriginatorName())
			}
			if output.ActorId() != input.ActorId() {
				t.Errorf("actorId: got %v, want %v", output.ActorId(), input.ActorId())
			}
		})
	}
}
