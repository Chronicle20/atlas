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
		{"solomon no experience", consumable2.ErrorTypeSolomonNoExperience, actionSolomonRejected},
		{"solomon level exceeded", consumable2.ErrorTypeSolomonLevelExceeded, actionSolomonRejected},
		{"solomon balance not empty", consumable2.ErrorTypeSolomonBalanceNotEmpty, actionSolomonRejected},
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

// The wire values must match atlas-consumables' hand-mirrored copy exactly
// (kafka/message/consumable/kafka.go on both sides); a typo in either would
// silently route a Solomon rejection to the CONSUME_FAILED catch-all instead
// of its distinct message.
func TestSolomonErrorWireValues(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{consumable2.ErrorTypeSolomonNoExperience, "SOLOMON_NO_EXPERIENCE"},
		{consumable2.ErrorTypeSolomonLevelExceeded, "SOLOMON_LEVEL_EXCEEDED"},
		{consumable2.ErrorTypeSolomonBalanceNotEmpty, "SOLOMON_BALANCE_NOT_EMPTY"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

// TestSolomonRejectionMessage pins each error type to its own message text
// and confirms an unrecognized type falls back to empty (never reached in
// practice -- consumableErrorAction only routes the three known types to
// actionSolomonRejected).
func TestSolomonRejectionMessage(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		want      string
	}{
		{"no experience", consumable2.ErrorTypeSolomonNoExperience, SolomonNoExperienceMessage},
		{"level exceeded", consumable2.ErrorTypeSolomonLevelExceeded, SolomonLevelExceededMessage},
		{"balance not empty", consumable2.ErrorTypeSolomonBalanceNotEmpty, SolomonBalanceNotEmptyMessage},
		{"unrecognized", "SOMETHING_ELSE", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := solomonRejectionMessage(tt.errorType); got != tt.want {
				t.Errorf("solomonRejectionMessage(%q) = %q, want %q", tt.errorType, got, tt.want)
			}
		})
	}
}
