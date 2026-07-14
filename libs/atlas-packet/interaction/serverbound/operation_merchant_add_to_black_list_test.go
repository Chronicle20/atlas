package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantAddToBlackList version=gms_v95 ida=0x51ed50
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantAddToBlackList version=gms_v87 ida=0x53c0e6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantAddToBlackList version=gms_v83 ida=0x519611
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantAddToBlackList version=jms_v185 ida=0x54bb75
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantAddToBlackList version=gms_v84 ida=0x5226c2
func TestOperationMerchantAddToBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantAddToBlackList{name: "BannedMerchant"}
			output := OperationMerchantAddToBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
