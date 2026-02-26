package shop

import (
	"atlas-merchant/frederick"
	"atlas-merchant/listing"
	"context"
	"encoding/json"
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
	require.NoError(t, listing.Migration(db))
	require.NoError(t, frederick.Migration(db))
	return db
}

func setupTestContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten), ten
}

func TestCreateShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, CharacterShop, m.ShopType())
	assert.Equal(t, Draft, m.State())
	assert.Equal(t, "Test Shop", m.Title())
	assert.Equal(t, uint32(910000001), m.MapId())
}

func TestCreateShop_NotFreemarketRoom(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	_, err := p.CreateShop(1000, CharacterShop, "Test Shop", 100000100, 0, 0, 0)
	assert.ErrorIs(t, err, ErrNotFreemarketRoom)
}

func TestCreateShop_DuplicateActiveShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	_, err := p.CreateShop(1000, CharacterShop, "Shop 1", 910000001, 0, 0, 0)
	require.NoError(t, err)

	_, err = p.CreateShop(1000, CharacterShop, "Shop 2", 910000002, 0, 0, 0)
	assert.ErrorIs(t, err, ErrShopLimitReached)
}

func TestOpenShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	opened, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Open, opened.State())
}

func TestOpenShop_NoListings(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	assert.ErrorIs(t, err, ErrNoListings)
}

func TestCloseShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	err = p.CloseShop(m.Id(), CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
	assert.Equal(t, CloseReasonManualClose, closed.CloseReason())
}

func TestCloseShop_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	err = p.CloseShop(m.Id(), CloseReasonManualClose)
	require.NoError(t, err)

	// Closing an already-closed shop should fail.
	err = p.CloseShop(m.Id(), CloseReasonManualClose)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestAddListing(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	li, err := p.AddListing(m.Id(), 2000000, 0, 5, 10, 1000, snapshot, 0)
	require.NoError(t, err)
	assert.Equal(t, uint32(2000000), li.ItemId())
	assert.Equal(t, uint16(5), li.BundleSize())
	assert.Equal(t, uint16(10), li.BundlesRemaining())
	assert.Equal(t, uint32(1000), li.PricePerBundle())
}

func TestAddListing_LimitReached(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	for i := 0; i < MaxListings; i++ {
		_, err = p.AddListing(m.Id(), 2000000, 0, 1, 1, 1000, snapshot, 0)
		require.NoError(t, err)
	}

	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 1, 1000, snapshot, 0)
	assert.ErrorIs(t, err, ErrListingLimitReached)
}

func TestPurchaseBundle(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 5, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	result, err := p.PurchaseBundle(2000, m.Id(), 0, 3)
	require.NoError(t, err)
	assert.Equal(t, uint16(3), result.BundlesPurchased)
	assert.Equal(t, uint16(7), result.BundlesRemaining)
	assert.Equal(t, int64(3000), result.TotalCost)
	assert.False(t, result.ShopClosed)
}

func TestPurchaseBundle_SoldOut(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 5, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	result, err := p.PurchaseBundle(2000, m.Id(), 0, 5)
	require.NoError(t, err)
	assert.Equal(t, uint16(0), result.BundlesRemaining)
	assert.True(t, result.ShopClosed)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
	assert.Equal(t, CloseReasonSoldOut, closed.CloseReason())
}

func TestPurchaseBundle_InsufficientBundles(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 3, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.OpenShop(m.Id())
	require.NoError(t, err)

	_, err = p.PurchaseBundle(2000, m.Id(), 0, 10)
	assert.ErrorIs(t, err, ErrInsufficientBundles)
}
