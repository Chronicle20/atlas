package listing

import (
	"context"
	"testing"
	"time"

	asset2 "atlas-merchant/kafka/message/asset"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupListingDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.New().String()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	l, _ := test.NewNullLogger()
	database.RegisterTenantCallbacks(l, db)
	require.NoError(t, Migration(db))
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return db, tenant.WithContext(context.Background(), ten)
}

func mkEntity(shopId uuid.UUID, order uint16) *Entity {
	return &Entity{
		Id:               uuid.New(),
		ShopId:           shopId,
		ItemId:           2000000,
		ItemType:         2,
		Quantity:         100,
		BundleSize:       100,
		BundlesRemaining: 1,
		PricePerBundle:   5000,
		ItemSnapshot:     asset2.AssetData{Quantity: 100},
		DisplayOrder:     order,
		Version:          1,
		ListedAt:         time.Now(),
	}
}

func TestCreateListingRoundTrip(t *testing.T) {
	db, ctx := setupListingDB(t)
	shopId := uuid.New()

	e := mkEntity(shopId, 0)
	created, err := createListing(e)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, e.Id, created.Id)

	got, err := getByShopIdAndDisplayOrder(shopId, 0)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, e.Id, got.Id)
	assert.Equal(t, uint32(2000000), got.ItemId)
	assert.Equal(t, uint16(1), got.BundlesRemaining)
	assert.Equal(t, uint32(1), got.Version)
}

// updateBundles is an optimistic-lock write: it only lands when the caller
// holds the current version, and bumps the version on success.
func TestUpdateBundles_VersionGate(t *testing.T) {
	db, ctx := setupListingDB(t)
	shopId := uuid.New()
	e := mkEntity(shopId, 0)
	_, err := createListing(e)(db.WithContext(ctx))()
	require.NoError(t, err)

	// Wrong expected version: no rows touched, row unchanged.
	rows, err := updateBundles(e.Id, 5, 500, 99)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)

	// Correct version: one row, version bumped.
	rows, err = updateBundles(e.Id, 5, 500, 1)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	got, err := getByShopIdAndDisplayOrder(shopId, 0)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, uint16(5), got.BundlesRemaining)
	assert.Equal(t, uint16(500), got.Quantity)
	assert.Equal(t, uint32(2), got.Version)

	// The stale version no longer matches after the bump.
	rows, err = updateBundles(e.Id, 1, 100, 1)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}

func TestDecrementDisplayOrderAfter(t *testing.T) {
	db, ctx := setupListingDB(t)
	shopId := uuid.New()
	for i := uint16(0); i < 4; i++ {
		_, err := createListing(mkEntity(shopId, i))(db.WithContext(ctx))()
		require.NoError(t, err)
	}

	// Remove index 1's successors' gap: rows 2 and 3 shift down; 0 stays.
	rows, err := decrementDisplayOrderAfter(shopId, 1)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(2), rows)

	all, err := getByShopId(shopId)(db.WithContext(ctx))()
	require.NoError(t, err)
	orders := make(map[uint16]int)
	for _, l := range all {
		orders[l.DisplayOrder]++
	}
	assert.Equal(t, map[uint16]int{0: 1, 1: 2, 2: 1}, orders)
}

func TestDeleteByShopId(t *testing.T) {
	db, ctx := setupListingDB(t)
	keep, drop := uuid.New(), uuid.New()
	_, err := createListing(mkEntity(keep, 0))(db.WithContext(ctx))()
	require.NoError(t, err)
	_, err = createListing(mkEntity(drop, 0))(db.WithContext(ctx))()
	require.NoError(t, err)
	_, err = createListing(mkEntity(drop, 1))(db.WithContext(ctx))()
	require.NoError(t, err)

	_, err = deleteByShopId(drop)(db.WithContext(ctx))()
	require.NoError(t, err)

	count, err := countByShopId(drop)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	count, err = countByShopId(keep)(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
