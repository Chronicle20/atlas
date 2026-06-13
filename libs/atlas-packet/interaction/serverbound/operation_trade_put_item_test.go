package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v83 ida=0x7c359f
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v95 ida=0x7641d0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradePutItem version=gms_v84 ida=0x7e96e5
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
