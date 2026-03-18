package frederick

import (
	"atlas-merchant/kafka/message/asset"
	"context"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := uuid.New().String()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)

	require.NoError(t, Migration(db))
	return db
}

func setupTestContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten), ten
}

func TestStoreAndGetItems(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	items := []StoredItem{
		{ItemId: 2000000, ItemType: 0, Quantity: 10, ItemSnapshot: asset.AssetData{}},
		{ItemId: 2000001, ItemType: 0, Quantity: 5, ItemSnapshot: asset.AssetData{}},
	}

	err := p.StoreItems(1000, items)
	require.NoError(t, err)

	result, err := p.GetItems(1000)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	itemIds := map[uint32]bool{}
	for _, item := range result {
		itemIds[item.ItemId()] = true
		assert.Equal(t, uint32(1000), item.CharacterId())
	}
	assert.True(t, itemIds[2000000])
	assert.True(t, itemIds[2000001])
}

func TestStoreAndGetMesos(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.StoreMesos(1000, 50000)
	require.NoError(t, err)

	result, err := p.GetMesos(1000)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(50000), result[0].Amount())
	assert.Equal(t, uint32(1000), result[0].CharacterId())
}

func TestStoreMesos_ZeroAmount(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.StoreMesos(1000, 0)
	require.NoError(t, err)

	result, err := p.GetMesos(1000)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestClearItems(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	items := []StoredItem{
		{ItemId: 2000000, ItemType: 0, Quantity: 10, ItemSnapshot: asset.AssetData{}},
	}
	require.NoError(t, p.StoreItems(1000, items))

	err := p.ClearItems(1000)
	require.NoError(t, err)

	result, err := p.GetItems(1000)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestClearMesos(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	require.NoError(t, p.StoreMesos(1000, 50000))

	err := p.ClearMesos(1000)
	require.NoError(t, err)

	result, err := p.GetMesos(1000)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestMultipleCharacterIsolation(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	items1 := []StoredItem{{ItemId: 2000000, ItemType: 0, Quantity: 10, ItemSnapshot: asset.AssetData{}}}
	items2 := []StoredItem{{ItemId: 2000001, ItemType: 0, Quantity: 5, ItemSnapshot: asset.AssetData{}}}

	require.NoError(t, p.StoreItems(1000, items1))
	require.NoError(t, p.StoreItems(2000, items2))
	require.NoError(t, p.StoreMesos(1000, 10000))
	require.NoError(t, p.StoreMesos(2000, 20000))

	result1, err := p.GetItems(1000)
	require.NoError(t, err)
	assert.Len(t, result1, 1)
	assert.Equal(t, uint32(2000000), result1[0].ItemId())

	result2, err := p.GetItems(2000)
	require.NoError(t, err)
	assert.Len(t, result2, 1)
	assert.Equal(t, uint32(2000001), result2[0].ItemId())

	mesos1, err := p.GetMesos(1000)
	require.NoError(t, err)
	assert.Len(t, mesos1, 1)
	assert.Equal(t, uint32(10000), mesos1[0].Amount())

	mesos2, err := p.GetMesos(2000)
	require.NoError(t, err)
	assert.Len(t, mesos2, 1)
	assert.Equal(t, uint32(20000), mesos2[0].Amount())

	// Clear character 1 — character 2 should be unaffected.
	require.NoError(t, p.ClearItems(1000))
	require.NoError(t, p.ClearMesos(1000))

	result1, err = p.GetItems(1000)
	require.NoError(t, err)
	assert.Empty(t, result1)

	result2, err = p.GetItems(2000)
	require.NoError(t, err)
	assert.Len(t, result2, 1)
}

func TestCreateNotification(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	err := p.CreateNotification(1000)
	require.NoError(t, err)

	// Verify notification was created by querying directly.
	var notifications []NotificationEntity
	err = db.WithContext(ctx).Where("character_id = ?", 1000).Find(&notifications).Error
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, uint16(2), notifications[0].NextDay)
}

func TestClearNotifications(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	require.NoError(t, p.CreateNotification(1000))

	err := p.ClearNotifications(1000)
	require.NoError(t, err)

	var notifications []NotificationEntity
	err = db.WithContext(ctx).Where("character_id = ?", 1000).Find(&notifications).Error
	require.NoError(t, err)
	assert.Empty(t, notifications)
}

func TestHasItemsOrMesos_None(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	has, err := HasItemsOrMesos(1000)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHasItemsOrMesos_Items(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	items := []StoredItem{{ItemId: 2000000, ItemType: 0, Quantity: 1, ItemSnapshot: asset.AssetData{}}}
	require.NoError(t, p.StoreItems(1000, items))

	has, err := HasItemsOrMesos(1000)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHasItemsOrMesos_Mesos(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	require.NoError(t, p.StoreMesos(1000, 5000))

	has, err := HasItemsOrMesos(1000)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.True(t, has)
}
