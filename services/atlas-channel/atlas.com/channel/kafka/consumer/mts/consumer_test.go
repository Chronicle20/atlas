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
		"noticeFailReasons": map[string]interface{}{
			"NOT_ENOUGH_NX": float64(66),
			"ITEM_SOLD":     float64(81),
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
// semantic key present in the tenant noticeFailReasons table becomes the
// reason-notice arm (config-resolved mode 24 + config-resolved reason byte),
// while an empty key keeps the operation's bare *Failed arm (mode byte only).
func TestFailNoticeOrRoutesReason(t *testing.T) {
	// "NOT_ENOUGH_NX" -> [24 66] ("You do not have enough NX").
	got := encode(t, failNoticeOr("NOT_ENOUGH_NX", fieldpkt.MtsOperationBuyItemFailedBody()))
	if len(got) != 2 || got[0] != 24 || got[1] != 66 {
		t.Fatalf("NOT_ENOUGH_NX bytes = %v, want [24 66]", got)
	}

	// "ITEM_SOLD" -> [24 81] ("The item has been sold").
	got = encode(t, failNoticeOr("ITEM_SOLD", fieldpkt.MtsOperationBidAuctionFailedBody()))
	if len(got) != 2 || got[0] != 24 || got[1] != 81 {
		t.Fatalf("ITEM_SOLD bytes = %v, want [24 81]", got)
	}

	// Empty key: unchanged legacy behavior — the bare per-operation arm.
	got = encode(t, failNoticeOr("", fieldpkt.MtsOperationBuyItemFailedBody()))
	if len(got) != 1 || got[0] != 52 {
		t.Fatalf("empty-key buy bytes = %v, want [52]", got)
	}
	got = encode(t, failNoticeOr("", fieldpkt.MtsOperationBidAuctionFailedBody()))
	if len(got) != 1 || got[0] != 60 {
		t.Fatalf("empty-key bid bytes = %v, want [60]", got)
	}
}

// TestFailNoticeOrFallsBackWithoutTable pins the tenant-compat contract: a
// tenant whose writer options lack the noticeFailReasons table (or the
// specific key) gets the bare *Failed arm — a soft fallback, never
// ResolveCode's 99-crash path.
func TestFailNoticeOrFallsBackWithoutTable(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	noTable := map[string]interface{}{
		"operations": map[string]interface{}{
			"GET_SEARCH_ITC_LIST_FAILED": float64(24),
			"BUY_ITEM_FAILED":            float64(52),
		},
	}
	got := failNoticeOr("NOT_ENOUGH_NX", fieldpkt.MtsOperationBuyItemFailedBody())(l, context.Background())(noTable)
	if len(got) != 1 || got[0] != 52 {
		t.Fatalf("missing-table bytes = %v, want bare [52]", got)
	}

	missingKey := testOptions()
	delete(missingKey["noticeFailReasons"].(map[string]interface{}), "ITEM_SOLD")
	got = failNoticeOr("ITEM_SOLD", fieldpkt.MtsOperationBidAuctionFailedBody())(l, context.Background())(missingKey)
	if len(got) != 1 || got[0] != 60 {
		t.Fatalf("missing-key bytes = %v, want bare [60]", got)
	}
}
