package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v79 ida=0x736c99
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v83 ida=0x7c359f
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v95 ida=0x7641d0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v84 ida=0x7e96e5
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v87 ida=0x816cce
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=jms_v185 ida=0x847f51
func TestOperationTradePutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTradePutItem{inventoryType: 2, slot: 5, quantity: 100, targetSlot: 3}
			output := OperationTradePutItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
			if output.TargetSlot() != input.TargetSlot() {
				t.Errorf("targetSlot: got %v, want %v", output.TargetSlot(), input.TargetSlot())
			}
		})
	}
}

// TestOperationTradePutItemV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CTradingRoomDlg::PutItem (sub_6FF1BE): Encode1(0xE)=mode @0x6ff358 then Encode1(invType)@0x6ff363, Encode2(slot)@0x6ff36e, Encode2(qty)@0x6ff379, Encode1(targetSlot)@0x6ff384. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v72 ida=0x6ff1be
func TestOperationTradePutItemV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTradePutItem{inventoryType: 2, slot: 5, quantity: 100, targetSlot: 3}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "020500640003" {
		t.Errorf("v72 bytes: got %s, want 020500640003", got)
	}
}
