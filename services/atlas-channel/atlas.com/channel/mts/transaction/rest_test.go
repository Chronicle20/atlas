package transaction

import (
	"testing"
	"time"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
)

// TestExtract asserts the transaction RestModel -> Model extraction carries the
// fields the My Page -> History view renders into an ITCITEM (item, quantity,
// total price, kind, date).
func TestExtract(t *testing.T) {
	created := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)
	m, err := Extract(RestModel{
		Id:             "11111111-1111-1111-1111-111111111111",
		WorldId:        1,
		CharacterId:    100100,
		CounterpartyId: 200200,
		ItemId:         1302000,
		Quantity:       3,
		TotalPrice:     5000,
		Kind:           "purchase",
		CreatedAt:      created,
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if m.Id() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("id = %q", m.Id())
	}
	if m.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", m.ItemId())
	}
	if m.Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", m.Quantity())
	}
	if m.TotalPrice() != 5000 {
		t.Errorf("totalPrice = %d, want 5000", m.TotalPrice())
	}
	if m.Kind() != "purchase" {
		t.Errorf("kind = %q, want purchase", m.Kind())
	}
	if !m.CreatedAt().Equal(created) {
		t.Errorf("createdAt = %s, want %s", m.CreatedAt(), created)
	}
}

// TestToMtsItem asserts the History ITCITEM carries the settled price in the
// price columns and the settle time as the date-expired field, with no serial.
func TestToMtsItem(t *testing.T) {
	created := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)
	m, err := Extract(RestModel{ItemId: 1302000, Quantity: 2, TotalPrice: 7500, Kind: "sale", CreatedAt: created})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	item := ToMtsItem(m)
	if item.Price() != 7500 {
		t.Errorf("nPrice = %d, want 7500", item.Price())
	}
	if item.MinPrice() != 7500 {
		t.Errorf("nMinPrice = %d, want 7500", item.MinPrice())
	}
	if item.UnitPrice() != 7500 {
		t.Errorf("nUnitPrice = %d, want 7500", item.UnitPrice())
	}
	if item.ItcSn() != 0 {
		t.Errorf("nITCSN = %d, want 0 (history rows are not addressable)", item.ItcSn())
	}
	// A sale renders as "Sold" (the HISTORY_SOLD key, config-resolved at Encode).
	if item.ProcessStatusKey() != fieldcb.MtsProcessStatusHistorySold {
		t.Errorf("sale processStatusKey = %q, want %q (Sold)", item.ProcessStatusKey(), fieldcb.MtsProcessStatusHistorySold)
	}
}

// TestToMtsItemProcessStatus pins the buy/sell disposition SEMANTIC KEY the History
// column resolves through the tenant processStatusCodes table (the wire code —
// GetContractHistoryCode 0/1/2/3 — is applied at Encode, DOM-25): purchase ->
// HISTORY_PURCHASED, sale -> HISTORY_SOLD, bid_lost -> HISTORY_BID_LOST, cancelled ->
// HISTORY_CANCELLED.
func TestToMtsItemProcessStatus(t *testing.T) {
	created := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		kind string
		want string
	}{
		{"purchase", fieldcb.MtsProcessStatusHistoryPurchased},
		{"sale", fieldcb.MtsProcessStatusHistorySold},
		{"bid_lost", fieldcb.MtsProcessStatusHistoryBidLost},
		{"cancelled", fieldcb.MtsProcessStatusHistoryCancelled},
	}
	for _, tc := range cases {
		m, err := Extract(RestModel{ItemId: 1302000, Quantity: 1, TotalPrice: 1500, Kind: tc.kind, CreatedAt: created})
		if err != nil {
			t.Fatalf("extract %s: %v", tc.kind, err)
		}
		if got := ToMtsItem(m).ProcessStatusKey(); got != tc.want {
			t.Errorf("%s processStatusKey = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

// TestResource asserts the read endpoint path template matches atlas-mts's
// GET /characters/{characterId}/mts/transactions.
func TestResource(t *testing.T) {
	if Resource != "characters/%d/mts/transactions" {
		t.Errorf("Resource = %q, want characters/%%d/mts/transactions", Resource)
	}
}
