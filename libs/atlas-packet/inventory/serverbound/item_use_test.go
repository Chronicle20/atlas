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

// TestItemUseBytesV48 pins the v48 USE_ITEM (sb op 65 / 0x41) send. IDA
// GMS_v48_1_DEVM.exe @port 13337: sub_719DD9@0x719f8e builds COutPacket(65),
// Encode4(updateTime)@0x719fa0, Encode2(source/a2)@0x719fab, Encode4(itemId/a3)
// @0x719fb6. No version gate — v48 body == v83..v95 (updateTime+slot+itemId).
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v48 ida=0x719dd9
func TestItemUseBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	in := ItemUse{operation: CharacterItemUseHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x719fa0 (LE)
		0x03, 0x02, // source/slot Encode2@0x719fab (LE)
		0x17, 0x16, 0x15, 0x14, // itemId Encode4@0x719fb6 (LE)
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
