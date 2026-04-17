package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestActionCompleteAutoStartRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionComplete{npcId: 9002, x: 30, y: 40, selection: 2, autoStart: true}
			output := *NewActionComplete(true)
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
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}

func TestActionCompleteNoAutoStartRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionComplete{npcId: 9002, selection: 5, autoStart: false}
			output := *NewActionComplete(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
			if output.X() != -1 {
				t.Errorf("x: got %v, want -1", output.X())
			}
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}
