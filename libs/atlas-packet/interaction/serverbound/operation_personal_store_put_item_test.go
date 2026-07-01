package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v79 ida=0x68a3e3
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v95 ida=0x69c880
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v87 ida=0x740ee6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v83 ida=0x6fd96c
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=jms_v185 ida=0x762a9e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v84 ida=0x719c8a
func TestOperationPersonalStorePutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStorePutItem{inventoryType: 1, slot: 5, quantity: 10, set: 3, price: 1000}
			output := OperationPersonalStorePutItem{}
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
			if output.Set() != input.Set() {
				t.Errorf("set: got %v, want %v", output.Set(), input.Set())
			}
			if output.Price() != input.Price() {
				t.Errorf("price: got %v, want %v", output.Price(), input.Price())
			}
		})
	}
}

// TestOperationPersonalStorePutItemV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::PutItem (sub_665F5F): Encode1(0x14 personal)=mode @0x6661e8 then Encode1(invType)@0x6661fd, Encode2(slot)@0x666208, Encode2(qty)@0x666216, Encode2(set)@0x666234, Encode4(price)@0x66623f. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStorePutItem version=gms_v72 ida=0x665f5f
func TestOperationPersonalStorePutItemV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStorePutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v72 bytes: got %s, want 0205006400070040420f00", got)
	}
}
