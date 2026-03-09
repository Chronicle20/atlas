package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestDropMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DropMeso{updateTime: 100, amount: 5000}
			output := DropMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}
