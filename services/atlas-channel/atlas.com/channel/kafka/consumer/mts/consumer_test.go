package mts

import (
	"context"
	"testing"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	"github.com/sirupsen/logrus"
)

// testOptions mirrors the tenant writer options shape ResolveCode consumes:
// options["operations"][KEY] = mode (JSON numbers decode as float64). Modes are
// the IDA-verified v83 values; the byte assertions below pin the WIRE bytes the
// failure routing produces, not just which body func was picked.
func testOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"GET_SEARCH_ITC_LIST_FAILED": float64(24),
			"BUY_ITEM_FAILED":            float64(52),
			"BID_AUCTION_FAILED":         float64(60),
		},
	}
}

func encode(t *testing.T, body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) []byte {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return body(l, context.Background())(testOptions())
}

// TestFailNoticeOrRoutesReason pins the descriptive-failure routing: a
// non-zero NoticeFailReason code becomes the reason-notice arm (mode 24 +
// reason byte — the client shows the matching NoticeFailReason string), while
// reason 0 keeps the operation's bare *Failed arm (mode byte only).
func TestFailNoticeOrRoutesReason(t *testing.T) {
	// Reason 66 ('B' — "You do not have enough NX") on a failed buy.
	got := encode(t, failNoticeOr(66, fieldpkt.MtsOperationBuyItemFailedBody()))
	if len(got) != 2 || got[0] != 24 || got[1] != 66 {
		t.Fatalf("reason-66 bytes = %v, want [24 66]", got)
	}

	// Reason 81 ('Q' — "The item has been sold") on a failed bid.
	got = encode(t, failNoticeOr(81, fieldpkt.MtsOperationBidAuctionFailedBody()))
	if len(got) != 2 || got[0] != 24 || got[1] != 81 {
		t.Fatalf("reason-81 bytes = %v, want [24 81]", got)
	}

	// Reason 0: unchanged legacy behavior — the bare per-operation arm.
	got = encode(t, failNoticeOr(0, fieldpkt.MtsOperationBuyItemFailedBody()))
	if len(got) != 1 || got[0] != 52 {
		t.Fatalf("reason-0 buy bytes = %v, want [52]", got)
	}
	got = encode(t, failNoticeOr(0, fieldpkt.MtsOperationBidAuctionFailedBody()))
	if len(got) != 1 || got[0] != 60 {
		t.Fatalf("reason-0 bid bytes = %v, want [60]", got)
	}
}
