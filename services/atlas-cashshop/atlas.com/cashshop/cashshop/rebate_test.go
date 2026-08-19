package cashshop

// TestRebate proves REQUEST_LOCKER_REBATE (task-240 task 11): refunding a
// locker asset's purchase price and removing it, atomically and idempotently
// via the ledger.Claim-first ordering described on RebateAndEmit.

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/wallet"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// rebateTestDatabase mirrors purchaseTestDatabase (processor_test.go), adding
// the shared idempotency table ledger.Claim writes through.
func rebateTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration, wallet.Migration, purchaserecord.Migration, database.IdempotencyMigration, outbox.Migration)
}

func seedRebateAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, cashId int64, commodityId uint32, currency uint32, expiration time.Time) uint32 {
	t.Helper()
	e := asset.Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        cashId,
		TemplateId:    5000000,
		CommodityId:   commodityId,
		Currency:      currency,
		Quantity:      1,
		PurchasedBy:   1,
		Expiration:    expiration,
		CreatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(&e).Error)
	return e.Id
}

// rebateOutboxEntries reads the outbox rows for topic -- callers that first
// called captureDirectPurchaseEvents must pass testPurchaseStatusTopic (the
// env override changes what EVENT_TOPIC_CASH_SHOP_STATUS *resolves to*, and
// EnqueueBuffer persists the RESOLVED topic, not the env var name).
func rebateOutboxEntries(t *testing.T, db *gorm.DB, topic string) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", topic).Find(&rows).Error)
	return rows
}

func decodeRebateEvents(t *testing.T, entries []outbox.Entity) []cashshop.StatusEvent[cashshop.LockerRebatedBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.LockerRebatedBody]
	for _, e := range entries {
		var ev cashshop.StatusEvent[cashshop.LockerRebatedBody]
		if err := json.Unmarshal(e.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == cashshop.StatusEventTypeLockerRebated {
			out = append(out, ev)
		}
	}
	return out
}

func TestRebate(t *testing.T) {
	tenantId := uuid.New()
	accountId := uint32(42)
	characterId := uint32(1000)
	cashId := int64(900001)
	commodityId := uint32(10000)
	price := uint32(1200)

	t.Run("refunds the commodity price", func(t *testing.T) {
		db := rebateTestDatabase(t)
		events := captureDirectPurchaseEvents(t)
		startPurchaseCommodityServer(t, commodityId, testPurchaseItemId, price)
		compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
		seedPurchaseWallet(t, db, tenantId, accountId, 0)
		seedRebateAsset(t, db, tenantId, compartmentId, cashId, commodityId, 0, time.Now().Add(30*24*time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, transactionId)
		require.NoError(t, err)

		// asset gone from the compartment
		ccm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(accountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, ccm.Assets(), "the rebated asset must be gone from the compartment")

		// wallet credited
		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, price, w.Credit(), "the default (currency 0 -> credit/NX) bucket must be credited the commodity price")

		entries := rebateOutboxEntries(t, db, testPurchaseStatusTopic)
		require.Len(t, entries, 1)
		evs := decodeRebateEvents(t, entries)
		require.Len(t, evs, 1)
		require.Equal(t, int32(price), evs[0].Body.Amount)
		require.Equal(t, cashId, evs[0].Body.CashId)
		require.Equal(t, transactionId, evs[0].Body.TransactionId)

		require.Empty(t, purchaseErrorEvents(t, events), "a successful rebate must not also emit an ERROR")

		t.Run("replay refunds once", func(t *testing.T) {
			err := NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, transactionId)
			require.NoError(t, err)

			w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
			require.NoError(t, err)
			require.Equal(t, price, w.Credit(), "a replay must not credit the wallet a second time")

			ccm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(accountId, compartment.TypeExplorer)
			require.NoError(t, err)
			require.Empty(t, ccm.Assets())

			require.Len(t, rebateOutboxEntries(t, db, testPurchaseStatusTopic), 1, "a replay must not emit a second event")
		})

		t.Run("the purchase record survives the rebate", func(t *testing.T) {
			// The rebated asset never went through purchaserecord.Record (this
			// is a direct entity seed, not a Purchase), so seed the record
			// directly to prove RebateAndEmit does not touch it (C4).
			require.NoError(t, purchaserecord.Record(db, tenantId, accountId, commodityId))
			count, err := purchaserecord.Get(db, tenantId, accountId, commodityId)
			require.NoError(t, err)
			require.Equal(t, uint32(1), count)

			require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, uuid.New()))

			count, err = purchaserecord.Get(db, tenantId, accountId, commodityId)
			require.NoError(t, err)
			require.Equal(t, uint32(1), count, "purchaserecord is a historical fact and must survive the rebate")
		})
	})

	t.Run("a new transaction id on a gone asset is rejected", func(t *testing.T) {
		db := rebateTestDatabase(t)
		events := captureDirectPurchaseEvents(t)
		startPurchaseCommodityServer(t, commodityId, testPurchaseItemId, price)
		compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
		seedPurchaseWallet(t, db, tenantId, accountId, 0)
		seedRebateAsset(t, db, tenantId, compartmentId, cashId, commodityId, 0, time.Now().Add(30*24*time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()

		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, uuid.New()))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, price, w.Credit())

		events.Reset()
		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, uuid.New()))

		w, err = wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, price, w.Credit(), "a second rebate for a gone asset must not credit the wallet again")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationRebate, errs[0].Body.Operation)
		require.Equal(t, "unknown_error", errs[0].Body.Error)
	})

	t.Run("an asset owned by another account is rejected", func(t *testing.T) {
		db := rebateTestDatabase(t)
		events := captureDirectPurchaseEvents(t)
		otherCashId := int64(900002)
		otherCompartmentId := seedPurchaseCompartment(t, db, tenantId, 99, 55)
		seedPurchaseWallet(t, db, tenantId, accountId, 0)
		seedRebateAsset(t, db, tenantId, otherCompartmentId, otherCashId, commodityId, 0, time.Now().Add(30*24*time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()

		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, otherCashId, uuid.New()))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(0), w.Credit(), "requesting account's wallet must not change")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationRebate, errs[0].Body.Operation)
	})

	t.Run("an expired asset is rejected", func(t *testing.T) {
		db := rebateTestDatabase(t)
		events := captureDirectPurchaseEvents(t)
		expiredCashId := int64(900003)
		compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
		seedPurchaseWallet(t, db, tenantId, accountId, 0)
		seedRebateAsset(t, db, tenantId, compartmentId, expiredCashId, commodityId, 0, time.Now().Add(-time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()

		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, expiredCashId, uuid.New()))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(0), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationRebate, errs[0].Body.Operation)
	})

	t.Run("an asset with no commodity id is rejected", func(t *testing.T) {
		db := rebateTestDatabase(t)
		events := captureDirectPurchaseEvents(t)
		giftCashId := int64(900004)
		compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
		seedPurchaseWallet(t, db, tenantId, accountId, 0)
		seedRebateAsset(t, db, tenantId, compartmentId, giftCashId, 0, 0, time.Now().Add(30*24*time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()

		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, giftCashId, uuid.New()))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(0), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationRebate, errs[0].Body.Operation)
	})

	// TestRebate/the currency actually round-trips proves C1: a rebate
	// credits the SAME bucket the asset's Currency column names, not the
	// default -- pinning the exact defect the controller correction closes
	// (Purchase used to debit a bucket and then discard which one).
	t.Run("the currency actually round-trips", func(t *testing.T) {
		db := rebateTestDatabase(t)
		pointsCashId := int64(900005)
		compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
		require.NoError(t, db.Create(&wallet.Entity{Id: uuid.New(), TenantId: tenantId, AccountId: accountId, Credit: 0, Points: 0}).Error)
		startPurchaseCommodityServer(t, commodityId, testPurchaseItemId, price)
		// currency 2 = Maple Points (wallet.Model.Balance's convention), a
		// NON-default bucket -- distinct from the credit/NX bucket the
		// other subtests exercise via currency 0.
		seedRebateAsset(t, db, tenantId, compartmentId, pointsCashId, commodityId, 2, time.Now().Add(30*24*time.Hour))

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()

		require.NoError(t, NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, pointsCashId, uuid.New()))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(0), w.Credit(), "the credit/NX bucket must NOT be touched")
		require.Equal(t, price, w.Points(), "the Points bucket recorded on the asset must be the one credited")
	})
}
