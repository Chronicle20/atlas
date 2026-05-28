package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestExpressionRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ExpressionRequest{emote: 7, duration: 3000, byItemOption: true}
			output := ExpressionRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Emote() != input.Emote() {
				t.Errorf("emote: got %v, want %v", output.Emote(), input.Emote())
			}
			// duration and byItemOption are only present in GMS>87 (NOT JMS v185).
			// IDA v83 and v87 CWvsContext::SendEmotionChange encode only Encode4(emotionId); v95 adds both.
			// IDA JMS v185 CWvsContext::SendEmotionChange@0xb0b8be encodes only Encode4(charId) —
			// fundamentally different: JMS sends charId only. No duration or byItemOption for JMS.
			hasDurationAndOption := v.Region == "GMS" && v.MajorVersion > 87
			if hasDurationAndOption {
				if output.Duration() != input.Duration() {
					t.Errorf("duration: got %v, want %v", output.Duration(), input.Duration())
				}
				if output.ByItemOption() != input.ByItemOption() {
					t.Errorf("byItemOption: got %v, want %v", output.ByItemOption(), input.ByItemOption())
				}
			} else {
				if output.Duration() != 0 {
					t.Errorf("duration: expected 0 for v83/v87/JMS, got %v", output.Duration())
				}
				if output.ByItemOption() != false {
					t.Errorf("byItemOption: expected false for v83/v87/JMS, got %v", output.ByItemOption())
				}
			}
		})
	}
}
