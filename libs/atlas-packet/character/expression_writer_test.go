package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterExpressionWRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterExpressionW{characterId: 12345, expression: 5}
			output := CharacterExpressionW{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Expression() != input.Expression() {
				t.Errorf("expression: got %v, want %v", output.Expression(), input.Expression())
			}
		})
	}
}
