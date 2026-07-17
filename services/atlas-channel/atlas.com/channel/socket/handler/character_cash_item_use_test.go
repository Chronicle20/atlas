package handler

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

// TestGetCashSlotItemType_SealingLock_VersionShifted pins the version-shifted
// numbers GetCashSlotItemType returns for a 5061xxx sealing-lock imprint
// (classification 506 = item.ClassificationItemImprints): 64 on non-GMS95,
// 65 on GMS95 (character_cash_item_use.go ~line 397-407).
func TestGetCashSlotItemType_SealingLock_VersionShifted(t *testing.T) {
	sealingLockItemId := item.Id(5061000)

	nonGMS95 := mustTenant(t, "GMS", 87, 1)
	if got := GetCashSlotItemType(nonGMS95)(sealingLockItemId); got != CashSlotItemTypeSealTimed {
		t.Errorf("non-GMS95 sealing lock: got %v, want CashSlotItemTypeSealTimed(%v)", got, CashSlotItemTypeSealTimed)
	}

	gms95 := mustTenant(t, "GMS", 95, 1)
	if got := GetCashSlotItemType(gms95)(sealingLockItemId); got != CashSlotItemTypeSealTimedV95 {
		t.Errorf("GMS95 sealing lock: got %v, want CashSlotItemTypeSealTimedV95(%v)", got, CashSlotItemTypeSealTimedV95)
	}
}

// TestGetCashSlotItemType_CrossVersionTypeCollision documents (does not fix -
// the fix lives in the dispatch `sealTimed` selection in
// CharacterCashItemUseHandleFunc) the exact collision that made the seal arm
// unconditionally matching CashSlotItemTypeSealTimed(64)/CashSlotItemTypeSealTimedV95(65)
// unsafe:
//   - non-GMS95: a ClassificationCharacterCreation item (5430xxx-5432xxx) also
//     returns type 65 (the GMS95 seal-timed number).
//   - GMS95: a category-552 item also returns type 64 (the non-GMS95
//     seal-timed number).
//
// GetCashSlotItemType itself is version-correct (each of these values is
// exactly what that version's client switch produces); the bug was the
// handler's seal-arm dispatch matching both numbers regardless of tenant
// version. Full dispatch-level coverage (proving the handler routes a
// CharacterCreation/552 item away from the seal saga) would require mocking
// session.Model, character2.Processor, cashData.Processor and saga.Processor
// end-to-end; that scaffolding does not exist for this handler today, so
// this test is scoped to the pure GetCashSlotItemType classification that
// the fix's version-selected `sealTimed` variable relies on. Noted as a
// dispatch-level coverage gap.
func TestGetCashSlotItemType_CrossVersionTypeCollision(t *testing.T) {
	nonGMS95 := mustTenant(t, "GMS", 87, 1)
	gms95 := mustTenant(t, "GMS", 95, 1)

	// itemId/1000 == 5431 -> "else" branch of ClassificationCharacterCreation
	// (itemId.Id is uint32; itemId/1000 must be >= 5431 or the subtraction
	// underflows and always takes the >1 branch).
	characterCreationItemId := item.Id(5431000)
	if got := GetCashSlotItemType(nonGMS95)(characterCreationItemId); got != CashSlotItemType(65) {
		t.Fatalf("non-GMS95 CharacterCreation item: got %v, want 65 (collides with CashSlotItemTypeSealTimedV95)", got)
	}
	if got := GetCashSlotItemType(gms95)(characterCreationItemId); got == CashSlotItemType(65) {
		t.Fatalf("GMS95 CharacterCreation item unexpectedly returned 65 too: %v", got)
	}

	// category 552.
	category552ItemId := item.Id(5520000)
	if got := GetCashSlotItemType(gms95)(category552ItemId); got != CashSlotItemType(64) {
		t.Fatalf("GMS95 category-552 item: got %v, want 64 (collides with CashSlotItemTypeSealTimed)", got)
	}
	if got := GetCashSlotItemType(nonGMS95)(category552ItemId); got == CashSlotItemType(64) {
		t.Fatalf("non-GMS95 category-552 item unexpectedly returned 64 too: %v", got)
	}
}

// TestIsPigmyEgg pins the incubatable Pigmy Egg id range (4170000-4170009)
// that the incubator arm re-validates server-side so a crafted request
// cannot sacrifice an arbitrary item.
func TestIsPigmyEgg(t *testing.T) {
	cases := map[item.Id]bool{
		4169999: false, 4170000: true, 4170005: true, 4170009: true, 4170010: false, 2000000: false,
	}
	for id, want := range cases {
		if got := isPigmyEgg(id); got != want {
			t.Errorf("isPigmyEgg(%d) = %v, want %v", id, got, want)
		}
	}
}

func TestGetCashSlotItemTypeVegasSpell(t *testing.T) {
	pre95 := mustTenant(t, "GMS", 83, 1)
	v95 := mustTenant(t, "GMS", 95, 1)
	jms := mustTenant(t, "JMS", 185, 1)

	cases := []struct {
		name string
		tn   tenant.Model
		id   item.Id
		want CashSlotItemType
	}{
		{"v83 vega 10", pre95, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
		{"v83 vega 60", pre95, item.VegasSpell60, CashSlotItemTypeVegasSpellPre95},
		{"v95 vega 10", v95, item.VegasSpell10, CashSlotItemTypeVegasSpell95},
		{"v95 vega 60", v95, item.VegasSpell60, CashSlotItemTypeVegasSpell95},
		{"jms vega 10", jms, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCashSlotItemType(tc.tn)(tc.id); got != tc.want {
				t.Errorf("GetCashSlotItemType(%d) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}
