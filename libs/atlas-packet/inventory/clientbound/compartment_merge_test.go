package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentMerge version=gms_v95 ida=0x9f1280
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentMerge version=gms_v87 ida=0xab5c54
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentMerge version=gms_v83 ida=0xa1e943
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentMerge version=jms_v185 ida=0xb05482
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentMerge version=gms_v84 ida=0xa69bf9
func TestCompartmentMerge(t *testing.T) {
	input := NewCompartmentMerge(3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
