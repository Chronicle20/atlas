package cashshop

// TestPurchaseRing proves REQUEST_RING_PURCHASE (task-240 task 19): the
// buyer's wallet is charged once and TWO ring assets are created -- one in
// the buyer's own locker, one in the partner's -- recorded as a single pair
// via ring.CreatePair, atomically and idempotently. Every "creates neither"
// subtest is the point: a half-created pair (one asset without its partner,
// or a pair row without both assets) is exactly the failure the ring
// domain's placement exists to prevent.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/ring"
	"atlas-cashshop/wallet"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// ringCommodityId/ringItemId/ringPrice are the fixed commodity fixture every
// subtest resolves through startRingCommodityServer -- a friendship ring
// (design.md §4.3, OQ-R1's confirmed-correct same-template case).
const (
	ringCommodityId = uint32(60000)
	ringItemId      = uint32(1112800)
	ringPrice       = uint32(2500)
)

// ringTestDatabase mirrors giftTestDatabase (gift_test.go), adding the
// cash_rings table ring.CreatePair writes through.
func ringTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration, wallet.Migration, purchaserecord.Migration, ring.Migration, database.IdempotencyMigration, outbox.Migration)
}

type ringCharacterFixture struct {
	accountId uint32
	jobId     uint16
	name      string
}

// startRingCharacterServer mirrors startGiftCharacterServer (gift_test.go),
// dispatching on the trailing path segment so a buyer AND a partner on two
// different accounts can be resolved.
func startRingCharacterServer(t *testing.T, chars map[uint32]ringCharacterFixture) {
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

// startRingCommodityServer mirrors startGiftCommodityServer, answering
// exactly ONE serial number and 404ing everything else -- the "unknown
// commodity" subtest needs a serial that genuinely does not resolve.
func startRingCommodityServer(t *testing.T, serialNumber uint32, itemId uint32, price uint32) {
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

func ringOutboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvEventTopicStatus).Find(&rows).Error)
	return rows
}

func decodeRingEvents(t *testing.T, entries []outbox.Entity) []cashshop.StatusEvent[cashshop.RingPurchasedBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.RingPurchasedBody]
	for _, e := range entries {
		var ev cashshop.StatusEvent[cashshop.RingPurchasedBody]
		if err := json.Unmarshal(e.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == cashshop.StatusEventTypeRingPurchased {
			out = append(out, ev)
		}
	}
	return out
}

func TestPurchaseRing(t *testing.T) {
	const (
		buyerCharacterId   = uint32(42)
		buyerAccountId     = uint32(1)
		partnerCharacterId = uint32(77)
		partnerAccountId   = uint32(2)
		senderName         = "Buyer"
	)

	t.Run("creates two assets and one pair", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship))
		require.NoError(t, err)

		buyerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(buyerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, buyerCcm.Assets(), 1, "the buyer's own half must land in their own compartment")
		require.Equal(t, ringItemId, buyerCcm.Assets()[0].TemplateId())

		partnerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(partnerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, partnerCcm.Assets(), 1, "the partner's half must land in the partner's compartment")
		require.Equal(t, ringItemId, partnerCcm.Assets()[0].TemplateId())

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Len(t, buyerRings, 1)
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Len(t, partnerRings, 1)
		require.Equal(t, buyerRings[0].PairId(), partnerRings[0].PairId(), "both halves must share one pair id")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(2500), w.Credit(), "buyer wallet must be charged the commodity price exactly once")

		entries := ringOutboxEntries(t, db)
		evs := decodeRingEvents(t, entries)
		require.Len(t, evs, 1)
		require.Equal(t, transactionId, evs[0].Body.TransactionId)
		require.Equal(t, ringItemId, evs[0].Body.TemplateId)
		require.Equal(t, "Partner", evs[0].Body.PartnerName)
		require.Equal(t, string(ring.TypeFriendship), evs[0].Body.RingType)
	})

	t.Run("partner locker full creates neither", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		// The buyer's own compartment is deliberately NOT full -- this
		// proves the capacity check runs against BOTH compartments, not
		// just the buyer's.
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		partnerCompartmentId := seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 1)
		seedPurchaseAsset(t, db, tenantId, partnerCompartmentId, testPurchaseItemId)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship))
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		buyerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(buyerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, buyerCcm.Assets(), "no asset may be created for the buyer when the partner has no room")

		partnerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(partnerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, partnerCcm.Assets(), 1, "still only the pre-seeded occupant")

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Empty(t, buyerRings)
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Empty(t, partnerRings)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit(), "the buyer must not be charged on a rejected ring purchase")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationFriendship, errs[0].Body.Operation)
	})

	t.Run("insufficient funds creates neither", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 100)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship))
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Empty(t, buyerRings)
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Empty(t, partnerRings)

		buyerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(buyerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, buyerCcm.Assets())
		partnerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(partnerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, partnerCcm.Assets())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationFriendship, errs[0].Body.Operation)
		require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	})

	t.Run("unknown commodity creates neither", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		unknownSerial := uint32(99999)
		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, unknownSerial, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship))
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Empty(t, buyerRings)
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Empty(t, partnerRings)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationFriendship, errs[0].Body.Operation)
	})

	t.Run("couple type is recorded distinctly", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeCouple))
		require.NoError(t, err)

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Len(t, buyerRings, 1)
		require.Equal(t, ring.TypeCouple, buyerRings[0].Type())
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Len(t, partnerRings, 1)
		require.Equal(t, ring.TypeCouple, partnerRings[0].Type())

		// Prove the failure operation key for the COUPLE arm is "COUPLE" --
		// reuses the unknown-commodity rejection path with a couple ring
		// type, on a fresh transaction id.
		events := captureDirectPurchaseEvents(t)
		unknownSerial := uint32(99999)
		err = NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, uuid.New(), 1, unknownSerial, partnerCharacterId, senderName, "Forever", string(ring.TypeCouple))
		require.NoError(t, err)
		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationCouple, errs[0].Body.Operation)
	})

	// TestPurchaseRing/"a late failure rolls back the cross-account write" is
	// task 24a's item 1: PurchaseRingAndEmit's purchaserecord.Record (step 8)
	// is the LAST write before the success event, running after the buyer's
	// wallet was already debited AND both halves -- the buyer's own asset AND
	// the PARTNER's asset, on a different account -- were already created and
	// paired, in the SAME outer transaction. Failing it here proves
	// database.ExecuteTransaction rolls the whole transaction back together,
	// not just the write that failed. Precedent: ring/administrator_test.go's
	// TestCreatePairIsAtomic proves the inner ring.CreatePair batch is atomic;
	// this proves the OUTER cross-account transaction is too, which
	// TestCreatePairIsAtomic cannot reach.
	t.Run("a late failure rolls back the cross-account write", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		partnerCompartmentId := seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)
		databasetest.FailWritesOn(t, db, "cash_purchase_records", databasetest.WriteCreate)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship))
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit(), "the buyer's debit must roll back with the failed purchase record")

		buyerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(buyerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Empty(t, buyerCcm.Assets(), "the buyer's own asset must roll back")

		partnerAssets, err := asset.NewProcessor(l, ctx, db).GetByCompartmentId(partnerCompartmentId)
		require.NoError(t, err)
		require.Empty(t, partnerAssets, "the partner's asset -- created on a DIFFERENT account than the one that failed -- must roll back too")

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Empty(t, buyerRings, "the pair row must roll back with everything else")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationFriendship, errs[0].Body.Operation)
		require.Equal(t, "UNKNOWN_ERROR", errs[0].Body.Error)
	})

	t.Run("replay creates one pair", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship)))
		require.NoError(t, NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship)))

		buyerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(buyerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, buyerCcm.Assets(), 1, "a replayed transaction id must not mint a second buyer asset")
		partnerCcm, err := compartment.NewProcessor(l, ctx, db).GetByAccountIdAndType(partnerAccountId, compartment.TypeExplorer)
		require.NoError(t, err)
		require.Len(t, partnerCcm.Assets(), 1, "a replayed transaction id must not mint a second partner asset")

		buyerRings, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Len(t, buyerRings, 1, "a replayed transaction id must not create a second pair")
		partnerRings, err := ring.GetByCharacterId(db, tenantId, partnerCharacterId)
		require.NoError(t, err)
		require.Len(t, partnerRings, 1)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(buyerAccountId)
		require.NoError(t, err)
		require.Equal(t, uint32(2500), w.Credit(), "a replayed transaction id must not charge the buyer twice")
	})

	t.Run("the pair is queryable by character id", func(t *testing.T) {
		db := ringTestDatabase(t)
		tenantId := uuid.New()
		startRingCharacterServer(t, map[uint32]ringCharacterFixture{
			buyerCharacterId:   {accountId: buyerAccountId, jobId: 0, name: senderName},
			partnerCharacterId: {accountId: partnerAccountId, jobId: 0, name: "Partner"},
		})
		startRingCommodityServer(t, ringCommodityId, ringItemId, ringPrice)
		seedPurchaseCompartment(t, db, tenantId, buyerAccountId, 10)
		seedPurchaseCompartment(t, db, tenantId, partnerAccountId, 10)
		seedPurchaseWallet(t, db, tenantId, buyerAccountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).PurchaseRingAndEmit(buyerCharacterId, transactionId, 1, ringCommodityId, partnerCharacterId, senderName, "Forever", string(ring.TypeFriendship)))

		rows, err := ring.GetByCharacterId(db, tenantId, buyerCharacterId)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, partnerCharacterId, rows[0].PartnerCharacterId())
	})
}
