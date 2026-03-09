package cash

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationIncreaseStorageItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationIncreaseStorage{isPoints: true, currency: 1, item: true, serialNumber: 12345}
			output := ShopOperationIncreaseStorage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsPoints() != input.IsPoints() {
				t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
			}
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.Item() != input.Item() {
				t.Errorf("item: got %v, want %v", output.Item(), input.Item())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

func TestShopOperationIncreaseStorageNoItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationIncreaseStorage{isPoints: false, currency: 2, item: false}
			output := ShopOperationIncreaseStorage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsPoints() != input.IsPoints() {
				t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
			}
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.Item() != input.Item() {
				t.Errorf("item: got %v, want %v", output.Item(), input.Item())
			}
		})
	}
}
