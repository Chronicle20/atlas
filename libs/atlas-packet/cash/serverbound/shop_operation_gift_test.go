package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestShopOperationGiftRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationGift{birthday: 19900101, serialNumber: 12345, name: "Player1", message: "Happy birthday!"}
			output := ShopOperationGift{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
