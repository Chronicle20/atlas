package npc

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestContinueConversationSelectionWideRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationSelection{selection: 42, wide: true}
			output := ContinueConversationSelection{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}

func TestContinueConversationSelectionNarrowRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationSelection{selection: 3, wide: false}
			output := ContinueConversationSelection{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}
