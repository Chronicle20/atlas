package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=gms_v83 ida=0x6d9a83
// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=gms_v87 ida=0x7170c8
// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=gms_v95 ida=0x679730
// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=jms_v185 ida=0x720782
// packet-audit:verify packet=npc/clientbound/NpcSpawnRequestController version=gms_v84 ida=0x6f0c26
func TestNpcSpawnRequestController(t *testing.T) {
	input := NewNpcSpawnRequestController(100, 9010000, 150, -300, 0, 500, -50, 250, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
