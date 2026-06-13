package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v83 ida=0x6d9993
// packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v87 ida=0x716fd5
// packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v95 ida=0x679680
// packet-audit:verify packet=npc/clientbound/NpcSpawn version=jms_v185 ida=0x72068f
// packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v84 ida=0x6f0b33
func TestNpcSpawn(t *testing.T) {
	input := NewNpcSpawn(100, 9010000, 150, -300, 0, 500, -50, 250)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
