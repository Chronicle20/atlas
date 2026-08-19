package cashshop

// TestPurchasePackage proves REQUEST_PACKAGE_PURCHASE (task-240 task 16):
// client modes 30 (buy-for-self) and 31 (gift) share PurchasePackageAndEmit,
// resolving one package commodity into N member commodities and delivering
// one cash_assets row per member, atomically and idempotently, charging the
// PACKAGE commodity's own price exactly once (FR-PKG-5) and checking
// capacity against the FULL member count before the wallet is ever touched
// (FR-PKG-6).

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/wallet"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// Fixture per the brief: a package commodity (serial 50000) resolving to
// item 9100000 priced at 3000, whose CashPackage entry names three member
// serial numbers, each resolving to its own item at 1200. The package price
// (3000) is deliberately far from 3 x 1200 (3600) so a wallet assertion of
// 2000 (5000 - 3000) can only pass if the PACKAGE commodity's own price was
// charged, never the sum of the members'.
const (
	pkgSerialNumber = uint32(50000)
	pkgItemId       = uint32(9100000)
	pkgPrice        = uint32(3000)

	pkgMember0SN     = uint32(10000)
	pkgMember1SN     = uint32(10001)
	pkgMember2SN     = uint32(10002)
	pkgMember0ItemId = uint32(5010000)
	pkgMember1ItemId = uint32(5010001)
	pkgMember2ItemId = uint32(5010002)
	pkgMemberPrice   = uint32(1200)
)

// packageCommodityFixture is one commodity/GetById response.
type packageCommodityFixture struct {
	itemId uint32
	price  uint32
	count  uint32
}

func defaultPackageCommodities() map[uint32]packageCommodityFixture {
	return map[uint32]packageCommodityFixture{
		pkgSerialNumber: {itemId: pkgItemId, price: pkgPrice, count: 1},
		pkgMember0SN:    {itemId: pkgMember0ItemId, price: pkgMemberPrice, count: 1},
		pkgMember1SN:    {itemId: pkgMember1ItemId, price: pkgMemberPrice, count: 1},
		pkgMember2SN:    {itemId: pkgMember2ItemId, price: pkgMemberPrice, count: 1},
	}
}

func defaultPackageCatalog() map[uint32][]uint32 {
	return map[uint32][]uint32{
		pkgItemId: {pkgMember0SN, pkgMember1SN, pkgMember2SN},
	}
}

// startPackageDataServer stubs BOTH remote atlas-data lookups
// PurchasePackageAndEmit needs -- commodity/GetById (data/commodity/items/{sn})
// and cashpackage/GetById (data/cashPackages/{itemId}) -- on the same root,
// mirroring how both resources really share DATA_SERVICE_URL in production.
// Dispatch is on the resource segment in the path, and an id absent from the
// relevant map 404s, so a test can name a genuinely unresolvable serial
// number or package id.
func startPackageDataServer(t *testing.T, commodities map[uint32]packageCommodityFixture, packages map[uint32][]uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		last := parts[len(parts)-1]
		id, err := strconv.Atoi(last)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if strings.Contains(path, "cashPackages") {
			sns, ok := packages[uint32(id)]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			snJSON, _ := json.Marshal(sns)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = fmt.Fprintf(w, `{"data":{"type":"cashPackages","id":"%d","attributes":{"serialNumbers":%s}}}`, id, string(snJSON))
			return
		}

		c, ok := commodities[uint32(id)]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"commodities","id":"%d","attributes":{"itemId":%d,"count":%d,"price":%d,"period":30}}}`, id, c.itemId, c.count, c.price)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func decodePackageEvents(t *testing.T, entries []outbox.Entity) []cashshop.StatusEvent[cashshop.PackagePurchasedBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.PackagePurchasedBody]
	for _, e := range entries {
		var ev cashshop.StatusEvent[cashshop.PackagePurchasedBody]
		if err := json.Unmarshal(e.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == cashshop.StatusEventTypePackagePurchased {
			out = append(out, ev)
		}
	}
	return out
}

func TestPurchasePackage(t *testing.T) {
	const (
		buyerCharacterId     = uint32(900)
		buyerAccountId       = uint32(1)
		recipientCharacterId = uint32(77)
		recipientAccountId   = uint32(2)
		buyerName            = "Buyer"
		recipientName        = "Recipient"
	)

	t.Run("creates one asset per member", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName)
		require.NoError(t, err)

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Len(t, assets, 3)
		var templateIds []uint32
		for _, a := range assets {
			templateIds = append(templateIds, a.TemplateId())
		}
		require.ElementsMatch(t, []uint32{pkgMember0ItemId, pkgMember1ItemId, pkgMember2ItemId}, templateIds)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(2000), w.Credit(), "5000 - 3000, the PACKAGE price, not 3 x 1200")

		entries := purchaseOutboxEntries(t, db)
		evs := decodePackageEvents(t, entries)
		require.Len(t, evs, 1)
		require.Len(t, evs[0].Body.AssetIds, 3)
	})

	t.Run("an unresolvable member creates nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		commodities := defaultPackageCommodities()
		delete(commodities, pkgMember1SN)
		startPackageDataServer(t, commodities, defaultPackageCatalog())
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Empty(t, assets)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationBuyPackage, errs[0].Body.Operation)
	})

	t.Run("capacity is checked against the full member count before charging", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 2)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Empty(t, assets, "the compartment has room for only 2 of the 3 members -- none may be created")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit(), "capacity must be checked BEFORE the wallet is debited")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationBuyPackage, errs[0].Body.Operation)
		require.Equal(t, "INVENTORY_FULL", errs[0].Body.Error)
	})

	t.Run("insufficient funds charges nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 100)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Empty(t, assets)

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	})

	t.Run("an unknown package creates nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		// The package COMMODITY resolves (id pkgSerialNumber -> item
		// pkgItemId), but no CashPackage entry exists for pkgItemId.
		startPackageDataServer(t, defaultPackageCommodities(), map[uint32][]uint32{})
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Empty(t, assets)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationBuyPackage, errs[0].Body.Operation)
	})

	t.Run("gift mode delivers to the recipient", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId:     {accountId: buyerAccountId, jobId: 0, name: buyerName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: recipientName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		buyerCompartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		recipientCompartmentId := seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, recipientCharacterId, buyerName)
		require.NoError(t, err)

		recipientAssets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(recipientCompartmentId)
		require.NoError(t, err)
		require.Len(t, recipientAssets, 3, "the members must land in the RECIPIENT's compartment")

		buyerAssets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(buyerCompartmentId)
		require.NoError(t, err)
		require.Empty(t, buyerAssets, "no member may land in the buyer's own compartment on a gift")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(2000), w.Credit(), "the BUYER's wallet pays, even on a gift")

		// A second gift attempt, now with only 2000 remaining (< the 3000
		// package price), must reject on the GIFT_PACKAGE arm -- proving the
		// operation key tracks RecipientCharacterId rather than being fixed
		// to BUY_PACKAGE.
		events := captureDirectPurchaseEvents(t)
		secondTransactionId := uuid.New()
		err = NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, secondTransactionId, 1, pkgSerialNumber, recipientCharacterId, buyerName)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationGiftPackage, errs[0].Body.Operation)
		require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	})

	t.Run("replay delivers once", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		compartmentId := seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName))
		require.NoError(t, NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName))

		assets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(compartmentId)
		require.NoError(t, err)
		require.Len(t, assets, 3, "a replayed transaction id must not deliver a second set of members")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(2000), w.Credit(), "a replayed transaction id must not charge the buyer twice")
	})

	t.Run("records the package and every member", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			buyerCharacterId: {accountId: buyerAccountId, jobId: 0, name: buyerName},
		})
		startPackageDataServer(t, defaultPackageCommodities(), defaultPackageCatalog())
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).PurchasePackageAndEmit(buyerCharacterId, transactionId, 1, pkgSerialNumber, 0, buyerName))

		for _, sn := range []uint32{pkgSerialNumber, pkgMember0SN, pkgMember1SN, pkgMember2SN} {
			count, err := purchaserecord.Get(db, tenantId, buyerAccountId, sn)
			require.NoError(t, err)
			require.Equal(t, uint32(1), count, "purchaserecord must resolve for serial [%d]", sn)
		}
	})
}
