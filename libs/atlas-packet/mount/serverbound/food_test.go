package serverbound

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestFoodDecode pins the v83 wire format that the taming-mob food handler
// (opcode 0x4D, SendTamingMobFoodItemUseRequest) consumes:
// ts(4), slot(2), itemId(4) -- all little-endian.
func TestFoodDecode(t *testing.T) {
	// ts = 100 (0x00000064), slot = 3 (0x0003), itemId = 2000000 (0x001E8480)
	raw := []byte{
		0x64, 0x00, 0x00, 0x00, // ts
		0x03, 0x00, // slot
		0x80, 0x84, 0x1E, 0x00, // itemId
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := Food{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.UpdateTime() != 100 {
		t.Errorf("ts: got %d, want 100", p.UpdateTime())
	}
	if p.Slot() != 3 {
		t.Errorf("slot: got %d, want 3", p.Slot())
	}
	if p.ItemId() != 2000000 {
		t.Errorf("itemId: got %d, want 2000000", p.ItemId())
	}
	if p.Operation() != MountFoodHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), MountFoodHandle)
	}
}

// TestFoodByteFixture pins the exact serverbound wire bytes per version,
// hand-computed from each version's decompiled send order (full body, never
// opcode-only). Body is version-invariant (update_time u32·slot i16·itemId u32);
// only the opcode differs, and it is config-resolved (template), not in the codec.
// IDA evidence:
//
//	gms_v48 SendTamingMobFoodItemUseRequest@0x70e00b: op 0x3D; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v61 SendTamingMobFoodItemUseRequest@0x831f44: op 0x48; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	(corrected from assumed 0x4C: decompile+disasm confirmed push 48h into COutPacket ctor;
//	registry/template op76→sub_832680 was a mislabel of a distinct 2-field packet, see
//	docs/packets/registry/gms_v61.yaml USE_MOUNT_FOOD note)
//
//	gms_v72 SendTamingMobFoodItemUseRequest@0x904419: op 0x4C; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v79 SendTamingMobFoodItemUseRequest@0x955781: op 0x4B; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v83 SendTamingMobFoodItemUseRequest@0xa09a64: op 0x4D; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v84 SendTamingMobFoodItemUseRequest@0xa53e46: op 0x4D; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v87 SendTamingMobFoodItemUseRequest@0xa9f310: op 0x50; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//	gms_v95 SendTamingMobFoodItemUseRequest@0x9d63a0: op 0x53; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
//
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v48 ida=0x70e00b
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v61 ida=0x831f44
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v72 ida=0x904419
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v79 ida=0x955781
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v83 ida=0xa09a64
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v84 ida=0xa53e46
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v87 ida=0xa9f310
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v95 ida=0x9d63a0
func TestFoodByteFixture(t *testing.T) {
	cases := []struct {
		variant pt.TenantVariant
		want    []byte
	}{
		{pt.Variants[7], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v48
		{pt.Variants[8], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v61
		{pt.Variants[9], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v72
		{pt.Variants[10], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // gms_v79
		{pt.Variants[1], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v83
		{pt.Variants[5], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v84
		{pt.Variants[2], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v87
		{pt.Variants[3], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}},  // gms_v95
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := Food{updateTime: 100, slot: 3, itemId: 2000000}
			got := input.Encode(logrus.New(), ctx)(nil)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("bytes: got % X, want % X", got, tc.want)
			}
			output := Food{}
			req := request.Request(tc.want)
			reader := request.NewRequestReader(&req, 0)
			output.Decode(logrus.New(), ctx)(&reader, nil)
			if output.UpdateTime() != 100 || output.Slot() != 3 || output.ItemId() != 2000000 {
				t.Errorf("decode round-trip mismatch: %s", output.String())
			}
		})
	}
}
