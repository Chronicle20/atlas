package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v95 ida=0x9ddfe0
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v87 ida=0xa9ead9
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v83 ida=0xa092fb
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=jms_v185 ida=0xaedea5
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v84 ida=0xa5360f
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v92 ida=0x9b3600
func TestItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUse{operation: CharacterItemUseHandle, updateTime: 12345, source: 5, itemId: 2000000}
			output := ItemUse{operation: CharacterItemUseHandle}
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
		})
	}
}

// TestItemUseBytesV48 pins the v48 USE_ITEM (sb op 56 / 0x38) send. IDA
// GMS_v48_1_DEVM.exe (session 93cc947e): CWvsContext::SendStatChangeItemUseRequest
// (already named in the IDB) @0x70db3c builds COutPacket(56)@0x70dc16,
// Encode4(updateTime)@0x70dc2a, Encode2(source/a2)@0x70dc35, Encode4(itemId/a3)
// @0x70dc40. No version gate — v48 body == v83..v95 (updateTime+slot+itemId).
//
// task-229 CORRECTION: this marker previously pointed at ida=0x719dd9
// (sub_719DD9), claiming that was the unnamed USE_ITEM sender. Re-decompile
// proved 0x719dd9 is already named CWvsContext::SendPortalScrollUseRequest —
// the USE_RETURN_SCROLL sender (opcode 65 / 0x41), a different op gated on a
// different item category (203, not the 200/201/202/205/221 potion family).
// The genuine USE_ITEM sender is CWvsContext::SendStatChangeItemUseRequest at
// opcode 56 / 0x38, re-pinned here.
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v48 ida=0x70db3c
func TestItemUseBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	in := ItemUse{operation: CharacterItemUseHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x70dc2a (LE)
		0x03, 0x02, // source/slot Encode2@0x70dc35 (LE)
		0x17, 0x16, 0x15, 0x14, // itemId Encode4@0x70dc40 (LE)
	}
	if len(got) != len(want) {
		t.Fatalf("v48 len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("v48 bytes = % X, want % X", got, want)
		}
	}
}
