package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestActionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Action{action: 1, questId: 1234}
			output := Action{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ActionType() != input.ActionType() {
				t.Errorf("action: got %v, want %v", output.ActionType(), input.ActionType())
			}
			if output.QuestId() != input.QuestId() {
				t.Errorf("questId: got %v, want %v", output.QuestId(), input.QuestId())
			}
		})
	}
}
