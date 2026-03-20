package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationInvite{targetCharacterId: 12345}
			output := OperationInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}
