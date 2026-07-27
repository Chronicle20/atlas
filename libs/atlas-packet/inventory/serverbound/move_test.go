package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryMove version=gms_v95 ida=0x9d9c10
// packet-audit:verify packet=inventory/serverbound/InventoryMove version=gms_v87 ida=0xa9e7e8
// packet-audit:verify packet=inventory/serverbound/InventoryMove version=jms_v185 ida=0xaeda01
func TestMoveRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Move{updateTime: 12345, inventoryType: 1, source: 5, destination: 10, count: 1}
			output := Move{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.Destination() != input.Destination() {
				t.Errorf("destination: got %v, want %v", output.Destination(), input.Destination())
			}
			if output.Count() != input.Count() {
				t.Errorf("count: got %v, want %v", output.Count(), input.Count())
			}
		})
	}
}

// TestMoveBytesV48 pins the v48 ITEM_MOVE (sb op 55 / 0x37) send. IDA
// GMS_v48_1_DEVM.exe @port 13337: sub_70D8DE@0x70d905 builds COutPacket(55),
// Encode4(updateTime)@0x70d917, Encode1(inventoryType/a2)@0x70d922,
// Encode2(source/a3)@0x70d92d, Encode2(destination/a4)@0x70d938,
// Encode2(count/a5)@0x70d943. No version gate — v48 body == v83..v95.
// packet-audit:verify packet=inventory/serverbound/InventoryMove version=gms_v48 ida=0x70d8de
func TestMoveBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := Move{updateTime: 0x0A0B0C0D, inventoryType: 0x01, source: 0x0203, destination: 0x0405, count: 0x0607}.Encode(nil, ctx)(nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x70d917 (LE)
		0x01,       // inventoryType Encode1@0x70d922
		0x03, 0x02, // source Encode2@0x70d92d (LE)
		0x05, 0x04, // destination Encode2@0x70d938 (LE)
		0x07, 0x06, // count Encode2@0x70d943 (LE)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("v48 bytes = % X, want % X", got, want)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("v48 len = %d, want %d", len(got), len(want))
	}
}
