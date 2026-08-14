package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body. The client body is
// byte-identical on every version that carries the opcode — a full sweep of all
// ten IDBs found no divergence (task-230 design §1.1), so no version gating is
// required or permitted.
//
//	Encode4(get_update_time())   // uint32 update time
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// Gated client-side on nItemID / 10000 == 243 under CanSendExclRequest(500, 0).
// v83+ additionally guards on CWvsContext::IsAbleToConsume, which v72/v79 lack;
// that is a client-side convenience check the server must not rely on.
// v95 alone also whitelists nItemID == 3994225 (an Install/Setup item) — out of
// scope per design D-3 and rejected server-side.
//
// The op is ABSENT from gms_v12, gms_v48, and gms_v61 (design §1.1 absence
// evidence: dense Send*ItemUseRequest export sets with no SendScriptRunItemRequest).
//
// packet-audit:verify markers are added per cell in the verification task; this
// round-trip alone is NOT a verification.
func TestScriptedItemRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ScriptedItem{updateTime: 0x1A2B3C4D, source: 7, itemId: 2430008}
			output := ScriptedItem{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// The field ORDER is the defect this guards. The sibling NpcItemUse codec has
// no leading updateTime; reading these two files side by side and copying the
// wrong prologue misaligns every subsequent field. Assert the exact bytes.
func TestScriptedItemWireLayout(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32
		0x06, 0x05, //             source, little-endian int16
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
