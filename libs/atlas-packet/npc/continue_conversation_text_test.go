package npc

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestContinueConversationTextRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationText{text: "Hello NPC"}
			output := ContinueConversationText{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Text() != input.Text() {
				t.Errorf("text: got %v, want %v", output.Text(), input.Text())
			}
		})
	}
}
