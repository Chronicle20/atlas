package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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
