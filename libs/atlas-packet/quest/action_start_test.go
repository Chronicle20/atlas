package quest

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestActionStartAutoStartRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionStart{npcId: 9000, x: 100, y: 200, autoStart: true}
			output := *NewActionStart(true)
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

func TestActionStartNoAutoStartRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionStart{npcId: 9000, autoStart: false}
			output := *NewActionStart(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
			if output.X() != -1 {
				t.Errorf("x: got %v, want -1", output.X())
			}
			if output.Y() != -1 {
				t.Errorf("y: got %v, want -1", output.Y())
			}
		})
	}
}
