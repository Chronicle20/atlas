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
// dispatcher-framed, not part of this sub-struct). The pinned sub_5FEBF2 is actually the
// v72 MemoryGame case-44 ASK_TIE handler (Encode1(0x2D=45); MemoryGame has no retreat), so
// this fixture verifies the shared bool-answer body only — tie and retreat answers are
// byte-identical (mode + bool). The true RETREAT_ANSWER send is the Omok ASK_RETREAT handler
// sub_64E953 (Encode1(0x31=49)); the RETREAT_ANSWER serverbound mode is 49 per
// character_interaction_handle.yaml. bool response. Body == v79. (task-133: corrected the
// mislabel; marker/body kept per shared-body equivalence.)
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v72 ida=0x5febf2
func TestOperationMemoryGameRetreatAnswerV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameRetreatAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01" {
		t.Errorf("v72 bytes: got %s, want 01", got)
	}
}
