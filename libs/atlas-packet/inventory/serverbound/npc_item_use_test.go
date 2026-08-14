package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body. Identical on all nine
// versions that carry the opcode (v61 through jms_v185), so no version gating.
//
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// THERE IS NO updateTime. The sibling ScriptedItem codec in this package leads
// with one; copying its prologue here misaligns every subsequent read. This is
// the single most likely defect in this pair.
//
// Client gate on every version:
//
//	(nItemID / 10000 == 545 || nItemID / 10000 == 239) && CanSendExclRequest(200, 0)
//
// plus two refusal arms that emit a chat message and send nothing (field flag
// bit 18 set; a CUniqueModeless dialog already open).
//
// ABSENT from gms_v12 and gms_v48 — confirmed by instruction scan for
// `cmp ,545` / `cmp ,239`, not by a missing symbol.
func TestNpcItemUseRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NpcItemUse{source: 3, itemId: 2390001}
			output := NpcItemUse{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Guards the no-updateTime invariant explicitly: the encoded frame must be
// exactly 6 bytes. A stray leading updateTime makes it 10.
func TestNpcItemUseWireLayoutHasNoUpdateTime(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
