package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationDeclineInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDeclineInvite{fromName: "Sender", myName: "Receiver", alwaysZero: 0}
			output := OperationDeclineInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.MyName() != input.MyName() {
				t.Errorf("myName: got %v, want %v", output.MyName(), input.MyName())
			}
			if output.AlwaysZero() != input.AlwaysZero() {
				t.Errorf("alwaysZero: got %v, want %v", output.AlwaysZero(), input.AlwaysZero())
			}
		})
	}
}
