package cashshop

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/message/item"
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
// case in every test here (none seed configuration's tenant cache). Mirrors
// surprise/processor_test.go's testBoxTemplateId.
const testBoxTemplateId = uint32(5222000)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(cashshop.EnvEventTopicStatus), string(cashshop.EnvEventTopicStatus))
	_ = os.Setenv(string(item.EnvStatusTopic), string(item.EnvStatusTopic))
	producertest.InstallNoop()
	os.Exit(m.Run())
}

// compartmentMigrationSqlite creates the cash_compartments table directly,
// mirroring surprise/processor_test.go -- compartment.Migration's
// AutoMigrate emits a PostgreSQL-only `DEFAULT uuid_generate_v4()` that
// sqlite's DDL parser rejects.
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

func startRewardPoolServer(t *testing.T, itemId uint32, quantity uint32, commodityId uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"gachapon-rewards","id":"1","attributes":{"itemId":%d,"quantity":%d,"commodityId":%d,"gachaponId":"5222000"}}}`, itemId, quantity, commodityId)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")
}

func outboxEntries(t *testing.T, db *gorm.DB) []outbox.Entity {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", cashshop.EnvEventTopicStatus).Find(&rows).Error)
	return rows
}

func decodeOpened(t *testing.T, raw []byte) cashshop.StatusEvent[cashshop.SurpriseOpenedEventBody] {
	t.Helper()
	var ev cashshop.StatusEvent[cashshop.SurpriseOpenedEventBody]
	require.NoError(t, json.Unmarshal(raw, &ev))
	return ev
}

// TestHandleOpenSurpriseInvokesProcessor dispatches an OPEN_SURPRISE command
// with a known transactionId/accountId/characterId/cashId and asserts the
// call reached surprise.Processor.OpenAndEmit with exactly those values --
// via the box decrement and the SURPRISE_OPENED event it produces, which
// only OpenAndEmit(transactionId, accountId, characterId, cashId) can emit.
func TestHandleOpenSurpriseInvokesProcessor(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	cashId := int64(777)
	transactionId := uuid.New()

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, cashId, testBoxTemplateId, 3)

	startRewardPoolServer(t, 5000000, 1, 40000)
	startCommodityServer(t, 40000, 5000000, 1, 30)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleOpenSurprise(db)(l, ctx, cashshop.Command[cashshop.OpenSurpriseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeOpenSurprise,
		Body: cashshop.OpenSurpriseCommandBody{
			TransactionId: transactionId,
			AccountId:     accountId,
			CashId:        cashId,
		},
	})

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 2, box.Quantity, "box decremented by the invoked OpenAndEmit call")

	entries := outboxEntries(t, db)
	require.Len(t, entries, 1)
	ev := decodeOpened(t, entries[0].MessageValue)
	require.Equal(t, cashshop.StatusEventTypeSurpriseOpened, ev.Type)
	require.EqualValues(t, cashId, ev.Body.BoxCashId, "cashId came from the command body")
}

// TestHandleOpenSurpriseIgnoresOtherCommandTypes proves the type guard: the
// command topic is shared, so a REQUEST_PURCHASE-shaped message dispatched
// into handleOpenSurprise must be ignored, not misread as an
// OpenSurpriseCommandBody with a zero transactionId/cashId.
func TestHandleOpenSurpriseIgnoresOtherCommandTypes(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	cashId := int64(778)

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	boxId := seedAsset(t, db, tenantId, compartmentId, cashId, testBoxTemplateId, 3)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleOpenSurprise(db)(l, ctx, cashshop.Command[cashshop.OpenSurpriseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPurchase,
		Body: cashshop.OpenSurpriseCommandBody{
			TransactionId: uuid.New(),
			AccountId:     accountId,
			CashId:        cashId,
		},
	})

	var box asset.Entity
	require.NoError(t, db.First(&box, boxId).Error)
	require.EqualValues(t, 3, box.Quantity, "wrong-type command must not touch the box")

	require.Empty(t, outboxEntries(t, db), "wrong-type command must not invoke the processor")
}

// TestHandleAcknowledgeGiftsInvokesProcessor dispatches an ACKNOWLEDGE_GIFTS
// command and asserts the named asset's GiftAcknowledged flag flips to true
// (task-240 Defect H) while an unrelated asset in the same compartment is
// left untouched.
func TestHandleAcknowledgeGiftsInvokesProcessor(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	giftedCashId := int64(9001)
	otherCashId := int64(9002)

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	giftedId := seedAsset(t, db, tenantId, compartmentId, giftedCashId, 1032001, 1)
	otherId := seedAsset(t, db, tenantId, compartmentId, otherCashId, 1032002, 1)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleCommandAcknowledgeGifts(db)(l, ctx, cashshop.Command[cashshop.AcknowledgeGiftsCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeAcknowledgeGifts,
		Body: cashshop.AcknowledgeGiftsCommandBody{
			AccountId: accountId,
			CashIds:   []int64{giftedCashId},
		},
	})

	var gifted, other asset.Entity
	require.NoError(t, db.First(&gifted, giftedId).Error)
	require.NoError(t, db.First(&other, otherId).Error)
	require.True(t, gifted.GiftAcknowledged, "named cashId must be acknowledged")
	require.False(t, other.GiftAcknowledged, "unrelated cashId must be untouched")
}

// TestHandleAcknowledgeGiftsIgnoresOtherCommandTypes proves the type guard.
func TestHandleAcknowledgeGiftsIgnoresOtherCommandTypes(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	cashId := int64(9003)

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	assetId := seedAsset(t, db, tenantId, compartmentId, cashId, 1032001, 1)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleCommandAcknowledgeGifts(db)(l, ctx, cashshop.Command[cashshop.AcknowledgeGiftsCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPurchase,
		Body: cashshop.AcknowledgeGiftsCommandBody{
			AccountId: accountId,
			CashIds:   []int64{cashId},
		},
	})

	var a asset.Entity
	require.NoError(t, db.First(&a, assetId).Error)
	require.False(t, a.GiftAcknowledged, "wrong-type command must not touch the asset")
}

// TestHandleMarkGiftNoteSentInvokesProcessor dispatches a MARK_GIFT_NOTE_SENT
// command and asserts the named asset's GiftNoteSent flag flips to true
// (task-240 Defect I) while an unrelated asset in the same compartment is
// left untouched.
func TestHandleMarkGiftNoteSentInvokesProcessor(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	giftedCashId := int64(9101)
	otherCashId := int64(9102)

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	giftedId := seedAsset(t, db, tenantId, compartmentId, giftedCashId, 1032001, 1)
	otherId := seedAsset(t, db, tenantId, compartmentId, otherCashId, 1032002, 1)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleCommandMarkGiftNoteSent(db)(l, ctx, cashshop.Command[cashshop.MarkGiftNoteSentCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeMarkGiftNoteSent,
		Body: cashshop.MarkGiftNoteSentCommandBody{
			AccountId: accountId,
			CashId:    giftedCashId,
		},
	})

	var gifted, other asset.Entity
	require.NoError(t, db.First(&gifted, giftedId).Error)
	require.NoError(t, db.First(&other, otherId).Error)
	require.True(t, gifted.GiftNoteSent, "named cashId must have its note marked sent")
	require.False(t, other.GiftNoteSent, "unrelated cashId must be untouched")
}

// TestHandleMarkGiftNoteSentIgnoresOtherCommandTypes proves the type guard.
func TestHandleMarkGiftNoteSentIgnoresOtherCommandTypes(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	accountId := uint32(500)
	characterId := uint32(1000)
	cashId := int64(9103)

	startCharacterServer(t, characterId, accountId, 0)
	compartmentId := seedCompartment(t, db, tenantId, accountId, compartment.TypeExplorer, 55)
	assetId := seedAsset(t, db, tenantId, compartmentId, cashId, 1032001, 1)

	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	handleCommandMarkGiftNoteSent(db)(l, ctx, cashshop.Command[cashshop.MarkGiftNoteSentCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPurchase,
		Body: cashshop.MarkGiftNoteSentCommandBody{
			AccountId: accountId,
			CashId:    cashId,
		},
	})

	var a asset.Entity
	require.NoError(t, db.First(&a, assetId).Error)
	require.False(t, a.GiftNoteSent, "wrong-type command must not touch the asset")
}
