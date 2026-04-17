package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationOpenRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationOpen{success: true}
			output := OperationOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}
