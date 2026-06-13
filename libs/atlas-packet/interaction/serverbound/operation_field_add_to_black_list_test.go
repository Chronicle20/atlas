package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v95 ida=0x539710
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v87 ida=0x55f2a3
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=gms_v83 ida=0x53792e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationFieldAddToBlackList version=jms_v185 ida=0x574b67
func TestOperationFieldAddToBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationFieldAddToBlackList{name: "BlockedUser"}
			output := OperationFieldAddToBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
