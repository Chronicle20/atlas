package consumable

import (
	"testing"

	"atlas-consumables/asset"

	"github.com/google/uuid"
)

func equipAsset(templateId uint32, hammersApplied uint32) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetHammersApplied(hammersApplied).
		Build()
}

func TestViciousHammerErrorCodeEligible(t *testing.T) {
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 7, false); code != 0 {
		t.Errorf("eligible target: got code %d, want 0", code)
	}
	if code := viciousHammerErrorCode(equipAsset(1302000, 1), 7, false); code != 0 {
		t.Errorf("one hammer applied: got code %d, want 0", code)
	}
}

func TestViciousHammerErrorCodeCapReached(t *testing.T) {
	// IDA-verified cap: error 2 = "2 upgrade increases have been used already".
	if code := viciousHammerErrorCode(equipAsset(1302000, 2), 7, false); code != ViciousHammerErrorCapReached {
		t.Errorf("cap reached: got code %d, want %d", code, ViciousHammerErrorCapReached)
	}
	if code := viciousHammerErrorCode(equipAsset(1302000, 3), 7, false); code != ViciousHammerErrorCapReached {
		t.Errorf("above cap: got code %d, want %d", code, ViciousHammerErrorCapReached)
	}
}

func TestViciousHammerErrorCodeNotUpgradable(t *testing.T) {
	// WZ tuc == 0 -> client notice 1 "The item is not upgradable".
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 0, false); code != ViciousHammerErrorNotUpgradable {
		t.Errorf("zero-slot equip: got code %d, want %d", code, ViciousHammerErrorNotUpgradable)
	}
	// Cash equips are excluded.
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 7, true); code != ViciousHammerErrorNotUpgradable {
		t.Errorf("cash equip: got code %d, want %d", code, ViciousHammerErrorNotUpgradable)
	}
}

func TestViciousHammerErrorCodeHorntail(t *testing.T) {
	// 1122000 = Horntail Necklace (WZ String.wz/Eqp.img.xml, GMS 83.1). It has
	// tuc=3, so the exclusion must fire on the id, not the slot count.
	if code := viciousHammerErrorCode(equipAsset(1122000, 0), 3, false); code != ViciousHammerErrorHorntail {
		t.Errorf("horntail necklace: got code %d, want %d", code, ViciousHammerErrorHorntail)
	}
}
