package quest

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestActionScriptEndRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionScriptEnd{npcId: 9003, x: 60, y: 90}
			output := ActionScriptEnd{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
