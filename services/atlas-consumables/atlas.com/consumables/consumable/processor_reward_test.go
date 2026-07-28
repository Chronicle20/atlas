package consumable

import (
	"testing"

	consumable3 "atlas-consumables/data/consumable"
)

// validateRewardTable is the pre-reserve guard used by RequestItemReward.
func TestValidateRewardTable(t *testing.T) {
	if err := validateRewardTable(nil); err == nil {
		t.Fatal("empty table must be rejected")
	}
	if err := validateRewardTable([]consumable3.RewardModel{rw(1, 1, 0)}); err == nil {
		t.Fatal("zero total prob must be rejected")
	}
	if err := validateRewardTable([]consumable3.RewardModel{rw(1, 1, 5)}); err != nil {
		t.Fatalf("valid table rejected: %v", err)
	}
}

// grantQuantity clamps count=0 up to 1 (design §5.4).
func TestGrantQuantity(t *testing.T) {
	if grantQuantity(0) != 1 {
		t.Fatalf("count 0 → %d, want 1", grantQuantity(0))
	}
	if grantQuantity(5) != 5 {
		t.Fatalf("count 5 → %d, want 5", grantQuantity(5))
	}
}
