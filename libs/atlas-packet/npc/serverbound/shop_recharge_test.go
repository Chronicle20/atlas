package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v83 ida=0x756c28
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v87 ida=0x7a278f
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v95 ida=0x6e4e90
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=jms_v185 ida=0x7caecf
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v84 ida=0x778edc
func TestShopRechargeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopRecharge{slot: 7}
			output := ShopRecharge{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}
