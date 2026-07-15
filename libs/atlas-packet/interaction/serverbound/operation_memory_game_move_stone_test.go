package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v79 ida=0x676fd6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v95 ida=0x6801e0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v87 ida=0x726570
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v83 ida=0x6e8a19
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=jms_v185 ida=0x72fedf
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v84 ida=0x6ffcc4
func TestOperationMemoryGameMoveStoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameMoveStone{point: 123456789, color: 5}
			output := OperationMemoryGameMoveStone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Point() != input.Point() {
				t.Errorf("point: got %v, want %v", output.Point(), input.Point())
			}
			if output.Color() != input.Color() {
				t.Errorf("color: got %v, want %v", output.Color(), input.Color())
			}
		})
	}
}

// TestOperationMemoryGameMoveStoneV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 COmokDlg::PutStoneChecker (sub_65320C): Encode1(0x3A)=mode @0x65322f then EncodeBuffer(&point,8)@0x65323d, Encode1(color)@0x65324e. int64 point + byte color. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v72 ida=0x65320c
func TestOperationMemoryGameMoveStoneV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameMoveStone{point: 1, color: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "010000000000000002" {
		t.Errorf("v72 bytes: got %s, want 010000000000000002", got)
	}
}
