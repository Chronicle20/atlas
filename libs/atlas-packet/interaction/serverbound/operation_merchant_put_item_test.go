package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v95 ida=0x69c880
func TestOperationMerchantPutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantPutItem{inventoryType: 2, slot: 7, quantity: 15, set: 4, price: 2000}
			output := OperationMerchantPutItem{}
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
