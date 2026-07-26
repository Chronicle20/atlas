package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
			// Mirror the codec's exact gate (start_conversation.go:39-44,
			// startConversationHasXY): MajorAtLeast(72) || ==61 || ==48 for GMS, true
			// for all other regions. IDA-verified per-version: v48 @0x568a2a,
			// v61 @0x7b1403, v72 @0x63fd91, v79 @0x8b7e10, v83+/JMS likewise carry x/y;
			// only pre-v48 GMS with no IDB (e.g. the v28 variant) is oid-only.
			hasXY := startConversationHasXY(tenant.MustFromContext(ctx))
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
