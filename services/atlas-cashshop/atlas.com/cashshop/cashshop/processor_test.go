package cashshop

// TestPurchaseTransactionId* prove the correlation carrier added in task-227
// task 37: TransactionId flows from RequestPurchaseCommandBody through
// PurchaseAndEmit to BOTH outcome events (PurchaseEventBody on success,
// ErrorEventBody on failure), and distinguishes two concurrent purchases for
// the SAME character. handleStatusEventPurchase on the channel side keys only
// off CharacterId today and cannot tell these apart without this id.
//
// Character and commodity lookups are REMOTE (HTTP) in production, so they
// are stubbed with httptest servers here, mirroring
// kafka/consumer/cashshop/consumer_test.go's startCharacterServer /
// startCommodityServer helpers (duplicated locally: that helper lives in a
// different package).

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/wallet"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// testPurchaseStatusTopic is what EVENT_TOPIC_CASH_SHOP_STATUS resolves to for
// the duration of the direct-producer-path tests -- mirrors
// coupon/processor_test.go's testStatusTopic / newProcessorTestEnv pattern.
const testPurchaseStatusTopic = "test-cash-shop-status-purchase"

// emittedPurchaseEvents is the package-wide producer capture, installed once
// from TestMain per the producertest.InstallCapturing contract -- the
// producer.Manager singleton must not be reset per test, only the Capture's
// recorded state (via Capture.Reset).
var emittedPurchaseEvents *producertest.Capture

func TestMain(m *testing.M) {
	emittedPurchaseEvents = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// captureDirectPurchaseEvents clears any messages recorded by a prior test
// and points EVENT_TOPIC_CASH_SHOP_STATUS at this test's own topic, so the
// rejectEmit / producer.ProviderImpl DIRECT path (INVENTORY_FULL,
// UNKNOWN_ERROR, NOT_ENOUGH_CASH) can be inspected without a live broker.
func captureDirectPurchaseEvents(t *testing.T) *producertest.Capture {
	t.Helper()
	t.Setenv("EVENT_TOPIC_CASH_SHOP_STATUS", testPurchaseStatusTopic)
	emittedPurchaseEvents.Reset()
	return emittedPurchaseEvents
}

func purchaseErrorEvents(t *testing.T, c *producertest.Capture) []cashshop.StatusEvent[cashshop.ErrorEventBody] {
	t.Helper()
	var out []cashshop.StatusEvent[cashshop.ErrorEventBody]
	for _, m := range c.Messages(testPurchaseStatusTopic) {
		var e cashshop.StatusEvent[cashshop.ErrorEventBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			continue
		}
		if e.Type == cashshop.StatusEventTypeError {
			out = append(out, e)
		}
	}
	return out
}

// testPurchaseItemId is a non-pet commodity item id (classification 200,
// per item.GetClassification's itemId/10000 floor division) so PurchaseAndEmit
// takes the plain-asset path rather than the pet-creation path, which needs
// its own (unrelated) remote data-service stub.
const testPurchaseItemId = uint32(2000000)

// purchaseCompartmentMigrationSqlite creates the cash_compartments table
// directly: compartment.Migration's AutoMigrate emits a PostgreSQL-only
// `DEFAULT uuid_generate_v4()` that sqlite's DDL parser rejects. Mirrors
// kafka/consumer/cashshop/consumer_test.go's compartmentMigrationSqlite.
func purchaseCompartmentMigrationSqlite(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS cash_compartments (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		account_id INTEGER NOT NULL,
		type INTEGER NOT NULL,
		capacity INTEGER NOT NULL DEFAULT 55
	)`).Error
}

func purchaseTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration, wallet.Migration, outbox.Migration)
}

func seedPurchaseCompartment(t *testing.T, db *gorm.DB, tenantId uuid.UUID, accountId uint32, capacity uint32) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Create(&compartment.Entity{Id: id, TenantId: tenantId, AccountId: accountId, Type: byte(compartment.TypeExplorer), Capacity: capacity}).Error)
	return id
}

func seedPurchaseWallet(t *testing.T, db *gorm.DB, tenantId uuid.UUID, accountId uint32, credit uint32) {
	t.Helper()
	require.NoError(t, db.Create(&wallet.Entity{Id: uuid.New(), TenantId: tenantId, AccountId: accountId, Credit: credit}).Error)
}

// seedPurchaseAsset occupies one compartment slot, mirroring
// kafka/consumer/cashshop/consumer_test.go's seedAsset.
func seedPurchaseAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, templateId uint32) {
	t.Helper()
	e := asset.Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        1,
		TemplateId:    templateId,
		Quantity:      1,
		PurchasedBy:   1,
		Expiration:    time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(&e).Error)
}

func startPurchaseCharacterServer(t *testing.T, characterId uint32, accountId uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"characters","id":"%d","attributes":{"accountId":%d,"jobId":0}}}`, characterId, accountId)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
}

func startPurchaseCommodityServer(t *testing.T, serialNumber uint32, itemId uint32, price uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"commodities","id":"%d","attributes":{"itemId":%d,"count":1,"price":%d,"period":30}}}`, serialNumber, itemId, price)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func purchaseOutboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvEventTopicStatus).Find(&rows).Error)
	return rows
}

func decodePurchaseEvent(t *testing.T, raw []byte) cashshop.StatusEvent[cashshop.PurchaseEventBody] {
	t.Helper()
	var ev cashshop.StatusEvent[cashshop.PurchaseEventBody]
	require.NoError(t, json.Unmarshal(raw, &ev))
	return ev
}

// TestPurchaseTransactionIdSurvivesToSuccessEvent proves the id threads
// command -> processor -> PurchaseEventBody on the success path.
func TestPurchaseTransactionIdSurvivesToSuccessEvent(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(9001)
	price := uint32(4000)
	transactionId := uuid.New()

	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, testPurchaseItemId, price)
	seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	seedPurchaseWallet(t, db, tenantId, accountId, price*2)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, serialNumber, transactionId)
	require.NoError(t, err)

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 1)
	ev := decodePurchaseEvent(t, entries[0].MessageValue)
	require.Equal(t, cashshop.StatusEventTypePurchase, ev.Type)
	require.Equal(t, transactionId, ev.Body.TransactionId, "success event must echo the command's transaction id")
}

// TestPurchaseTransactionIdSurvivesToErrorEvent proves the id threads through
// on a rejection path too -- a failure that drops the correlation is the same
// defect as never adding it (brief Step 3).
//
// This uses the INVENTORY_FULL rejection (zero-capacity compartment) rather
// than NOT_ENOUGH_CASH: Purchase()'s other error emits (including
// NOT_ENOUGH_CASH) go through mb.Put inside the ExecuteTransaction/Emit
// closure, and message.Emit only flushes the buffer to the outbox when the
// closure returns nil (kafka/message/message.go:49-52) -- so those emits are
// already unobservable through PurchaseAndEmit's outer call today, a
// pre-existing behavior this task does not touch. INVENTORY_FULL is captured
// via rejectEmit BEFORE the transaction returns, on the DIRECT producer path
// (kafka/producer.ProviderImpl), so it is the one error emit this harness can
// actually observe.
func TestPurchaseTransactionIdSurvivesToErrorEvent(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(9002)
	price := uint32(1)
	transactionId := uuid.New()

	events := captureDirectPurchaseEvents(t)
	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, testPurchaseItemId, price)
	// GORM's `default:` tag makes an explicit Capacity:0 omit the column from
	// the INSERT and fall through to the schema DEFAULT (55) instead -- so a
	// full compartment is seeded as capacity 1 with its one slot pre-occupied,
	// not capacity 0.
	compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 1)
	seedPurchaseAsset(t, db, tenantId, compartmentId, testPurchaseItemId)
	seedPurchaseWallet(t, db, tenantId, accountId, price*10)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, serialNumber, transactionId)
	require.NoError(t, err, "rejectEmit short-circuits with a nil return -- see Purchase()'s rejectEmit handling")

	errs := purchaseErrorEvents(t, events)
	require.Len(t, errs, 1)
	require.Equal(t, "INVENTORY_FULL", errs[0].Body.Error)
	require.Equal(t, transactionId, errs[0].Body.TransactionId, "error event must echo the command's transaction id")
}

// TestPurchaseTransactionIdDistinguishesConcurrentPurchases proves two
// purchases for the SAME character carrying different ids produce events an
// observer can tell apart -- the whole point of the carrier: today
// handleStatusEventPurchase keys only off CharacterId, so two BUYs in flight
// for one character are indistinguishable without this.
func TestPurchaseTransactionIdDistinguishesConcurrentPurchases(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	price := uint32(1000)
	txA := uuid.New()
	txB := uuid.New()

	startPurchaseCharacterServer(t, characterId, accountId)
	seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	seedPurchaseWallet(t, db, tenantId, accountId, price*4)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	startPurchaseCommodityServer(t, 9101, testPurchaseItemId, price)
	require.NoError(t, NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, 9101, txA))

	startPurchaseCommodityServer(t, 9102, testPurchaseItemId, price)
	require.NoError(t, NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, 9102, txB))

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 2)
	first := decodePurchaseEvent(t, entries[0].MessageValue)
	second := decodePurchaseEvent(t, entries[1].MessageValue)
	require.Equal(t, txA, first.Body.TransactionId)
	require.Equal(t, txB, second.Body.TransactionId)
	require.NotEqual(t, first.Body.TransactionId, second.Body.TransactionId, "two concurrent purchases for the same character must be distinguishable by transaction id")
}

// TestPurchaseInsufficientFundsReachesConsumer proves the NOT_ENOUGH_CASH
// error event actually reaches an observer. Before the Step-0 fix, this emit
// went through mb.Put inside the ExecuteTransaction/message.Emit closure
// (see TestPurchaseTransactionIdSurvivesToErrorEvent's comment), and
// message.Emit only flushes its buffer when the wrapped closure returns nil
// (kafka/message/message.go:49-52) -- but the balance check returns
// ErrInsufficientFunds, a non-nil error, so the buffered NOT_ENOUGH_CASH
// event was silently discarded and PurchaseAndEmit produced zero outbox rows
// and zero direct-producer events. The fix routes this emit through
// rejectEmit on the DIRECT producer path, mirroring INVENTORY_FULL, so it is
// now observable here exactly like the INVENTORY_FULL case above.
func TestPurchaseInsufficientFundsReachesConsumer(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(9004)
	price := uint32(4000)
	transactionId := uuid.New()

	events := captureDirectPurchaseEvents(t)
	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, testPurchaseItemId, price)
	seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	// Wallet balance is well under price, so the balance check fails before
	// any state-changing write.
	seedPurchaseWallet(t, db, tenantId, accountId, 1)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, serialNumber, transactionId)
	require.NoError(t, err, "rejectEmit short-circuits with a nil return, mirroring INVENTORY_FULL")

	// No outbox row: the rejection fires on the DIRECT producer path, not
	// through the tx's outbox writes.
	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 0)

	errs := purchaseErrorEvents(t, events)
	require.Len(t, errs, 1, "NOT_ENOUGH_CASH must reach the consumer instead of being silently dropped")
	require.Equal(t, "NOT_ENOUGH_CASH", errs[0].Body.Error)
	require.Equal(t, transactionId, errs[0].Body.TransactionId, "error event must echo the command's transaction id")
}

// TestPurchaseZeroTransactionIdAccepted proves the zero UUID -- what every
// existing caller sends today, since atlas-channel is not minting real ids
// until task 38/39 -- is accepted and passed through unchanged, so this
// change is backward compatible on the wire (brief Step 1).
func TestPurchaseZeroTransactionIdAccepted(t *testing.T) {
	db := purchaseTestDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	serialNumber := uint32(9003)
	price := uint32(4000)

	startPurchaseCharacterServer(t, characterId, accountId)
	startPurchaseCommodityServer(t, serialNumber, testPurchaseItemId, price)
	seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	seedPurchaseWallet(t, db, tenantId, accountId, price*2)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	err := NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 1, serialNumber, uuid.Nil)
	require.NoError(t, err)

	entries := purchaseOutboxEntries(t, db)
	require.Len(t, entries, 1)
	ev := decodePurchaseEvent(t, entries[0].MessageValue)
	require.Equal(t, uuid.Nil, ev.Body.TransactionId, "zero UUID means no correlation and must round-trip as zero")
}
