package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v83 ida=0x95fe9e
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v87 ida=0x9e3066
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v95 ida=0x9321f0
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=jms_v185 ida=0xa2cc90
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v84 ida=0x99ec4e
func TestStartConversationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StartConversation{oid: 42, x: 100, y: -50}
			output := StartConversation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Oid() != input.Oid() {
				t.Errorf("oid: got %v, want %v", output.Oid(), input.Oid())
			}
			// Every IDA-verified GMS build (v48/v61/v72/v79/v83+) and JMS carries
			// the user-position x/y shorts after the npc oid; only pre-v48 GMS with
			// no IDB (e.g. the v28 variant below) is oid-only, so its x/y stay zero
			// after the round trip. This test's variants are all v79+ or JMS or v28.
			hasXY := (v.Region == "GMS" && v.MajorVersion >= 79) || v.Region == "JMS"
			if hasXY {
				if output.X() != input.X() {
					t.Errorf("x: got %v, want %v", output.X(), input.X())
				}
				if output.Y() != input.Y() {
					t.Errorf("y: got %v, want %v", output.Y(), input.Y())
				}
			} else {
				if output.X() != 0 || output.Y() != 0 {
					t.Errorf("legacy oid-only: expected x/y zero, got x=%v y=%v", output.X(), output.Y())
				}
			}
		})
	}
}
