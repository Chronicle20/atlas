package interaction

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationMerchantNameChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantNameChange{unk1: 99999}
			output := OperationMerchantNameChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk1() != input.Unk1() {
				t.Errorf("unk1: got %v, want %v", output.Unk1(), input.Unk1())
			}
		})
	}
}
