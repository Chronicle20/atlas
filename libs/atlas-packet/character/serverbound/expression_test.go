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
			// duration and byItemOption are only present in GMS>83 and JMS.
			// IDA v83 CWvsContext::SendEmotionChange@0xa24470 encodes only Encode4(emotionId).
			hasDurationAndOption := (v.Region == "GMS" && v.MajorVersion > 83) || v.Region == "JMS"
			if hasDurationAndOption {
				if output.Duration() != input.Duration() {
					t.Errorf("duration: got %v, want %v", output.Duration(), input.Duration())
				}
				if output.ByItemOption() != input.ByItemOption() {
					t.Errorf("byItemOption: got %v, want %v", output.ByItemOption(), input.ByItemOption())
				}
			} else {
				if output.Duration() != 0 {
					t.Errorf("duration: expected 0 for v83, got %v", output.Duration())
				}
				if output.ByItemOption() != false {
					t.Errorf("byItemOption: expected false for v83, got %v", output.ByItemOption())
				}
			}
		})
	}
}
