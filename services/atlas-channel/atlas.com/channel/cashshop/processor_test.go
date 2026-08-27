package cashshop

import (
	"atlas-channel/kafka/message/cashshop"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestResolvePurchaseCurrency verifies the isPoints->currency mapping applied by
// RequestPurchase before the buy is forwarded to the wallet command. JMS cash
// buys carry isPoints but no currency on the wire (currency==0); an isPoints buy
// must be steered to MaplePoints (wallet code 2) instead of falling through to
// prepaid. The resolved currency is asserted by driving it through the same
// RequestPurchaseCommandProvider the processor emits and reading back
// RequestPurchaseCommandBody.Currency.
func TestResolvePurchaseCurrency(t *testing.T) {
	const characterId = uint32(42)
	const serial = uint32(0xAABBCCDD)

	tests := []struct {
		name         string
		isPoints     bool
		currency     uint32
		wantCurrency uint32
	}{
		{name: "JMS isPoints with no currency -> MaplePoints (2)", isPoints: true, currency: 0, wantCurrency: 2},
		{name: "no isPoints with no currency -> prepaid default (0)", isPoints: false, currency: 0, wantCurrency: 0},
		{name: "explicit currency is not overridden by isPoints", isPoints: true, currency: 5, wantCurrency: 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := resolvePurchaseCurrency(tc.isPoints, tc.currency)
			if resolved != tc.wantCurrency {
				t.Fatalf("resolvePurchaseCurrency(%v, %d) = %d, want %d", tc.isPoints, tc.currency, resolved, tc.wantCurrency)
			}

			// The resolved currency must survive into the wallet command body the
			// processor forwards.
			provider := RequestPurchaseCommandProvider(characterId, serial, resolved, uuid.Nil, "")
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("got %d messages, want 1", len(msgs))
			}

			var cmd cashshop.Command[cashshop.RequestPurchaseCommandBody]
			if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
				t.Fatalf("unmarshal command: %v", err)
			}
			if cmd.Body.Currency != tc.wantCurrency {
				t.Errorf("Body.Currency = %d, want %d", cmd.Body.Currency, tc.wantCurrency)
			}
		})
	}
}

// TestResolveOptionCurrency verifies the option->currency mapping shared by
// every BUY arm whose wire body carries option -- the ring arms
// (BUY_COUPLE/BUY_FRIENDSHIP) and the package arm (BUY_PACKAGE). On GMS >=
// 83, isPoints/currency never arrive on the wire for these arms -- the
// user's confirmation-dialog payment-method choice lives entirely in option
// (docs/tasks/task-240-cash-shop-stub-operations/derivation.md §6, D4a):
// 1 = NX Credit, 2 = Maple Point, 4 = NX Prepaid, mapping onto wallet.Model's
// currency codes 1/2/3. option == 0 falls back to resolvePurchaseCurrency.
func TestResolveOptionCurrency(t *testing.T) {
	tests := []struct {
		name         string
		option       uint32
		isPoints     bool
		currency     uint32
		wantCurrency uint32
	}{
		{name: "option 1 -> NX Credit (wallet 1)", option: 1, isPoints: false, currency: 0, wantCurrency: 1},
		{name: "option 2 -> Maple Point (wallet 2)", option: 2, isPoints: false, currency: 0, wantCurrency: 2},
		{name: "option 4 -> NX Prepaid (wallet 3)", option: 4, isPoints: false, currency: 0, wantCurrency: 3},
		{name: "option 0, isPoints true -> falls back to MaplePoints (2)", option: 0, isPoints: true, currency: 0, wantCurrency: 2},
		{name: "option 0, legacy non-zero currency -> passthrough", option: 0, isPoints: false, currency: 5, wantCurrency: 5},
		{name: "unexpected option value -> falls back to resolvePurchaseCurrency", option: 3, isPoints: false, currency: 0, wantCurrency: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := resolveOptionCurrency(tc.option, tc.isPoints, tc.currency)
			if resolved != tc.wantCurrency {
				t.Fatalf("resolveOptionCurrency(%d, %v, %d) = %d, want %d", tc.option, tc.isPoints, tc.currency, resolved, tc.wantCurrency)
			}
		})
	}
}
