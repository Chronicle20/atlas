package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v79 ida=0x51b10b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v95 ida=0x52e9e0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v84 ida=0x53bc2a
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v83 ida=0x52fad4
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v87 ida=0x556cfe
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=jms_v185 ida=0x56c859
func TestOperationInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationInvite{targetCharacterId: 12345}
			output := OperationInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}

// TestOperationInviteV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CField::SendInviteTradingRoomMsg (sub_514092): Init(121) Encode1(2)=mode @0x5141b2 then Encode4(charId) @0x5141c0. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v72 ida=0x514092
func TestOperationInviteV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationInvite{targetCharacterId: 0x12345678}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "78563412" {
		t.Errorf("v72 bytes: got %s, want 78563412", got)
	}
}
