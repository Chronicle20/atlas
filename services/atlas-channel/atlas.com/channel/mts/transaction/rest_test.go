package transaction

import (
	"testing"
	"time"
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
	// A sale renders as "Sold" (contract-history code 0).
	if item.ProcessStatus() != 0 {
		t.Errorf("sale nProcessStatus = %d, want 0 (Sold)", item.ProcessStatus())
	}
}

// TestToMtsItemProcessStatus pins the buy/sell disposition code the v83 client
// renders in the My Page -> History status column (CITCWnd_List::GetContractHistoryCode,
// IDA-verified): a purchase is 1 ("Purchased"), a sale is 0 ("Sold").
func TestToMtsItemProcessStatus(t *testing.T) {
	created := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)

	buy, err := Extract(RestModel{ItemId: 1302000, Quantity: 1, TotalPrice: 1500, Kind: "purchase", CreatedAt: created})
	if err != nil {
		t.Fatalf("extract purchase: %v", err)
	}
	if got := ToMtsItem(buy).ProcessStatus(); got != 1 {
		t.Errorf("purchase nProcessStatus = %d, want 1 (Purchased)", got)
	}

	sale, err := Extract(RestModel{ItemId: 1302000, Quantity: 1, TotalPrice: 1500, Kind: "sale", CreatedAt: created})
	if err != nil {
		t.Fatalf("extract sale: %v", err)
	}
	if got := ToMtsItem(sale).ProcessStatus(); got != 0 {
		t.Errorf("sale nProcessStatus = %d, want 0 (Sold)", got)
	}
}

// TestResource asserts the read endpoint path template matches atlas-mts's
// GET /characters/{characterId}/mts/transactions.
func TestResource(t *testing.T) {
	if Resource != "characters/%d/mts/transactions" {
		t.Errorf("Resource = %q, want characters/%%d/mts/transactions", Resource)
	}
}
