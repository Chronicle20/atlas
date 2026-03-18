package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationRebateLockerItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationRebateLockerItem{birthday: 19900101, unk: 123456789}
			output := ShopOperationRebateLockerItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
		})
	}
}
