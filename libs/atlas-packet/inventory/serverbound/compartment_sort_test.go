package serverbound

import (
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
