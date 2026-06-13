package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v83 ida=0x756a04
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v87 ida=0x7a256b
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v95 ida=0x6e7260
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=jms_v185 ida=0x7cacab
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v84 ida=0x778cb8
func TestShopSellRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopSell{slot: 5, itemId: 1000000, quantity: 10}
			output := ShopSell{}
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
		})
	}
}
