package cashshop

// TestPurchaseEquipSlot proves REQUEST_EQUIP_SLOT_INCREASE (task-240 task
// 23): the buyer's wallet is charged once, and on success the character's
// equip-slot extension is extended via atlas-character's write route and an
// EQUIP_SLOT_INCREASED event is emitted -- exactly like every other purchase
// in this package, except there is no locker asset to create.

import (
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

const (
	equipSlotCommodityId = uint32(70000)
	equipSlotPrice       = uint32(4000)
	equipSlotPeriodDays  = uint32(30)
)

// startEquipSlotCharacterServer answers GET (character lookup) and POST
// (the ExtendEquipSlot write) on the same httptest server, mirroring
// atlas-character's equipslot resource: GET returns the character, POST
// acknowledges the extension with an arbitrary (unused by the caller)
// expiresAt.
func startEquipSlotCharacterServer(t *testing.T, characterId uint32, accountId uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if r.Method == http.MethodPost {
			_, _ = fmt.Fprintf(w, `{"data":{"type":"equip-slot-extensions","id":"1","attributes":{"characterId":%d,"slotIndex":-59,"expiresAt":"2030-01-01T00:00:00Z"}}}`, characterId)
			return
		}
		_, _ = fmt.Fprintf(w, `{"data":{"type":"characters","id":"%d","attributes":{"accountId":%d,"jobId":0}}}`, characterId, accountId)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
}

func startEquipSlotCommodityServer(t *testing.T, serialNumber uint32, price uint32, periodDays uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		id, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil || uint32(id) != serialNumber {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"commodities","id":"%d","attributes":{"itemId":0,"count":1,"price":%d,"period":%d}}}`, id, price, periodDays)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func equipSlotTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, wallet.Migration, purchaserecord.Migration, database.IdempotencyMigration, outbox.Migration)
}

func decodeEquipSlotEvents(t *testing.T, entries []outbox.Entity) []cashshop.StatusEvent[cashshop.EquipSlotIncreasedBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.EquipSlotIncreasedBody]
	for _, e := range entries {
		var ev cashshop.StatusEvent[cashshop.EquipSlotIncreasedBody]
		if err := json.Unmarshal(e.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == cashshop.StatusEventTypeEquipSlotIncreased {
			out = append(out, ev)
		}
	}
	return out
}

func TestPurchaseEquipSlot(t *testing.T) {
	const (
		characterId = uint32(42)
		accountId   = uint32(1)
	)

	t.Run("charges and emits the slot and duration", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		startEquipSlotCharacterServer(t, characterId, accountId)
		startEquipSlotCommodityServer(t, equipSlotCommodityId, equipSlotPrice, equipSlotPeriodDays)
		seedPurchaseWallet(t, db, tenantId, accountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, equipSlotCommodityId, transactionId)
		require.NoError(t, err)

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(1000), w.Credit(), "5000 - 4000 price")

		entries := purchaseOutboxEntries(t, db)
		evs := decodeEquipSlotEvents(t, entries)
		require.Len(t, evs, 1)
		require.Equal(t, transactionId, evs[0].Body.TransactionId)
		require.Equal(t, uint16(30), evs[0].Body.Days)
		require.Equal(t, int16(-59), evs[0].Body.SlotIndex, "must be the pendant2 canonical position, never the wire value")
	})

	t.Run("insufficient funds charges nothing", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startEquipSlotCharacterServer(t, characterId, accountId)
		startEquipSlotCommodityServer(t, equipSlotCommodityId, equipSlotPrice, equipSlotPeriodDays)
		seedPurchaseWallet(t, db, tenantId, accountId, 100)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, equipSlotCommodityId, transactionId)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(100), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationEnableEquipSlot, errs[0].Body.Operation)
		require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	})

	t.Run("unknown commodity charges nothing", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startEquipSlotCharacterServer(t, characterId, accountId)
		startEquipSlotCommodityServer(t, equipSlotCommodityId, equipSlotPrice, equipSlotPeriodDays)
		seedPurchaseWallet(t, db, tenantId, accountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		unknownSerial := uint32(99999)
		err := NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, unknownSerial, transactionId)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit())

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationEnableEquipSlot, errs[0].Body.Operation)
	})

	t.Run("replay charges once", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		startEquipSlotCharacterServer(t, characterId, accountId)
		startEquipSlotCommodityServer(t, equipSlotCommodityId, equipSlotPrice, equipSlotPeriodDays)
		seedPurchaseWallet(t, db, tenantId, accountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, equipSlotCommodityId, transactionId))
		require.NoError(t, NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, equipSlotCommodityId, transactionId))

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(1000), w.Credit(), "a replayed transaction id must not charge the buyer twice")
	})
}
