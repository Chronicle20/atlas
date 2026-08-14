package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// SummonBagItemUse is an audit-only wrapper: it exists so USE_SUMMON_BAG gets
// its own packet id, audit report and evidence key instead of borrowing
// InventoryItemUse's (which pins the *potion* sender
// CWvsContext::SendStatChangeItemUseRequest). The production handler in
// atlas-channel keeps decoding the shared ItemUse directly. See task-229 and
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
//
// Byte fixtures with `packet-audit:verify` markers are appended per version by
// the per-version verification tasks; this round-trip covers every tenant
// variant.
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=gms_v83 ida=0xa0977b
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=gms_v61 ida=0x831c83
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=gms_v84 ida=0xa53b5d
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=gms_v87 ida=0xa9f027
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=gms_v92 ida=0x9b3b80
func TestSummonBagItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SummonBagItemUse{ItemUse: ItemUse{
				operation:  CharacterItemUseSummonBagHandle,
				updateTime: 12345,
				source:     5,
				itemId:     2100000,
			}}
			output := SummonBagItemUse{ItemUse: ItemUse{operation: CharacterItemUseSummonBagHandle}}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Operation() != CharacterItemUseSummonBagHandle {
				t.Errorf("operation: got %v, want %v", output.Operation(), CharacterItemUseSummonBagHandle)
			}
		})
	}
}

// The wrapper must not drift from the shared body: byte-for-byte identical
// encodings are the whole point of embedding rather than redeclaring.
func TestSummonBagItemUseMatchesSharedBody(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	shared := ItemUse{operation: CharacterItemUseSummonBagHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	wrapped := SummonBagItemUse{ItemUse: shared}
	a := shared.Encode(nil, ctx)(nil)
	b := wrapped.Encode(nil, ctx)(nil)
	if len(a) != len(b) {
		t.Fatalf("len: shared %d, wrapper %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("bytes: shared % X, wrapper % X", a, b)
		}
	}
}
