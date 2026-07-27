package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=gms_v95 ida=0x51ee20
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=gms_v87 ida=0x53c16d
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=gms_v83 ida=0x519695
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=jms_v185 ida=0x54bbf9
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=gms_v84 ida=0x522746
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList version=gms_v79 ida=0x505911
func TestOperationMerchantRemoveFromBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantRemoveFromBlackList{name: "UnbannedMerchant"}
			output := OperationMerchantRemoveFromBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
