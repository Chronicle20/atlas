package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationTradeAddMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTradeAddMeso{amount: 50000}
			output := OperationTradeAddMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}
