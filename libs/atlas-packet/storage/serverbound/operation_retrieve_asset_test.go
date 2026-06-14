package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=storage/serverbound/StorageOperationRetrieveAsset version=gms_v95 ida=0x769e00
// packet-audit:verify packet=storage/serverbound/StorageOperationRetrieveAsset version=jms_v185 ida=0x84dea0
// packet-audit:verify packet=storage/serverbound/StorageOperationRetrieveAsset version=gms_v87 ida=0x81bc1f
func TestOperationRetrieveAssetRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationRetrieveAsset{inventoryType: 1, slot: 5}
			output := OperationRetrieveAsset{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}
