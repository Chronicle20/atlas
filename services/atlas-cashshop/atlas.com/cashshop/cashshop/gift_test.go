package cashshop

// TestGift proves REQUEST_GIFT_PURCHASE (task-240 task 13): the sender's
// wallet is charged and the commodity is delivered into the RECIPIENT's
// locker, atomically and idempotently via the same ledger.Claim-first
// ordering GiftAndEmit documents.

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
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
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// giftCommodityId/giftItemId/giftPrice are the fixed commodity fixture every
// subtest resolves through startGiftCommodityServer, mirroring
// processor_test.go's testPurchaseItemId.
const (
	giftCommodityId = uint32(10000)
	giftItemId      = uint32(5010000)
	giftPrice       = uint32(1200)
)

// giftTestDatabase mirrors rebateTestDatabase (rebate_test.go), adding the
// shared idempotency table ledger.Claim writes through.
func giftTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration, wallet.Migration, purchaserecord.Migration, database.IdempotencyMigration, outbox.Migration)
}

// giftCharacterFixture is one character server.NewProcessor(...).GetById can
// resolve, keyed by character id in the request path.
type giftCharacterFixture struct {
	accountId uint32
	jobId     uint16
	name      string
}

// startGiftCharacterServer stubs the remote character lookup for MULTIPLE
// characters, dispatching on the trailing path segment
// (character/requests.go's ById = "characters/%d") -- processor_test.go's
// startPurchaseCharacterServer answers every request identically, which
// cannot stand in for a sender AND a recipient on two different accounts.
func startGiftCharacterServer(t *testing.T, chars map[uint32]giftCharacterFixture) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		id, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		c, ok := chars[uint32(id)]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"characters","id":"%d","attributes":{"accountId":%d,"jobId":%d,"name":"%s"}}}`, id, c.accountId, c.jobId, c.name)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
}

// startGiftCommodityServer stubs the remote commodity lookup for exactly ONE
// serial number, 404ing everything else -- the "unknown commodity" subtest
// needs a serial that genuinely does not resolve, unlike
// processor_test.go's startPurchaseCommodityServer which answers every
// request identically regardless of the requested serial.
func startGiftCommodityServer(t *testing.T, serialNumber uint32, itemId uint32, price uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		id, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil || uint32(id) != serialNumber {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"commodities","id":"%d","attributes":{"itemId":%d,"count":1,"price":%d,"period":30}}}`, id, itemId, price)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func giftOutboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvEventTopicStatus).Find(&rows).Error)
	return rows
}

func decodeGiftEvents(t *testing.T, entries []outbox.Entity) []cashshop.StatusEvent[cashshop.GiftPurchasedBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.GiftPurchasedBody]
	for _, e := range entries {
		var ev cashshop.StatusEvent[cashshop.GiftPurchasedBody]
		if err := json.Unmarshal(e.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == cashshop.StatusEventTypeGiftPurchased {
			out = append(out, ev)
		}
	}
	return out
}

func TestGift(t *testing.T) {
	const (
		senderCharacterId    = uint32(42)
		senderAccountId      = uint32(1)
		recipientCharacterId = uint32(77)
		recipientAccountId   = uint32(2)
		senderName           = "Sender"
		giftText             = "Enjoy!"
	)

	t.Run("delivers to the recipient locker", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		recipientCompartmentId := seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 55)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText)
		require.NoError(t, err)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(senderAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(3800), w.Credit(), "sender wallet must be charged the commodity price")

		recipientCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(recipientAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, recipientCcm.Assets(), 1, "the gifted item must land in the recipient's compartment")
		gifted := recipientCcm.Assets()[0]
		require.Equal(t, giftItemId, gifted.TemplateId())
		require.Equal(t, senderName, gifted.GiftFrom())
		require.Equal(t, giftText, gifted.GiftMessage())

		recipientAssets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(recipientCompartmentId)
		require.NoError(t, err)
		require.Len(t, recipientAssets, 1, "control: this reads the RECIPIENT compartment directly, confirming the row lives there")

		senderSeededCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(senderAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, senderSeededCcm.Assets(), "no new row may land in the sender's own compartment")

		entries := giftOutboxEntries(t, db)
		evs := decodeGiftEvents(t, entries)
		require.Len(t, evs, 1)
		require.Equal(t, transactionId, evs[0].Body.TransactionId)
		require.Equal(t, "Recipient", evs[0].Body.RecipientName)
		require.Equal(t, giftItemId, evs[0].Body.TemplateId)
		require.Equal(t, giftPrice, evs[0].Body.Price)
		require.Equal(t, recipientCharacterId, evs[0].Body.RecipientCharacterId)
	})

	t.Run("insufficient funds charges nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 55)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 100)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		recipientCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(recipientAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, recipientCcm.Assets(), "no asset may be created on a rejected gift")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(senderAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(100), w.Credit(), "the sender must not be charged on a rejected gift")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationGift, errs[0].Body.Operation)
		require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	})

	t.Run("recipient locker full charges nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		// The sender's own compartment is deliberately NOT full -- this is
		// the fixture that proves the capacity check runs against the
		// RECIPIENT's compartment: if both were full, the test would pass
		// even if the code (wrongly) checked the sender's.
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		recipientCompartmentId := seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 1)
		seedPurchaseAsset(t, db, tenantId, recipientCompartmentId, testPurchaseItemId)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		recipientCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(recipientAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, recipientCcm.Assets(), 1, "still only the pre-seeded occupant, no gift created")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(senderAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit(), "the sender must not be charged on a rejected gift")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationGift, errs[0].Body.Operation)
		require.Equal(t, "CANNOT_GIFT_RECIPIENT_INVENTORY_FULL", errs[0].Body.Error)
	})

	t.Run("unknown commodity charges nothing", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 55)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		unknownSerial := uint32(99999)
		err := NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, unknownSerial, recipientCharacterId, senderName, giftText)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		recipientCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(recipientAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, recipientCcm.Assets())

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(senderAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationGift, errs[0].Body.Operation)
	})

	t.Run("replay delivers once", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 55)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText))
		require.NoError(t, NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText))

		recipientCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(recipientAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, recipientCcm.Assets(), 1, "a replayed transaction id must not deliver a second asset")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(senderAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(3800), w.Credit(), "a replayed transaction id must not charge the sender twice")
	})

	t.Run("records the purchase for the sender", func(t *testing.T) {
		db := giftTestDatabase(t)
		tenantId := uuid.New()
		startGiftCharacterServer(t, map[uint32]giftCharacterFixture{
			senderCharacterId:    {accountId: senderAccountId, jobId: 0, name: senderName},
			recipientCharacterId: {accountId: recipientAccountId, jobId: 0, name: "Recipient"},
		})
		startGiftCommodityServer(t, giftCommodityId, giftItemId, giftPrice)
		seedPurchaseCompartment(t, db, tenantId, senderAccountId, 55)
		seedPurchaseCompartment(t, db, tenantId, recipientAccountId, 55)
		seedPurchaseWallet(t, db, tenantId, senderAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).GiftAndEmit(senderCharacterId, transactionId, giftCommodityId, recipientCharacterId, senderName, giftText))

		senderCount, err := purchaserecord.Get(db, tenantId, senderAccountId, giftCommodityId)
		require.NoError(t, err)
		require.Equal(t, uint32(1), senderCount, "the SENDER bought it")

		recipientCount, err := purchaserecord.Get(db, tenantId, recipientAccountId, giftCommodityId)
		require.NoError(t, err)
		require.Equal(t, uint32(0), recipientCount, "the recipient never bought anything -- it must not be recorded against their account")
	})
}
