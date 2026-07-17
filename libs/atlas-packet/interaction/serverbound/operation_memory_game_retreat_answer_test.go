package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v79 ida=0x672728
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v95 ida=0x6804b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v87 ida=0x721c80
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v83 ida=0x6e416b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=jms_v185 ida=0x72b69c
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v84 ida=0x6fb416
func TestOperationMemoryGameRetreatAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameRetreatAnswer{response: true}
			output := OperationMemoryGameRetreatAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}

// TestOperationMemoryGameRetreatAnswerV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). Marker re-pinned (task-133 item 5) to the
// true RETREAT_ANSWER send: v72 Omok ASK_RETREAT handler sub_64E953 @0x64e953 —
// COutPacket(121)+Encode1(0x31=49)=mode then Encode1(YesNo==6). bool response. Body == v79.
// (The old ida=0x5febf2 was the MemoryGame ASK_TIE handler; RETREAT_ANSWER serverbound
// mode is 49 per character_interaction_handle.yaml.)
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v72 ida=0x64e953
func TestOperationMemoryGameRetreatAnswerV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameRetreatAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01" {
		t.Errorf("v72 bytes: got %s, want 01", got)
	}
}
