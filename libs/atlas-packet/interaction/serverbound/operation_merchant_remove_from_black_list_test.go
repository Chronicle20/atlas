package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

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
