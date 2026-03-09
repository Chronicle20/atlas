package npc

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopBuyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}
			output := ShopBuy{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
			if output.DiscountPrice() != input.DiscountPrice() {
				t.Errorf("discountPrice: got %v, want %v", output.DiscountPrice(), input.DiscountPrice())
			}
		})
	}
}
