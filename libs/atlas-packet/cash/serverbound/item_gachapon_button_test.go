package serverbound

import (
	"context"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// The client sends only EncodeBuffer(&m_liItemSN, 8) — a little-endian
// int64 cash serial. Identical on v79/v83/v84/v87/v92/v95/jms_v185
// (design.md §1.2), so one fixture covers every version.
// packet-audit:verify cash/serverbound/CashItemGachaponButton
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

func TestCashItemGachaponButtonRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	out := NewCashItemGachaponButton(1234567890).Encode(l, context.Background())(map[string]interface{}{})
	if got := hex.EncodeToString(out); got != "d202964900000000" {
		t.Fatalf("encoded = %s", got)
	}
	req := request.Request(out)
	r := request.NewRequestReader(&req, 0)
	var back CashItemGachaponButton
	back.Decode(l, context.Background())(&r, map[string]interface{}{})
	if back.CashId() != 1234567890 {
		t.Fatalf("round-trip cashId = %d", back.CashId())
	}
}

func TestCashItemGachaponButtonOperation(t *testing.T) {
	if NewCashItemGachaponButton(1).Operation() != CashItemGachaponHandle {
		t.Fatal("Operation must return CashItemGachaponHandle")
	}
}
