package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v79 ida=0x61d6a6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v95 ida=0x627e60
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v87 ida=0x68826d
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v83 ida=0x64e363
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=jms_v185 ida=0x6c815e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v84 ida=0x664034
func TestOperationMemoryGameTieAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameTieAnswer{response: true}
			output := OperationMemoryGameTieAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}

// TestOperationMemoryGameTieAnswerV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CMemoryGameDlg::OnTieRequest (sub_64E893): Encode1(0x31)=mode @0x64e8b8 then Encode1(YesNo==6) @0x64e8f1. bool response. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v72 ida=0x64e893
func TestOperationMemoryGameTieAnswerV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameTieAnswer{response: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01" {
		t.Errorf("v72 bytes: got %s, want 01", got)
	}
}
