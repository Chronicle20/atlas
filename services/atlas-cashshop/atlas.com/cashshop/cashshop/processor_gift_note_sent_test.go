package cashshop

// TestMarkGiftNoteSentAndEmit proves MARK_GIFT_NOTE_SENT (task-240 Defect I):
// marking the named asset's gift-forward note as sent, and leaving every
// other asset untouched.

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func giftNoteSentTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration)
}

func seedGiftNoteSentAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, cashId int64) uint32 {
	t.Helper()
	e := asset.Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        cashId,
		TemplateId:    1032001,
		Quantity:      1,
		PurchasedBy:   1,
		Expiration:    time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:     time.Now(),
		GiftFrom:      "Sender1",
	}
	require.NoError(t, db.Create(&e).Error)
	return e.Id
}

func TestMarkGiftNoteSentAndEmit(t *testing.T) {
	tenantId := uuid.New()
	accountId := uint32(42)
	db := giftNoteSentTestDatabase(t)
	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	giftedId := seedGiftNoteSentAsset(t, db, tenantId, compartmentId, 12345)
	otherId := seedGiftNoteSentAsset(t, db, tenantId, compartmentId, 67890)

	err := NewProcessor(l, ctx, db).MarkGiftNoteSentAndEmit(accountId, 12345)
	require.NoError(t, err)

	var gifted, other asset.Entity
	require.NoError(t, db.First(&gifted, giftedId).Error)
	require.NoError(t, db.First(&other, otherId).Error)
	require.True(t, gifted.GiftNoteSent, "named cashId must have its note marked sent")
	require.False(t, other.GiftNoteSent, "unrelated cashId must be untouched")
}

// TestMarkGiftNoteSentAndEmitUnknownCashId confirms marking an unowned cashId
// updates nothing and returns no error -- there is nothing client-facing to
// answer either way.
func TestMarkGiftNoteSentAndEmitUnknownCashId(t *testing.T) {
	tenantId := uuid.New()
	accountId := uint32(42)
	db := giftNoteSentTestDatabase(t)
	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	assetId := seedGiftNoteSentAsset(t, db, tenantId, compartmentId, 12345)

	err := NewProcessor(l, ctx, db).MarkGiftNoteSentAndEmit(accountId, 99999)
	require.NoError(t, err)

	var a asset.Entity
	require.NoError(t, db.First(&a, assetId).Error)
	require.False(t, a.GiftNoteSent)
}
