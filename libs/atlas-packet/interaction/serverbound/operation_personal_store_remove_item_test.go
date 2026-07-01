package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v79 ida=0x68a756
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v95 ida=0x6987a0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v87 ida=0x741271
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v83 ida=0x6fdcdf
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=jms_v185 ida=0x762e26
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v84 ida=0x719ffd
func TestOperationPersonalStoreRemoveItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreRemoveItem{index: 7}
			output := OperationPersonalStoreRemoveItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}
