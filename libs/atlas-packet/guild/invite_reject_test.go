package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestInviteRejectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InviteReject{unk: 1, from: "SomeGuildLeader"}
			output := InviteReject{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
		})
	}
}
