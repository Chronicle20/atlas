package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func mustTenant(t *testing.T, region string, majorVersion uint16) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, majorVersion, 1)
	if err != nil {
		t.Fatalf("unable to create tenant: %v", err)
	}
	return tm
}

// TestGetCashSlotItemType_SealingLock_VersionShifted pins the version-shifted
// numbers GetCashSlotItemType returns for a 5061xxx sealing-lock imprint
// (classification 506 = item.ClassificationItemImprints): 64 on non-GMS95,
// 65 on GMS95 (character_cash_item_use.go ~line 397-407).
func TestGetCashSlotItemType_SealingLock_VersionShifted(t *testing.T) {
	sealingLockItemId := item.Id(5061000)

	nonGMS95 := mustTenant(t, "GMS", 87)
	if got := GetCashSlotItemType(nonGMS95)(sealingLockItemId); got != CashSlotItemTypeSealTimed {
		t.Errorf("non-GMS95 sealing lock: got %v, want CashSlotItemTypeSealTimed(%v)", got, CashSlotItemTypeSealTimed)
	}

	gms95 := mustTenant(t, "GMS", 95)
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
	nonGMS95 := mustTenant(t, "GMS", 87)
	gms95 := mustTenant(t, "GMS", 95)

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
