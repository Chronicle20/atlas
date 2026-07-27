package mts

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mtsmsg "atlas-channel/kafka/message/mts"

	"github.com/sirupsen/logrus"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
)

// TestReasonFieldNoDecodeCollision guards the EVENT_TOPIC_MTS_STATUS reason-tag
// discipline (task-102, DOM-25): every handler decodes every message, so ALL
// failure events must carry their semantic key under the SAME JSON tag "reasonKey"
// (string) — never a bare numeric "reason". An earlier design had cancel/take-home
// on a numeric "reason" tag, which collided with buy/bid's string "reasonKey" on
// the shared topic (a string-vs-number mismatch on one tag) and dropped messages.
func TestReasonFieldNoDecodeCollision(t *testing.T) {
	// Cancel/take-home failure bodies MUST serialize "reasonKey" (string), never a
	// bare "reason" — this is the exact regression the DOM-25 migration fixed.
	for _, b := range [][]byte{
		mustMarshal(t, mtsmsg.StatusEventListingCancelFailedBody{ReasonKey: "ITEM_SOLD"}),
		mustMarshal(t, mtsmsg.StatusEventTakeHomeFailedBody{ReasonKey: "ITEM_SOLD"}),
	} {
		if !strings.Contains(string(b), `"reasonKey":"ITEM_SOLD"`) {
			t.Fatalf("failure body must serialize a string reasonKey, got %s", b)
		}
		if strings.Contains(string(b), `"reason":`) {
			t.Fatalf("failure body must NOT carry a bare numeric reason tag, got %s", b)
		}
	}

	// Cross-decode safety: a BUY_FAILED event decoded by a cancel-failed body (every
	// handler decodes every message) must not error — the shared "reasonKey" tag has
	// one type across all events, and the handler's e.Type guard (not the decode) is
	// what scopes it to its own event.
	buyFailed := []byte(`{"transactionId":"00000000-0000-0000-0000-000000000000","type":"BUY_FAILED","body":{"worldId":0,"serial":5,"buyerId":1,"reasonKey":"NOT_ENOUGH_NX"}}`)
	var asCancel mtsmsg.StatusEvent[mtsmsg.StatusEventListingCancelFailedBody]
	if err := json.Unmarshal(buyFailed, &asCancel); err != nil {
		t.Fatalf("cross-decode of a reasonKey event must not error: %v", err)
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

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
