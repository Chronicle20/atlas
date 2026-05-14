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
			// duration is present in GMS>87 and JMS v185.
			// byItemOption is only present in GMS>87 (NOT JMS v185).
			// IDA v83/v87: only Decode4(expressionId) inline. v95: adds duration+byItemOption.
			// IDA JMS v185 CUser::OnEmotion@0x9f636b: reads Decode4(nEmotion)+Decode4(tDuration) only.
			hasDuration := (v.Region == "GMS" && v.MajorVersion > 87) || v.Region == "JMS"
			hasByItemOption := v.Region == "GMS" && v.MajorVersion > 87
			if hasDuration {
				if output.Duration() != input.Duration() {
					t.Errorf("duration: got %v, want %v", output.Duration(), input.Duration())
				}
			} else {
				if output.Duration() != 0 {
					t.Errorf("duration: expected 0 for v83/v87, got %v", output.Duration())
				}
			}
			if hasByItemOption {
				if output.ByItemOption() != input.ByItemOption() {
					t.Errorf("byItemOption: got %v, want %v", output.ByItemOption(), input.ByItemOption())
				}
			} else {
				if output.ByItemOption() != false {
					t.Errorf("byItemOption: expected false for v83/v87/JMS, got %v", output.ByItemOption())
				}
			}
		})
	}
}
