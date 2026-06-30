package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v95 ida=0x481bc0
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v87 ida=0x47d27e
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v83 ida=0x472958
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=jms_v185 ida=0x48441d
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v84 ida=0x47544e
//
// v79 (CCashShop::TrySendQueryCashRequest, unnamed v79 twin sub_46C34C
// @0x46C34C): COutPacket(220=0xDC CHECK_CASH sb op) + SendPacket — NO Encode
// calls, an opcode-only packet. The body Atlas decodes is empty, matching this
// codec's []byte{}. Export entry resolved from the unnamed twin's decompile.
// packet-audit:verify packet=cash/serverbound/CashCheckWallet version=gms_v79 ida=0x46c34c
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
