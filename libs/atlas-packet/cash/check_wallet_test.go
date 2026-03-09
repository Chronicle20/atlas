package cash

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCheckWalletRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CheckWallet{}
			output := CheckWallet{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
