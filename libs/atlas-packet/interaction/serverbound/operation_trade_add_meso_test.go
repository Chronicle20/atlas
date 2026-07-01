package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v79 ida=0x736ec4
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v83 ida=0x7c37ca
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v95 ida=0x764450
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v84 ida=0x7e9910
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v87 ida=0x816efc
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=jms_v185 ida=0x84817c
func TestOperationTradeAddMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTradeAddMeso{amount: 50000}
			output := OperationTradeAddMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}

// TestOperationTradeAddMesoV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CTradingRoomDlg::PutMoney (sub_6FF3E9): Encode1(0xF)=mode @0x6ff560 then Encode4(amount) @0x6ff571. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeAddMeso version=gms_v72 ida=0x6ff3e9
func TestOperationTradeAddMesoV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTradeAddMeso{amount: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "40420f00" {
		t.Errorf("v72 bytes: got %s, want 40420f00", got)
	}
}
