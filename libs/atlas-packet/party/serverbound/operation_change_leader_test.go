package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationChangeLeaderRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationChangeLeader{targetCharacterId: 300}
			output := OperationChangeLeader{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}
