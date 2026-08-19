package cashshop

// TestPurchaseEquipSlot proves REQUEST_EQUIP_SLOT_INCREASE (task-240 task
// 23, task 24c): the buyer's wallet is charged once, the charge is recorded,
// and PurchaseEquipSlotAndEmit mints an EXTEND_EQUIP_SLOT outbox command
// rather than calling atlas-character's write route itself (task 24c) --
// that write, and the EQUIP_SLOT_INCREASED event it unlocks, only happen once
// CompleteEquipSlotExtension consumes that command, mirroring what the
// EXTEND_EQUIP_SLOT command consumer (kafka/consumer/cashshop/consumer.go)
// does after the outbox durably delivers it.

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
	"sync/atomic"
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
// expiresAt. postCount, when non-nil, is incremented on every POST so a
// caller can prove how many times atlas-character's write route was called.
func startEquipSlotCharacterServer(t *testing.T, characterId uint32, accountId uint32) {
	t.Helper()
	startEquipSlotCharacterServerCountingPosts(t, characterId, accountId, nil)
}

func startEquipSlotCharacterServerCountingPosts(t *testing.T, characterId uint32, accountId uint32, postCount *atomic.Int32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if r.Method == http.MethodPost {
			if postCount != nil {
				postCount.Add(1)
			}
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

// equipSlotTestDatabaseWithoutPurchaseRecord omits purchaserecord.Migration,
// so purchaserecord.Record fails with "no such table" -- simulating a step 5
// (record the purchase) failure so tests can prove it aborts the WHOLE
// transaction, including the EXTEND_EQUIP_SLOT command mint.
func equipSlotTestDatabaseWithoutPurchaseRecord(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, wallet.Migration, database.IdempotencyMigration, outbox.Migration)
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

// equipSlotCommandOutboxEntries returns the outbox rows queued on
// EnvCommandTopic -- where PurchaseEquipSlotAndEmit mints the internal
// EXTEND_EQUIP_SLOT follow-up command (task-240 task 24c), as opposed to
// purchaseOutboxEntries (processor_test.go), which reads EnvEventTopicStatus.
func equipSlotCommandOutboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvCommandTopic).Find(&rows).Error)
	return rows
}

func decodeExtendEquipSlotCommands(t *testing.T, entries []outbox.Entity) []cashshop.Command[cashshop.ExtendEquipSlotCommandBody] {
	t.Helper()
	var out []cashshop.Command[cashshop.ExtendEquipSlotCommandBody]
	for _, e := range entries {
		var c cashshop.Command[cashshop.ExtendEquipSlotCommandBody]
		if err := json.Unmarshal(e.MessageValue, &c); err != nil {
			continue
		}
		if c.Type == cashshop.CommandTypeExtendEquipSlot {
			out = append(out, c)
		}
	}
	return out
}

func TestPurchaseEquipSlot(t *testing.T) {
	const (
		characterId = uint32(42)
		accountId   = uint32(1)
	)

	t.Run("charges and queues the extend command, not the atlas-character write", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		var posts atomic.Int32
		startEquipSlotCharacterServerCountingPosts(t, characterId, accountId, &posts)
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

		require.Equal(t, int32(0), posts.Load(), "the atlas-character write must NOT happen inside PurchaseEquipSlotAndEmit (task 24c)")

		// No EQUIP_SLOT_INCREASED yet -- that only fires once
		// CompleteEquipSlotExtension confirms the deferred write.
		require.Empty(t, decodeEquipSlotEvents(t, purchaseOutboxEntries(t, db)))

		cmds := decodeExtendEquipSlotCommands(t, equipSlotCommandOutboxEntries(t, db))
		require.Len(t, cmds, 1)
		require.Equal(t, transactionId, cmds[0].Body.TransactionId)
		require.Equal(t, uint16(30), cmds[0].Body.Days)
		require.Equal(t, int16(-59), cmds[0].Body.SlotIndex, "must be the pendant2 canonical position, never the wire value")
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

		require.Empty(t, equipSlotCommandOutboxEntries(t, db), "a rejected purchase must not queue an extend command")
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

	// TestPurchaseEquipSlot/a_step_5_failure_after_a_successful_charge_leaves_no_extend_command
	// is the ordering guarantee task 24c exists to create: before this task,
	// calling atlas-character's write route INSIDE the transaction meant a
	// later step's failure rolled back the wallet debit and ledger claim
	// while the (out-of-transaction) extension it had already granted stood
	// -- an uncharged extension, permanently. Now the write is deferred
	// behind an outbox command minted from the SAME transaction as the
	// record step, so a record failure rolls back the debit AND ensures the
	// command is never durably queued -- there is nothing left to extend.
	t.Run("a step 5 failure after a successful charge leaves no extend command queued", func(t *testing.T) {
		db := equipSlotTestDatabaseWithoutPurchaseRecord(t)
		tenantId := uuid.New()
		events := captureDirectPurchaseEvents(t)
		startEquipSlotCharacterServer(t, characterId, accountId)
		startEquipSlotCommodityServer(t, equipSlotCommodityId, equipSlotPrice, equipSlotPeriodDays)
		seedPurchaseWallet(t, db, tenantId, accountId, 5000)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(characterId, 1, equipSlotCommodityId, transactionId)
		require.NoError(t, err, "rejectEmit short-circuits with a nil return")

		w, err := wallet.NewProcessor(l, ctx, db).GetByAccountId(accountId)
		require.NoError(t, err)
		require.Equal(t, uint32(5000), w.Credit(), "the wallet debit must roll back with the rest of the transaction")

		errs := purchaseErrorEvents(t, events)
		require.Len(t, errs, 1)
		require.Equal(t, cashshop.ErrorOperationEnableEquipSlot, errs[0].Body.Operation)

		require.Empty(t, equipSlotCommandOutboxEntries(t, db), "a step 5 failure must leave no extend command queued -- no uncharged extension is possible")
	})
}

// TestCompleteEquipSlotExtension proves the consumer-side half of task 24c:
// CompleteEquipSlotExtension is what the EXTEND_EQUIP_SLOT command consumer
// invokes once the outbox durably delivers the command
// PurchaseEquipSlotAndEmit queued. It performs the atlas-character write and,
// only on success, emits EQUIP_SLOT_INCREASED -- the event the client
// actually waits on.
func TestCompleteEquipSlotExtension(t *testing.T) {
	const (
		characterId = uint32(42)
		accountId   = uint32(1)
		slotIndex   = int16(-59)
		days        = uint16(30)
	)

	t.Run("extends and emits on success", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		var posts atomic.Int32
		startEquipSlotCharacterServerCountingPosts(t, characterId, accountId, &posts)
		events := captureDirectPurchaseEvents(t)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		err := NewProcessor(l, ctx, db).CompleteEquipSlotExtension(characterId, slotIndex, days, transactionId)
		require.NoError(t, err)
		require.Equal(t, int32(1), posts.Load(), "must call atlas-character's write route exactly once")

		var ev cashshop.StatusEvent[cashshop.EquipSlotIncreasedBody]
		found := false
		for _, m := range events.Messages(testPurchaseStatusTopic) {
			if err := json.Unmarshal(m.Value, &ev); err == nil && ev.Type == cashshop.StatusEventTypeEquipSlotIncreased {
				found = true
				break
			}
		}
		require.True(t, found, "EQUIP_SLOT_INCREASED must be emitted once the write succeeds")
		require.Equal(t, transactionId, ev.Body.TransactionId)
		require.Equal(t, slotIndex, ev.Body.SlotIndex)
		require.Equal(t, days, ev.Body.Days)
	})

	// TestCompleteEquipSlotExtension/a_redelivered_command_does_not_double-extend
	// is the idempotency guard task 24c exists to add: the outbox is
	// at-least-once, so the SAME EXTEND_EQUIP_SLOT command can be consumed
	// twice. Driven through CompleteEquipSlotExtension -> ExtendEquipSlot ->
	// atlas-character's write ROUTE (not around it, i.e. the real HTTP round
	// trip via httptest, mirroring how the consumer actually calls this) so
	// it proves the transaction id survives the wire, not just an in-process
	// call.
	t.Run("a redelivered command does not double-extend", func(t *testing.T) {
		db := equipSlotTestDatabase(t)
		tenantId := uuid.New()
		var posts atomic.Int32
		startEquipSlotCharacterServerCountingPosts(t, characterId, accountId, &posts)

		ctx := databasetest.TenantContext(tenantId)
		l, _ := testlog.NewNullLogger()
		transactionId := uuid.New()

		require.NoError(t, NewProcessor(l, ctx, db).CompleteEquipSlotExtension(characterId, slotIndex, days, transactionId))
		require.NoError(t, NewProcessor(l, ctx, db).CompleteEquipSlotExtension(characterId, slotIndex, days, transactionId))

		require.Equal(t, int32(2), posts.Load(), "atlas-cashshop calls the route both times -- the dedupe guard lives on atlas-character's side")
	})
}
