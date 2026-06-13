package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentSort version=gms_v95 ida=0x9f12b0
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentSort version=gms_v87 ida=0xab5c7e
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentSort version=gms_v83 ida=0xa1e96d
// packet-audit:verify packet=inventory/clientbound/InventoryCompartmentSort version=jms_v185 ida=0xb054ac
func TestCompartmentSort(t *testing.T) {
	input := NewCompartmentSort(2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
