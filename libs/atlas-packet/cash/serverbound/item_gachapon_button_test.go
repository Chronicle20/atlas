package serverbound

import (
	"context"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v79 ida=0x8efda6
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v83 ida=0x99a9a7
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v84 ida=0x9db82d
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v87 ida=0xa215f6
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v92 ida=0x944110
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=gms_v95 ida=0x96aa40
// packet-audit:verify packet=cash/serverbound/CashItemGachaponButton version=jms_v185 ida=0xa6e309
//
// The client sends only EncodeBuffer(&m_liItemSN, 8) — a little-endian
// int64 cash serial. Identical on every version above (design.md §1.2), so
// one fixture covers all of them plus every pt.Variants tenant exercised
// below. gms_v48/v61/v72 have no CUICashItemGachapon in the binary and are
// n-a — no marker.
func TestCashItemGachaponButtonRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashItemGachaponButton(1234567890)
			output := CashItemGachaponButton{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CashId() != 1234567890 {
				t.Errorf("cashId = %d, want 1234567890", output.CashId())
			}
		})
	}
}

// TestCashItemGachaponButtonDecode pins the exact wire bytes: an 8-byte
// little-endian int64 cash serial, matching EncodeBuffer(&m_liItemSN, 8).
func TestCashItemGachaponButtonDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// 1234567890 as a little-endian int64.
	raw, err := hex.DecodeString("d202964900000000")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	req := request.Request(raw)
	r := request.NewRequestReader(&req, 0)
	m := CashItemGachaponButton{}
	m.Decode(l, context.Background())(&r, map[string]interface{}{})
	if m.CashId() != 1234567890 {
		t.Fatalf("cashId = %d, want 1234567890", m.CashId())
	}
}

func TestCashItemGachaponButtonOperation(t *testing.T) {
	if NewCashItemGachaponButton(1).Operation() != CashItemGachaponHandle {
		t.Fatal("Operation must return CashItemGachaponHandle")
	}
}
