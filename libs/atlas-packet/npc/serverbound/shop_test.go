package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcShop version=gms_v87 ida=0x7a249e
// packet-audit:verify packet=npc/serverbound/NpcShop version=gms_v95 ida=0x6e4b80
// packet-audit:verify packet=npc/serverbound/NpcShop version=jms_v185 ida=0x7ca2c9
func TestShopRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Shop{op: 2}
			output := Shop{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Op() != input.Op() {
				t.Errorf("op: got %v, want %v", output.Op(), input.Op())
			}
		})
	}
}
