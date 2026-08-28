package surprise

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message/cashshop"
	itemmsg "atlas-cashshop/kafka/message/item"
	"atlas-cashshop/surprise/opening"
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

// The stock Cash Shop Surprise box template id -- GetSurpriseBoxTemplateIds
// falls back to this when a tenant has no configured list, which is the
// case in every test here (none seed configuration's tenant cache).
const testBoxTemplateId = uint32(5222000)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(cashshop.EnvEventTopicStatus), string(cashshop.EnvEventTopicStatus))
	_ = os.Setenv(string(itemmsg.EnvStatusTopic), string(itemmsg.EnvStatusTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}

// compartmentMigrationSqlite creates the cash_compartments table directly.
// compartment.Migration's AutoMigrate emits a `DEFAULT uuid_generate_v4()`
// column default, which is PostgreSQL-specific and fails sqlite's DDL
// parser -- see cashshop/inventory/compartment/resource_paginate_test.go for
// the established precedent. Tests always supply an explicit Id, so the
// default is never actually needed.
func compartmentMigrationSqlite(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS cash_compartments (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		account_id INTEGER NOT NULL,
		type INTEGER NOT NULL,
		capacity INTEGER NOT NULL DEFAULT 55
	)`).Error
}

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, compartmentMigrationSqlite, asset.Migration, opening.Migration, outbox.Migration)
}

func seedCompartment(t *testing.T, db *gorm.DB, tenantId uuid.UUID, accountId uint32, type_ compartment.CompartmentType, capacity uint32) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Create(&compartment.Entity{Id: id, TenantId: tenantId, AccountId: accountId, Type: byte(type_), Capacity: capacity}).Error)
	return id
}

func seedAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, cashId int64, templateId uint32, quantity uint32) uint32 {
	t.Helper()
	e := asset.Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        cashId,
		TemplateId:    templateId,
		Quantity:      quantity,
		PurchasedBy:   1,
		Expiration:    time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(&e).Error)
	return e.Id
}

func startCharacterServer(t *testing.T, characterId uint32, accountId uint32, jobId uint16) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"characters","id":"%d","attributes":{"accountId":%d,"jobId":%d}}}`, characterId, accountId, jobId)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
}

func startCommodityServer(t *testing.T, commodityId uint32, itemId uint32, count uint32, period uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"commodities","id":"%d","attributes":{"itemId":%d,"count":%d,"price":0,"period":%d}}}`, commodityId, itemId, count, period)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func startRewardPoolServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")
}

func rewardPoolOK(itemId uint32, quantity uint32, commodityId uint32) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"gachapon-rewards","id":"1","attributes":{"itemId":%d,"quantity":%d,"commodityId":%d,"gachaponId":"5222000"}}}`, itemId, quantity, commodityId)
	}
}

func rewardPoolStatus(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}
}

func setup(t *testing.T, capacity uint32) (db *gorm.DB, tenantId uuid.UUID, accountId uint32, characterId uint32, compartmentId uuid.UUID) {
	t.Helper()
	tenantId = uuid.New()
	accountId = 500
	characterId = 1000
	db = testDatabase(t)
	startCharacterServer(t, characterId, accountId, 0) // jobId 0 => Explorer
	compartmentId = seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, capacity)
	return
}

func rewardAssets(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID) []asset.Entity {
	t.Helper()
	var rows []asset.Entity
	require.NoError(t, db.Where("tenant_id = ? AND compartment_id = ?", tenantId, compartmentId).Find(&rows).Error)
	return rows
}

// outboxEntries returns only the surprise-status outbox rows (topic
// EVENT_TOPIC_CASH_SHOP_STATUS). asset.Create independently enqueues its own
// STATUS_TOPIC_CASH_ITEM row on every successful grant; that event is real
// and expected, but it is not what these tests are asserting about, so it is
// filtered out here rather than inflating every success-path entry count.
func outboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvEventTopicStatus).Find(&rows).Error)
	return rows
}

// rewardAssetRows returns the assets in compartmentId OTHER than the box
// itself -- i.e. the granted reward(s), if any.
func rewardAssetRows(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, boxId uint32) []asset.Entity {
	t.Helper()
	var rewards []asset.Entity
	for _, r := range rewardAssets(t, db, tenantId, compartmentId) {
		if r.Id != boxId {
			rewards = append(rewards, r)
		}
	}
	return rewards
}

func decodeOpened(t *testing.T, raw []byte) cashshop.StatusEvent[cashshop.SurpriseOpenedEventBody] {
	t.Helper()
	var ev cashshop.StatusEvent[cashshop.SurpriseOpenedEventBody]
	require.NoError(t, json.Unmarshal(raw, &ev))
	return ev
}

func openingCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Table("cash_surprise_openings").Count(&count).Error)
	return count
}

// TestOpenGrantsRewardAndDecrementsBox: box quantity 3 -> decremented to 2,
// reward created in the SAME compartment, SURPRISE_OPENED emitted carrying
// boxRemaining 2.
func TestOpenGrantsRewardAndDecrementsBox(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 777, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 777)
	require.NoError(t, err)

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 2, box.Quantity)

	rewardRows := rewardAssetRows(t, db, tenantId, compartmentId, boxId)
	require.Len(t, rewardRows, 1)
	require.EqualValues(t, 5000000, rewardRows[0].TemplateId)
	require.EqualValues(t, 40000, rewardRows[0].CommodityId)
	require.EqualValues(t, 1, rewardRows[0].Quantity)
	require.EqualValues(t, characterId, rewardRows[0].PurchasedBy)

	entries := outboxEntries(t, db)
	require.Len(t, entries, 1)
	require.Equal(t, string(cashshop.EnvEventTopicStatus), entries[0].Topic)
	ev := decodeOpened(t, entries[0].MessageValue)
	require.Equal(t, cashshop.StatusEventTypeSurpriseOpened, ev.Type)
	require.EqualValues(t, 2, ev.Body.BoxRemaining)
	require.EqualValues(t, 777, ev.Body.BoxCashId)
}

// TestOpenReleasesBoxAtZeroQuantity: quantity 1 -> the locker row is
// released, SURPRISE_OPENED carries boxRemaining 0 (the client removes the
// row on 0).
func TestOpenReleasesBoxAtZeroQuantity(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 778, testBoxTemplateId, 1)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 778)
	require.NoError(t, err)

	var box asset.Entity
	boxErr := db.First(&box, boxId).Error
	require.Error(t, boxErr, "the box row must be released (soft-deleted) at zero quantity")

	entries := outboxEntries(t, db)
	require.Len(t, entries, 1)
	ev := decodeOpened(t, entries[0].MessageValue)
	require.EqualValues(t, 0, ev.Body.BoxRemaining)
}

// TestOpenRewardCarriesCommodityDerivedFields: the reward carries
// commodityId, templateId, quantity and an expiration derived from the
// commodity's period -- all read back off the created asset.
func TestOpenRewardCarriesCommodityDerivedFields(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 779, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000001, 4, 40001))
	startCommodityServer(t, 40001, 5000001, 4, 7)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	before := time.Now()
	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 779)
	require.NoError(t, err)

	rows := rewardAssetRows(t, db, tenantId, compartmentId, boxId)
	require.Len(t, rows, 1)
	reward := &rows[0]
	require.EqualValues(t, 5000001, reward.TemplateId)
	require.EqualValues(t, 40001, reward.CommodityId)
	require.EqualValues(t, 4, reward.Quantity)

	wantExpiration := before.AddDate(0, 0, 7)
	require.WithinDuration(t, wantExpiration, reward.Expiration, 5*time.Second)
}

// TestOpenRejectsAssetOwnedByAnotherAccount (FR-2.1): an asset belonging to
// another account is not in this account's compartment, so the open is
// rejected with NO state change.
func TestOpenRejectsAssetOwnedByAnotherAccount(t *testing.T) {
	db, tenantId, _, _, ownerCompartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, ownerCompartmentId, 780, testBoxTemplateId, 3)

	// A different account, with its own (empty) compartment, attempts to
	// open the owner's box by cashId.
	forgerAccountId := uint32(999)
	forgerCharacterId := uint32(1999)
	seedCompartment(t, db, tenantId, forgerAccountId, compartment.TypeExplorer, 55)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), forgerAccountId, forgerCharacterId, 780)
	require.NoError(t, err, "a handled rejection is swallowed, not propagated")

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity, "the owner's box must be untouched")

	require.Empty(t, rewardAssetRows(t, db, tenantId, ownerCompartmentId, boxId), "no reward may be granted")
	require.Zero(t, openingCount(t, db), "no idempotency row may be written")
	require.Empty(t, outboxEntries(t, db), "no event may ride the outbox for a rejection")
}

// TestOpenRejectsNonSurpriseTemplate (FR-2.2): an asset that exists and is
// owned but whose templateId is not a configured Surprise box is rejected.
func TestOpenRejectsNonSurpriseTemplate(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 781, 5300000, 3)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 781)
	require.NoError(t, err)

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity)
	require.Empty(t, outboxEntries(t, db))
}

// TestOpenRejectsWhenLockerFullAndBoxIsStacked / TestOpenSucceedsWhenLockerFullAndBoxIsLast
// exercise both branches of HasRoomForSwap (FR-2.3) through the real path.
func TestOpenRejectsWhenLockerFullAndBoxIsStacked(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 2)
	seedAsset(t, db, tenantId, compartmentId, 900, 5100000, 1) // filler
	boxId := seedAsset(t, db, tenantId, compartmentId, 782, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 782)
	require.NoError(t, err)

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity, "a stacked box at a full locker must not be consumed")
	require.Empty(t, outboxEntries(t, db))
}

func TestOpenSucceedsWhenLockerFullAndBoxIsLast(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 2)
	seedAsset(t, db, tenantId, compartmentId, 901, 5100000, 1) // filler
	boxId := seedAsset(t, db, tenantId, compartmentId, 783, testBoxTemplateId, 1)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 783)
	require.NoError(t, err)

	var box asset.Entity
	require.Error(t, db.First(&box, boxId).Error, "the last box must be released, freeing the slot the reward takes")
	require.Len(t, outboxEntries(t, db), 1)
}

// TestOpenWithEmptyPoolLeavesBoxIntact (FR-6.4 / FR-4.1): an empty pool
// leaves the box untouched.
func TestOpenWithEmptyPoolLeavesBoxIntact(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 784, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolStatus(http.StatusConflict)) // 409 -> ErrPoolEmpty

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 784)
	require.NoError(t, err)

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity)
	require.Empty(t, rewardAssetRows(t, db, tenantId, compartmentId, boxId), "must contain only the untouched box")
	require.Zero(t, openingCount(t, db))
	require.Empty(t, outboxEntries(t, db))
}

// TestOpenIsAtomicWhenGrantFails (FR-4.1): forcing the reward creation to
// fail must roll the decrement (and the idempotency insert) back.
func TestOpenIsAtomicWhenGrantFails(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 785, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	// Force the reward asset INSERT to fail. UpdateQuantity (an UPDATE) is
	// unaffected, so the decrement succeeds before the forced failure --
	// exactly the ordering that must be rolled back atomically.
	databasetest.FailWritesOn(t, db, "cash_assets", databasetest.WriteCreate)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 785)
	require.Error(t, err, "a genuine write failure must propagate, unlike a handled rejection")

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity, "the decrement must have rolled back")

	require.Zero(t, openingCount(t, db), "the idempotency row must have rolled back too")
	require.Empty(t, outboxEntries(t, db))
}

// TestOpenIsIdempotentOnTransactionId (FR-4.4): the same transactionId
// twice grants exactly once and consumes exactly one box.
func TestOpenIsIdempotentOnTransactionId(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 786, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	txId := uuid.New()
	require.NoError(t, p.OpenAndEmit(txId, accountId, characterId, 786))
	require.NoError(t, p.OpenAndEmit(txId, accountId, characterId, 786), "a redelivery of the same transactionId must not error")

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 2, box.Quantity, "only ONE decrement across both calls")

	require.Len(t, rewardAssetRows(t, db, tenantId, compartmentId, boxId), 1, "only ONE reward across both calls")
	require.EqualValues(t, 1, openingCount(t, db))
	require.Len(t, outboxEntries(t, db), 1, "the replay must enqueue no additional event")
}

// TestOpenTwiceWithDistinctTransactionIdsGrantsTwice: a genuine second click
// (new transactionId) opens a second box.
func TestOpenTwiceWithDistinctTransactionIdsGrantsTwice(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 787, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 40000))
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	require.NoError(t, p.OpenAndEmit(uuid.New(), accountId, characterId, 787))
	require.NoError(t, p.OpenAndEmit(uuid.New(), accountId, characterId, 787))

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 1, box.Quantity, "TWO decrements across two distinct clicks")

	require.Len(t, rewardAssetRows(t, db, tenantId, compartmentId, boxId), 2)
	require.EqualValues(t, 2, openingCount(t, db))
	require.Len(t, outboxEntries(t, db), 2)
}

// TestOpenRejectsZeroCommodityId documents the deliberate task-207 decision:
// a reward pool entry with commodityId 0 has no price/period basis to
// derive an expiration from, so it is rejected (COMMODITY_MISSING) rather
// than granted with a fabricated default expiration.
func TestOpenRejectsZeroCommodityId(t *testing.T) {
	db, tenantId, accountId, characterId, compartmentId := setup(t, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, 788, testBoxTemplateId, 3)

	startRewardPoolServer(t, rewardPoolOK(5000000, 1, 0))

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.OpenAndEmit(uuid.New(), accountId, characterId, 788)
	require.NoError(t, err)

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity)
	require.Empty(t, rewardAssetRows(t, db, tenantId, compartmentId, boxId))
	require.Zero(t, openingCount(t, db))
}
