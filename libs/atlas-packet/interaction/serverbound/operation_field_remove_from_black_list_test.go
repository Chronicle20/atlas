package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v79 ida=0x522cff
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v95 ida=0x5397d0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v87 ida=0x55f31e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v83 ida=0x5379a6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=jms_v185 ida=0x574bdf
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldRemoveFromBlackList version=gms_v84 ida=0x543ca4
func TestOperationFieldRemoveFromBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationFieldRemoveFromBlackList{name: "UnblockedUser"}
			output := OperationFieldRemoveFromBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
