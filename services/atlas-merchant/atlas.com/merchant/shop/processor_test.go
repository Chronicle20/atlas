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

// --- State Transition Tests ---

func TestEnterMaintenance(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))
	require.NoError(t, p.EnterMaintenance(m.Id()))

	result, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Maintenance, result.State())
}

func TestEnterMaintenance_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	// Draft → Maintenance should fail.
	err = p.EnterMaintenance(m.Id())
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestExitMaintenance_Reopen(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))
	require.NoError(t, p.EnterMaintenance(m.Id()))

	closed, err := p.ExitMaintenance(m.Id())
	require.NoError(t, err)
	assert.False(t, closed)

	result, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Open, result.State())
}

func TestExitMaintenance_CloseWhenEmpty(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))
	require.NoError(t, p.EnterMaintenance(m.Id()))

	// Remove the only listing.
	_, err = p.RemoveListing(m.Id(), 0)
	require.NoError(t, err)

	closed, err := p.ExitMaintenance(m.Id())
	require.NoError(t, err)
	assert.True(t, closed)

	result, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, result.State())
	assert.Equal(t, CloseReasonEmpty, result.CloseReason())
}

func TestExitMaintenance_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	// Draft → ExitMaintenance should fail.
	_, err = p.ExitMaintenance(m.Id())
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

// --- Listing Operation Tests ---

func TestRemoveListing_DisplayOrderCollapse(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0) // index 0
	require.NoError(t, err)
	_, err = p.AddListing(m.Id(), 2000001, 0, 1, 10, 2000, snapshot, 0) // index 1
	require.NoError(t, err)
	_, err = p.AddListing(m.Id(), 2000002, 0, 1, 10, 3000, snapshot, 0) // index 2
	require.NoError(t, err)

	// Remove the first listing (index 0).
	removed, err := p.RemoveListing(m.Id(), 0)
	require.NoError(t, err)
	assert.Equal(t, uint32(2000000), removed.ItemId())

	// Remaining listings should have collapsed display orders.
	listings, err := p.GetListings(m.Id())
	require.NoError(t, err)
	assert.Len(t, listings, 2)

	displayOrders := map[uint16]uint32{}
	for _, li := range listings {
		displayOrders[li.DisplayOrder()] = li.ItemId()
	}
	assert.Equal(t, uint32(2000001), displayOrders[0])
	assert.Equal(t, uint32(2000002), displayOrders[1])
}

func TestUpdateListing(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	err = p.UpdateListing(m.Id(), 0, 2000, 5, 20)
	require.NoError(t, err)

	listings, err := p.GetListings(m.Id())
	require.NoError(t, err)
	require.Len(t, listings, 1)
	assert.Equal(t, uint32(2000), listings[0].PricePerBundle())
	assert.Equal(t, uint16(5), listings[0].BundleSize())
	assert.Equal(t, uint16(20), listings[0].BundlesRemaining())
}

func TestAddListing_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))

	// Open state should not allow adding listings.
	_, err = p.AddListing(m.Id(), 2000001, 0, 1, 10, 1000, snapshot, 0)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestAddListing_ZeroValues(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})

	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 1, 0, snapshot, 0)
	assert.Error(t, err)

	_, err = p.AddListing(m.Id(), 2000000, 0, 0, 1, 1000, snapshot, 0)
	assert.Error(t, err)

	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 0, 1000, snapshot, 0)
	assert.Error(t, err)
}

// --- Fee Tier Tests ---

func TestGetFee_Tiers(t *testing.T) {
	tests := []struct {
		name     string
		meso     int64
		expected int64
	}{
		{"below 100k — no fee", 99999, 0},
		{"exactly 100k — 0.8%", 100000, 800},
		{"500k — 0.8%", 500000, 4000},
		{"exactly 1M — 1.8%", 1000000, 18000},
		{"3M — 1.8%", 3000000, 54000},
		{"exactly 5M — 3%", 5000000, 150000},
		{"8M — 3%", 8000000, 240000},
		{"exactly 10M — 4%", 10000000, 400000},
		{"20M — 4%", 20000000, 800000},
		{"exactly 25M — 5%", 25000000, 1250000},
		{"50M — 5%", 50000000, 2500000},
		{"exactly 100M — 6%", 100000000, 6000000},
		{"200M — 6%", 200000000, 12000000},
		{"zero — no fee", 0, 0},
		{"1 meso — no fee", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetFee(tt.meso))
		})
	}
}

// --- Hired Merchant Tests ---

func TestHiredMerchant_MesoAccumulation(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, HiredMerchant, "Hired Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, m.ExpiresAt())

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))

	// Purchase 3 bundles at 1000 each = 3000 total. Below 100k so no fee.
	result, err := p.PurchaseBundle(2000, m.Id(), 0, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), result.TotalCost)
	assert.Equal(t, int64(0), result.Fee)
	assert.Equal(t, int64(3000), result.NetAmount)

	// Verify meso balance accumulated.
	updated, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, uint32(3000), updated.MesoBalance())
}

func TestCharacterShop_NoExpiry(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Char Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)
	assert.Nil(t, m.ExpiresAt())
}

func TestCharacterShop_NoMesoAccumulation(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))

	_, err = p.PurchaseBundle(2000, m.Id(), 0, 3)
	require.NoError(t, err)

	// Character shop should not accumulate meso balance.
	updated, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, uint32(0), updated.MesoBalance())
}

func TestPurchaseBundle_ZeroBundleCount(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))

	_, err = p.PurchaseBundle(2000, m.Id(), 0, 0)
	assert.Error(t, err)
}

func TestCloseShop_FromDraft(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	err = p.CloseShop(m.Id(), CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
}

func TestCloseShop_FromMaintenance(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 910000001, 0, 0, 0)
	require.NoError(t, err)

	snapshot, _ := json.Marshal(map[string]interface{}{"flag": 0})
	_, err = p.AddListing(m.Id(), 2000000, 0, 1, 10, 1000, snapshot, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(m.Id()))
	require.NoError(t, p.EnterMaintenance(m.Id()))

	err = p.CloseShop(m.Id(), CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
}
