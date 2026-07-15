package merchant

import "testing"

// The byte values are a persisted + wire contract (shops.state column,
// REST/Kafka payloads); this pin prevents accidental renumbering.
func TestShopEnumWireValues(t *testing.T) {
	if ShopTypeCharacter != 1 || ShopTypeHiredMerchant != 2 {
		t.Fatalf("ShopType values changed: character=%d hired=%d", ShopTypeCharacter, ShopTypeHiredMerchant)
	}
	if ShopStateDraft != 1 || ShopStateOpen != 2 || ShopStateMaintenance != 3 || ShopStateClosed != 4 {
		t.Fatalf("ShopState values changed: %d %d %d %d", ShopStateDraft, ShopStateOpen, ShopStateMaintenance, ShopStateClosed)
	}
}
