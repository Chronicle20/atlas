package cash

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationMoveToCashInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationMoveToCashInventory{serialNumber: 9876543210, inventoryType: 3}
			output := ShopOperationMoveToCashInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
		})
	}
}
