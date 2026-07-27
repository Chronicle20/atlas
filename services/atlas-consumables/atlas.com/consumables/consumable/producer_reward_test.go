package consumable

import (
	"testing"

	consumable2 "atlas-consumables/kafka/message/consumable"
)

func TestRewardEventTypeConstants(t *testing.T) {
	if consumable2.EventTypeRewardEffect != "REWARD_EFFECT" {
		t.Errorf("EventTypeRewardEffect = %q", consumable2.EventTypeRewardEffect)
	}
	if consumable2.EventTypeRewardWon != "REWARD_WON" {
		t.Errorf("EventTypeRewardWon = %q", consumable2.EventTypeRewardWon)
	}
	if consumable2.ErrorTypeInventoryFull != "INVENTORY_FULL" {
		t.Errorf("ErrorTypeInventoryFull = %q", consumable2.ErrorTypeInventoryFull)
	}
}

func TestRewardEventProvidersProduceOneMessage(t *testing.T) {
	if msgs, err := RewardEffectEventProvider(7, 2022309, "Effect/BasicEff/Event1/Good")(); err != nil || len(msgs) != 1 {
		t.Fatalf("RewardEffectEventProvider: msgs=%d err=%v", len(msgs), err)
	}
	if msgs, err := RewardWonEventProvider(7, 2022309, 1132010, "Hero got Belt")(); err != nil || len(msgs) != 1 {
		t.Fatalf("RewardWonEventProvider: msgs=%d err=%v", len(msgs), err)
	}
}
