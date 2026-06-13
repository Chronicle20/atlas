package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v95 ida=0x481bc0
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v87 ida=0x47d27e
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
