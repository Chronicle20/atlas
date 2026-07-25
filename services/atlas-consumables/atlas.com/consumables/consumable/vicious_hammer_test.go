package consumable

import (
	"atlas-consumables/asset"
	"testing"

	"github.com/google/uuid"
)

func equipAsset(templateId uint32, hammersApplied uint32) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetHammersApplied(hammersApplied).
		Build()
}

func TestViciousHammerReasonEligible(t *testing.T) {
	if reason := viciousHammerReason(equipAsset(1302000, 0), 7, false); reason != "" {
		t.Errorf("eligible target: got reason %q, want \"\"", reason)
	}
	if reason := viciousHammerReason(equipAsset(1302000, 1), 7, false); reason != "" {
		t.Errorf("one hammer applied: got reason %q, want \"\"", reason)
	}
}

func TestViciousHammerReasonCapReached(t *testing.T) {
	// IDA-verified cap: error 2 = "2 upgrade increases have been used already".
	if reason := viciousHammerReason(equipAsset(1302000, 2), 7, false); reason != ViciousHammerReasonCapReached {
		t.Errorf("cap reached: got reason %q, want %q", reason, ViciousHammerReasonCapReached)
	}
	if reason := viciousHammerReason(equipAsset(1302000, 3), 7, false); reason != ViciousHammerReasonCapReached {
		t.Errorf("above cap: got reason %q, want %q", reason, ViciousHammerReasonCapReached)
	}
}

func TestViciousHammerReasonNotUpgradable(t *testing.T) {
	// WZ tuc == 0 -> client notice 1 "The item is not upgradable".
	if reason := viciousHammerReason(equipAsset(1302000, 0), 0, false); reason != ViciousHammerReasonNotUpgradable {
		t.Errorf("zero-slot equip: got reason %q, want %q", reason, ViciousHammerReasonNotUpgradable)
	}
	// Cash equips are excluded.
	if reason := viciousHammerReason(equipAsset(1302000, 0), 7, true); reason != ViciousHammerReasonNotUpgradable {
		t.Errorf("cash equip: got reason %q, want %q", reason, ViciousHammerReasonNotUpgradable)
	}
}

func TestViciousHammerReasonHorntail(t *testing.T) {
	// 1122000 = Horntail Necklace (WZ String.wz/Eqp.img.xml, GMS 83.1). It has
	// tuc=3, so the exclusion must fire on the id, not the slot count.
	if reason := viciousHammerReason(equipAsset(1122000, 0), 3, false); reason != ViciousHammerReasonHorntail {
		t.Errorf("horntail necklace: got reason %q, want %q", reason, ViciousHammerReasonHorntail)
	}
}
