package cashshop

// TestPurchaseInventoryIncreaseByItem* are regression tests for Defect B
// (bug-cash-shop-live-testing.md): PurchaseInventoryIncreaseByItemAndEmit
// computed the inventory type as `ci.ItemId() - 9110000/1000` -- Go binds
// `/` tighter than `-`, so this evaluated `ci.ItemId() - 9110`, not
// `(ci.ItemId() - 9110000) / 1000`. For itemId 9114000 the intended type is
// 4 (inventory.TypeValueETC); the actual value narrowed to -6, which has no
// compartment, so the wallet was still debited and an
// InventoryCapacityIncreased event fired for a type the client cannot
// parse -- disconnecting it after the cash was already taken.
//
// The character inventory lookup (character/inventory, INVENTORY_SERVICE_URL)
// is not stubbed here: RootUrlFor errors on the unset env var, and
// (character.ProcessorImpl).InventoryDecorator swallows that error and
// returns the character model with an empty compartment map -- so
// CompartmentByType reads capacity 0 regardless, exactly like the live
// pod logs. IncreaseCapacity is a local Kafka-buffer write
// (character/compartment.ProcessorImpl), not a remote call, so it needs no
// stub either.

import (
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/wallet"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// TestPurchaseInventoryIncreaseByItemComputesETCType proves itemId 9114000
// (the reported ETC slot-expansion item) now resolves to inventory type 4
// (inventory.TypeValueETC) and the purchase succeeds, mirroring the "Expected"
// section of Defect B.
func TestPurchaseInventoryIncreaseByItemComputesETCType(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(50200095)
	price := uint32(6800)

	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, 9114000, price)
	seedPurchaseWallet(t, db, tenantId, accountId, 96000)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByItemAndEmit(characterId, 1, serialNumber)
	require.NoError(t, err)

	var w wallet.Entity
	require.NoError(t, db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).First(&w).Error)
	require.Equal(t, uint32(96000-6800), w.Credit, "wallet must be debited once the compartment type resolves correctly")

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 1)
	var ev cashshop.StatusEvent[cashshop.InventoryCapacityIncreasedBody]
	require.NoError(t, json.Unmarshal(entries[0].MessageValue, &ev))
	require.Equal(t, cashshop.StatusEventTypeInventoryCapacityIncreased, ev.Type)
	require.Equal(t, byte(4), ev.Body.InventoryType, "9114000 must resolve to inventory.TypeValueETC (4), not the pre-fix -6")
}

// TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem proves an item id
// whose computed inventory type is not one of inventory.Types is rejected
// BEFORE the wallet is touched -- the tester lost 6800 cash to this defect
// because the debit happened before the compartment was found not to exist.
func TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(50200099)
	price := uint32(6800)
	// 9104000 -> (9104000-9110000)/1000 = -6, not a member of inventory.Types.
	const outOfRangeItemId = uint32(9104000)

	events := captureDirectPurchaseEvents(t)
	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, outOfRangeItemId, price)
	seedPurchaseWallet(t, db, tenantId, accountId, 96000)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByItemAndEmit(characterId, 1, serialNumber)
	require.Error(t, err, "out-of-range inventory type must be rejected")

	var w wallet.Entity
	require.NoError(t, db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).First(&w).Error)
	require.Equal(t, uint32(96000), w.Credit, "wallet must be untouched when the compartment type does not exist")

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 0, "rejection fires on the direct producer path, not the outbox")

	errs := purchaseErrorEvents(t, events)
	require.Len(t, errs, 1)
	require.Equal(t, "UNKNOWN_ERROR", errs[0].Body.Error)
}

// TestPurchaseInventoryIncreaseByItemRejectsItemBelowBaseOffset proves an
// item id below the 9110000 inventory-type base offset is rejected before
// `ci.ItemId() - 9110000` is ever evaluated -- ItemId() and 9110000 are both
// uint32, so for itemId < 9110000 that subtraction underflows and wraps
// before truncation to inventory.Type (int8). itemId 95704 is chosen because
// the wrapped, truncated value coincidentally lands on inventory.TypeValueEquip
// (1), a member of inventory.Types -- so the post-hoc isValidInventoryType
// range check alone would NOT have caught it; the lower-bound guard added
// ahead of the subtraction must reject it instead.
func TestPurchaseInventoryIncreaseByItemRejectsItemBelowBaseOffset(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(50200098)
	price := uint32(6800)
	// 95704 < 9110000: (95704-9110000) underflows as uint32 and, divided by
	// 1000 and truncated to int8, coincidentally equals 1
	// (inventory.TypeValueEquip).
	const belowOffsetItemId = uint32(95704)

	events := captureDirectPurchaseEvents(t)
	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, belowOffsetItemId, price)
	seedPurchaseWallet(t, db, tenantId, accountId, 96000)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByItemAndEmit(characterId, 1, serialNumber)
	require.Error(t, err, "item id below the inventory-type base offset must be rejected")

	var w wallet.Entity
	require.NoError(t, db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).First(&w).Error)
	require.Equal(t, uint32(96000), w.Credit, "wallet must be untouched when the item id is below the base offset")

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 0, "rejection fires on the direct producer path, not the outbox")

	errs := purchaseErrorEvents(t, events)
	require.Len(t, errs, 1)
	require.Equal(t, "UNKNOWN_ERROR", errs[0].Body.Error)
}

// TestPurchaseInventoryIncreaseByTypeGrantsFourSlots is a regression test for
// Defect D (bug-cash-shop-live-testing-round-2.md): PurchaseInventoryIncreaseByTypeAndEmit
// hard-coded amount 8, granting double the advertised +4 slots for the 4000 NX
// by-type expansion. This asserts parity with the by-item path
// (TestPurchaseInventoryIncreaseByItemComputesETCType), which has always
// passed amount 4.
func TestPurchaseInventoryIncreaseByTypeGrantsFourSlots(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)

	startPurchaseCharacterServer(t, characterId, accountId)
	seedPurchaseWallet(t, db, tenantId, accountId, 96000)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByTypeAndEmit(characterId, 1, inventory.TypeValueUse)
	require.NoError(t, err)

	var w wallet.Entity
	require.NoError(t, db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).First(&w).Error)
	require.Equal(t, uint32(96000-4000), w.Credit, "wallet must be debited the 4000 by-type cost")

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 1)
	var ev cashshop.StatusEvent[cashshop.InventoryCapacityIncreasedBody]
	require.NoError(t, json.Unmarshal(entries[0].MessageValue, &ev))
	require.Equal(t, cashshop.StatusEventTypeInventoryCapacityIncreased, ev.Type)
	require.Equal(t, uint32(4), ev.Body.Amount, "by-type must grant +4 slots, matching the by-item path and the v83 UI's advertised amount")
	require.Equal(t, uint32(4), ev.Body.Capacity, "capacity moves by the same +4 as Amount from the stubbed character's empty compartment")
}
