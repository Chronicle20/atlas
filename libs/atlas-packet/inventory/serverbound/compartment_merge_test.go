package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v95 ida=0x9d5b70
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v87 ida=0xa9e6c4
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v83 ida=0xa08ee6
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=jms_v185 ida=0xaed8dd
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v84 ida=0xa531fa
//
// v79 (ITEM_SORT op 67, unnamed twin sub_954C6B @0x954C6B): COutPacket(67) +
// Encode4(get_update_time) + Encode1(a2=compartmentType, guarded a2 in [1,5]) —
// matches Decode4(updateTime)+Decode1(compartmentType). Export entry resolved
// from the unnamed twin's decompile.
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v79 ida=0x954c6b
//
// v72 (ITEM_SORT op 68, sub_903945 @0x903945): COutPacket(68) + Encode4(updateTime)
// @0x903988 + Encode1(a2=compartmentType, guarded a2 in [1,5])@0x903993 — identical
// to v79 (updateTime + compartmentType). No version gate on the codec.
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v72 ida=0x903945
func TestCompartmentMergeRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, CompartmentMergeRequest{updateTime: 100, compartmentType: 1}.Encode, nil)
	want := []byte{0x64, 0x00, 0x00, 0x00, 0x01} // updateTime=100 (LE), compartmentType=1
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// v61 (ITEM_SORT op 64, sub_8314D0 @0x8314d0): COutPacket(64) +
// Encode4(get_update_time)@0x831515 + Encode1(a2=compartmentType, guarded a2 in
// [1,5])@0x831520 — body byte-identical to v72 [E4,E1]. The op-64 send-site was
// located by structure-match to the v72 gather twin sub_903945 (SetExclRequestSent
// + this[2083..2084] + guard [1,5]); the prior registry op68/sub_831BB7 was a
// different item-id send (mislabel). No version gate on the codec.
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v61 ida=0x8314d0
func TestCompartmentMergeRequestBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := pt.Encode(t, ctx, CompartmentMergeRequest{updateTime: 100, compartmentType: 1}.Encode, nil)
	want := []byte{0x64, 0x00, 0x00, 0x00, 0x01} // updateTime=100 (LE), compartmentType=1
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

func TestCompartmentMergeRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CompartmentMergeRequest{updateTime: 100, compartmentType: 1}
			output := CompartmentMergeRequest{}
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
