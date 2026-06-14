package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/clientbound/InteractionInteractionChat version=gms_v95 ida=0x639ad0
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v95 ida=0x637510
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v95 ida=0x639500
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v95 ida=0x638f80
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v95 ida=0x637d70
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v95 ida=0x637a40
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v95 ida=0x639500
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v95 ida=0x51cc30
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

func TestInteractionChatRoundTrip(t *testing.T) {
	input := NewInteractionChat(6, 7, 1, "TestPlayer : Hello world")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionChat{}).Decode, nil)
		})
	}
}

func TestInteractionLeaveRoundTrip(t *testing.T) {
	input := NewInteractionLeave(10, 2, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionLeave{}).Decode, nil)
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
