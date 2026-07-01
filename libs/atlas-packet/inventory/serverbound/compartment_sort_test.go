package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentSortRequest version=gms_v95 ida=0x9d5c60
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentSortRequest version=gms_v87 ida=0xa9e756
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentSortRequest version=jms_v185 ida=0xaed96f
//
// v79 (ITEM_SORT2 op 68, unnamed twin sub_954CFD @0x954CFD): COutPacket(68) +
// Encode4(get_update_time) + Encode1(a2=compartmentType, guarded a2 in [1,5]) —
// matches Decode4(updateTime)+Decode1(compartmentType). Export entry spliced
// from the unnamed twin's decompile (SendSortItemRequest was absent from the
// v79 export).
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentSortRequest version=gms_v79 ida=0x954cfd
//
// v72 (ITEM_SORT2 op 69, sub_9039D7 @0x9039d7): COutPacket(69) + Encode4(updateTime)
// @0x903a1a + Encode1(a2=compartmentType, guarded a2 in [1,5])@0x903a25 — identical
// to v79. No version gate on the codec.
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentSortRequest version=gms_v72 ida=0x9039d7
func TestCompartmentSortRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, CompartmentSortRequest{updateTime: 100, compartmentType: 2}.Encode, nil)
	want := []byte{0x64, 0x00, 0x00, 0x00, 0x02} // updateTime=100 (LE), compartmentType=2
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

func TestCompartmentSortRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CompartmentSortRequest{updateTime: 100, compartmentType: 2}
			output := CompartmentSortRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CompartmentType() != input.CompartmentType() {
				t.Errorf("compartmentType: got %v, want %v", output.CompartmentType(), input.CompartmentType())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
