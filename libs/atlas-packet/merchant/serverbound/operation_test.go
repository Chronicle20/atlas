package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationRoundTrip exercises the entrusted-shop (hired-merchant) serverbound
// dispatcher: a single mode byte followed by the 8-byte cash-item serial number.
// Shape confirmed in JMS v185 CWvsContext::SendEntrustedShopCheckRequest (opcode 0x37):
//
//	Encode1(0)               // mode, always 0 (EntrustedShopCheck)
//	EncodeBuffer(&liSN, 8)   // cash-item serial number (uint64)
func TestOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Operation{mode: ModeEntrustedShopCheck, cashItemSerialNumber: 0x1122334455667788}
			output := Operation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.CashItemSerialNumber() != input.CashItemSerialNumber() {
				t.Errorf("cashItemSerialNumber: got %v, want %v", output.CashItemSerialNumber(), input.CashItemSerialNumber())
			}
		})
	}
}
