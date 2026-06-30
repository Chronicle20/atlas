package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=gms_v83 ida=0x706393
// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=gms_v87 ida=0x74a35f
// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=gms_v95 ida=0x6a0dd0
// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=jms_v185 ida=0x76c05e
// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=gms_v84 ida=0x722df2
func TestExcludeItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ExcludeItem{petId: 12345, itemIds: []int32{1000, 2000, 3000}}
			output := ExcludeItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if len(output.ItemIds()) != len(input.ItemIds()) {
				t.Fatalf("itemIds length: got %v, want %v", len(output.ItemIds()), len(input.ItemIds()))
			}
			for i, id := range output.ItemIds() {
				if id != input.ItemIds()[i] {
					t.Errorf("itemIds[%d]: got %v, want %v", i, id, input.ItemIds()[i])
				}
			}
		})
	}
}

// v79 PET_EXCLUDE_ITEMS (sb op 168=0xA8) send order, verified GMS_v79_1_DEVM.exe
// (port 13340): sub_692ABB — COutPacket(168)@0x692ad4, EncodeBuffer(petId,8)@0x692ae9,
// Encode1(count)@0x692afe, count×Encode4(itemId)@0x692b16. Wire =
// petId(8)+count(1)+itemIds(4 each); byte-identical to v83.
// packet-audit:verify packet=pet/serverbound/PetExcludeItem version=gms_v79 ida=0x692abb
func TestExcludeItemBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ExcludeItem{petId: 0x0102030405060708, itemIds: []int32{0x11223344, 0x55667788}}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x692ae9 (LE)
		0x02,                   // count Encode1@0x692afe
		0x44, 0x33, 0x22, 0x11, // itemId[0] Encode4@0x692b16 (LE)
		0x88, 0x77, 0x66, 0x55, // itemId[1] Encode4@0x692b16 (LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
