package interaction

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestInteractionInviteRoundTrip(t *testing.T) {
	input := NewInteractionInvite(4, 1, "TestPlayer", 12345)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionInvite{}).Decode, nil)
		})
	}
}

func TestInteractionInviteResultRoundTrip(t *testing.T) {
	input := NewInteractionInviteResult(5, 1, "Room is full")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionInviteResult{}).Decode, nil)
		})
	}
}

func TestInteractionEnterResultErrorRoundTrip(t *testing.T) {
	input := NewInteractionEnterResultError(5, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionEnterResultError{}).Decode, nil)
		})
	}
}
