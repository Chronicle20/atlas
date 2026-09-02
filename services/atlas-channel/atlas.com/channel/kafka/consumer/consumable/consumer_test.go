package consumable

import (
	"testing"

	consumable2 "atlas-channel/kafka/message/consumable"
)

// TestConsumableErrorAction pins the ERROR-event routing table.
// task-280 FR-7: POTION_LOCKED must be RECOGNIZED -- it may not be served by
// the catch-all, whose response is free to change when a future error type
// needs a different default. The remaining rows are regression pins: the
// extraction of consumableErrorAction out of handleErrorConsumableEvent must
// not move any existing type to a different arm.
func TestConsumableErrorAction(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		want      errorAction
	}{
		{"pet cannot consume", consumable2.ErrorTypePetCannotConsume, actionPetCashFoodError},
		{"inventory full", consumable2.ErrorTypeInventoryFull, actionInventoryFull},
		{"vega invalid", consumable2.ErrorTypeVegaInvalid, actionVegaInvalid},
		{"potion locked", consumable2.ErrorTypePotionLocked, actionUnstick},
		{"consume failed falls through", "CONSUME_FAILED", actionUnstick},
		{"empty falls through", "", actionUnstick},
		{"unrecognized falls through", "SOMETHING_ELSE", actionUnstick},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := consumableErrorAction(tt.errorType); got != tt.want {
				t.Fatalf("consumableErrorAction(%q) = %v, want %v", tt.errorType, got, tt.want)
			}
		})
	}
}

// The wire value must match atlas-consumables' hand-mirrored copy exactly;
// a typo in either would silently route POTION_LOCKED to the catch-all.
func TestPotionLockedWireValue(t *testing.T) {
	if consumable2.ErrorTypePotionLocked != "POTION_LOCKED" {
		t.Errorf("ErrorTypePotionLocked = %q, want \"POTION_LOCKED\"", consumable2.ErrorTypePotionLocked)
	}
}
