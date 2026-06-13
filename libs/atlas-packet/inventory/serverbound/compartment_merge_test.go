package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v95 ida=0x9d5b70
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v87 ida=0xa9e6c4
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v83 ida=0xa08ee6
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=jms_v185 ida=0xaed8dd
// packet-audit:verify packet=inventory/serverbound/InventoryCompartmentMergeRequest version=gms_v84 ida=0xa531fa
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
