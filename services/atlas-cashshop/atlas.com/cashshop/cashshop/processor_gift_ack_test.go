package cashshop

// TestAcknowledgeGiftsAndEmit proves ACKNOWLEDGE_GIFTS (task-240 Defect H):
// draining the "gift list presented" flag on every named asset in the
// account's compartments, and leaving every other asset untouched.

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

func giftAckTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, purchaseCompartmentMigrationSqlite, asset.Migration)
}

func seedGiftAckAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, cashId int64) uint32 {
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

func TestAcknowledgeGiftsAndEmit(t *testing.T) {
	tenantId := uuid.New()
	accountId := uint32(42)
	db := giftAckTestDatabase(t)
	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	giftedId := seedGiftAckAsset(t, db, tenantId, compartmentId, 12345)
	otherId := seedGiftAckAsset(t, db, tenantId, compartmentId, 67890)

	err := NewProcessor(l, ctx, db).AcknowledgeGiftsAndEmit(accountId, []int64{12345})
	require.NoError(t, err)

	var gifted, other asset.Entity
	require.NoError(t, db.First(&gifted, giftedId).Error)
	require.NoError(t, db.First(&other, otherId).Error)
	require.True(t, gifted.GiftAcknowledged, "named cashId must be acknowledged")
	require.False(t, other.GiftAcknowledged, "unrelated cashId must be untouched")
}

// TestAcknowledgeGiftsAndEmitEmpty confirms an empty cashIds list is a no-op,
// not an unbounded update.
func TestAcknowledgeGiftsAndEmitEmpty(t *testing.T) {
	tenantId := uuid.New()
	accountId := uint32(42)
	db := giftAckTestDatabase(t)
	ctx := databasetest.TenantContext(tenantId)
	l, _ := testlog.NewNullLogger()

	compartmentId := seedPurchaseCompartment(t, db, tenantId, accountId, 55)
	assetId := seedGiftAckAsset(t, db, tenantId, compartmentId, 12345)

	err := NewProcessor(l, ctx, db).AcknowledgeGiftsAndEmit(accountId, nil)
	require.NoError(t, err)

	var a asset.Entity
	require.NoError(t, db.First(&a, assetId).Error)
	require.False(t, a.GiftAcknowledged)
}
