package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v83 ida=0x7561c1
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v87 ida=0x7a1d49
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v95 ida=0x6e9bb0
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v84 ida=0x778475
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
			// The trailing discountPrice int is GMS-only; JMS185 omits it
			// (CShopDlg::SendBuyRequest@0x7ca2c9 ends after the quantity short).
			if v.Region == "GMS" {
				if output.DiscountPrice() != input.DiscountPrice() {
					t.Errorf("discountPrice: got %v, want %v", output.DiscountPrice(), input.DiscountPrice())
				}
			}
		})
	}
}

// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=jms_v185 ida=0x7ca2c9
func TestShopBuyDiscountPriceGate(t *testing.T) {
	input := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}

	gmsCtx := pt.CreateContext("GMS", 95, 1)
	gms := input.Encode(nil, gmsCtx)(nil)
	// slot(2) + itemId(4) + quantity(2) + discountPrice(4) = 12
	if len(gms) != 12 {
		t.Errorf("GMS: expected 12 bytes (with discountPrice), got %d", len(gms))
	}

	jmsCtx := pt.CreateContext("JMS", 185, 1)
	jms := input.Encode(nil, jmsCtx)(nil)
	// slot(2) + itemId(4) + quantity(2) = 8 (no discountPrice)
	if len(jms) != 8 {
		t.Errorf("JMS: expected 8 bytes (no discountPrice), got %d", len(jms))
	}
}
