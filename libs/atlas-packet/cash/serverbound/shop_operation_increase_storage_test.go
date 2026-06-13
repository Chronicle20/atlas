package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=gms_v95 ida=0x48dc70
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=gms_v87 ida=0x4763e0
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=gms_v83 ida=0x46c55b
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=jms_v185 ida=0x47c766
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
