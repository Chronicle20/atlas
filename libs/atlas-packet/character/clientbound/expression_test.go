package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterExpressionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterExpression{characterId: 12345, expression: 5, duration: 3000, byItemOption: true}
			output := CharacterExpression{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Expression() != input.Expression() {
				t.Errorf("expression: got %v, want %v", output.Expression(), input.Expression())
			}
			if output.Duration() != input.Duration() {
				t.Errorf("duration: got %v, want %v", output.Duration(), input.Duration())
			}
			if output.ByItemOption() != input.ByItemOption() {
				t.Errorf("byItemOption: got %v, want %v", output.ByItemOption(), input.ByItemOption())
			}
		})
	}
}
