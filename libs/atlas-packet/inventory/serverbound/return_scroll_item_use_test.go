package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ReturnScrollItemUse is an audit-only wrapper: it exists so USE_RETURN_SCROLL
// gets its own packet id, audit report and evidence key instead of borrowing
// InventoryItemUse's (which pins the *potion* sender). The production handler in
// atlas-channel keeps decoding the shared ItemUse directly. See task-229 and
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
// packet-audit:verify packet=inventory/serverbound/InventoryReturnScrollItemUse version=gms_v83 ida=0xa1e1ce
// packet-audit:verify packet=inventory/serverbound/InventoryReturnScrollItemUse version=gms_v61 ida=0x841aa5
// packet-audit:verify packet=inventory/serverbound/InventoryReturnScrollItemUse version=gms_v84 ida=0xa6946d
// packet-audit:verify packet=inventory/serverbound/InventoryReturnScrollItemUse version=gms_v87 ida=0xab5507
func TestReturnScrollItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ReturnScrollItemUse{ItemUse: ItemUse{
				operation:  CharacterItemUseTownScrollHandle,
				updateTime: 12345,
				source:     5,
				itemId:     2030000,
			}}
			output := ReturnScrollItemUse{ItemUse: ItemUse{operation: CharacterItemUseTownScrollHandle}}
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
			if output.Operation() != CharacterItemUseTownScrollHandle {
				t.Errorf("operation: got %v, want %v", output.Operation(), CharacterItemUseTownScrollHandle)
			}
		})
	}
}

func TestReturnScrollItemUseMatchesSharedBody(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	shared := ItemUse{operation: CharacterItemUseTownScrollHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	wrapped := ReturnScrollItemUse{ItemUse: shared}
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
