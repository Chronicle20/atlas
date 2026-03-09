package cash

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationIncreaseCharacterSlotRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationIncreaseCharacterSlot{isPoints: true, currency: 1, serialNumber: 12345}
			output := ShopOperationIncreaseCharacterSlot{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsPoints() != input.IsPoints() {
				t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
			}
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}
