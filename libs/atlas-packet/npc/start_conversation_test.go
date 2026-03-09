package npc

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestStartConversationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StartConversation{oid: 42, x: 100, y: -50}
			output := StartConversation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Oid() != input.Oid() {
				t.Errorf("oid: got %v, want %v", output.Oid(), input.Oid())
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
