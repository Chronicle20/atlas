package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v79 ida=0x61e16e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v95 ida=0x6279b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v87 ida=0x688d3b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v83 ida=0x64ee2b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=jms_v185 ida=0x6c8b94
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v84 ida=0x664afc
func TestOperationMemoryGameFlipCardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameFlipCard{first: true, index: 7}
			output := OperationMemoryGameFlipCard{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.First() != input.First() {
				t.Errorf("first: got %v, want %v", output.First(), input.First())
			}
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}

// TestOperationMemoryGameFlipCardV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CMemoryGameDlg::SendTurnUpCard (sub_5FF6BA): Encode1(0x3E)=mode @0x5ff6da then Encode1(first)@0x5ff6e5, Encode1(index)@0x5ff6f0. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v72 ida=0x5ff6ba
func TestOperationMemoryGameFlipCardV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMemoryGameFlipCard{first: true, index: 2}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0102" {
		t.Errorf("v72 bytes: got %s, want 0102", got)
	}
}
