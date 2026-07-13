package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ExpressionRequest v48 byte-fixture — FACE_EXPRESSION serverbound, op 42.
//
// Client send — CWvsContext::SendEmotionChange @0x71d251 (send-site 0x71d2f6):
// after CAvatar::SetEmotion + the 2000ms cooldown + emote<=0x17 guard, builds
// COutPacket(42)@0x71d2f6 then Encode4(emotion)@0x71d31a. v48 (GMS<87) sends NO
// Encode4(duration) and NO Encode1(byItemOption) — both are GMS>87 additions;
// expression.go gates them on GMS>87. Body = emote(4) == v61. v48 op 42 (v61
// FACE_EXPRESSION=48, Δ-6).
//
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v48 ida=0x71d251
func TestExpressionRequestV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ExpressionRequest{emote: 5}.Encode(nil, ctx)(nil)
	want := []byte{0x05, 0x00, 0x00, 0x00} // emote 5 (Encode4) /*0x71d31a*/
	if !bytes.Equal(got, want) {
		t.Errorf("v48 ExpressionRequest wire: got %x want %x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/ExpressionRequest version=jms_v185 ida=0xb0b8be
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v83 ida=0xa24470
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v87 ida=0xabbfbb
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v95 ida=0x9f9320
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v84 ida=0xa6fb0d
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v79 ida=0x96e5c6
// v79 CWvsContext::SendEmotionChange@0x96e5c6: COutPacket(0x31)+Encode4(emotionId) only — no
// duration/byItemOption (added v95). GMS major 79 <= 87 takes the emote-only wire (same as v83/v87).
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
