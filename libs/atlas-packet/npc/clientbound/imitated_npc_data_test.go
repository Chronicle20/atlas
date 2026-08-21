package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v61 ida=0x5efc2e
// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v72 ida=0x645d28
// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v83 ida=0x6d97c6
// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v84 ida=0x6f0966
// packet-audit:verify packet=npc/clientbound/ImitatedNpcData version=gms_v79 ida=0x668877
func TestImitatedNpcData(t *testing.T) {
	input := NewImitatedNpcData([]ImitatedNpc{
		NewImitatedNpc(9901000, "Hero", packetmodel.NewAvatar(
			0,                                    // gender
			3,                                    // skin
			20000,                                // face
			false,                                // mega
			30030,                                // hair
			map[slot.Position]uint32{5: 1040010}, // equipment
			map[slot.Position]uint32{},           // masked
			map[int8]uint32{},                    // pets
		)),
	})

	expected := []byte{
		0x01,                   // entry count
		0xC8, 0x13, 0x97, 0x00, // templateId 9901000 (little-endian; brief's "1B" is a typo for "13")
		0x04, 0x00, 0x48, 0x65, 0x72, 0x6F, // name "Hero"
		0x00,                   // gender
		0x03,                   // skin
		0x20, 0x4E, 0x00, 0x00, // face 20000
		0x01,                   // !mega
		0x4E, 0x75, 0x00, 0x00, // hair 30030
		0x05, 0x8A, 0xDE, 0x0F, 0x00, // equip slot 5 -> item 1040010
		0xFF,                   // equip terminator
		0xFF,                   // masked terminator
		0x00, 0x00, 0x00, 0x00, // nWeaponStickerID
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // three pet ints
	}

	ctx := test.CreateContext("GMS", 83, 1)
	got := input.Encode(nil, ctx)(nil)
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d bytes, want %d bytes\ngot:  % X\nwant: % X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("byte %d mismatch: got %#02x, want %#02x\ngot:  % X\nwant: % X", i, got[i], expected[i], got, expected)
		}
	}

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestImitatedNpcDataEmpty(t *testing.T) {
	input := NewImitatedNpcData(nil)
	got := input.Encode(nil, test.CreateContext("GMS", 83, 1))(nil)
	expected := []byte{0x00}
	if len(got) != len(expected) || got[0] != expected[0] {
		t.Fatalf("got %v, want %v", got, expected)
	}
}
