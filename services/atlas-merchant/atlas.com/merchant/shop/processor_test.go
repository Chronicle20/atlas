package shop

import (
	"atlas-merchant/frederick"
	message "atlas-merchant/kafka/message"
	asset2 "atlas-merchant/kafka/message/asset"
	compartment "atlas-merchant/kafka/message/compartment"
	merchantmsg "atlas-merchant/kafka/message/merchant"
	"atlas-merchant/listing"
	"atlas-merchant/visitor"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func testBuffer() *message.Buffer {
	return message.NewBuffer()
}

func TestCreateShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
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

	_, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 100000100, uuid.Nil, 0, 0, 0)
	assert.ErrorIs(t, err, ErrNotFreemarketRoom)
}

func TestCreateShop_DuplicateActiveShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	_, err := p.CreateShop(1000, CharacterShop, "Shop 1", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	_, err = p.CreateShop(1000, CharacterShop, "Shop 2", 0, 0, 910000002, uuid.Nil, 0, 0, 0)
	assert.ErrorIs(t, err, ErrShopLimitReached)
}

func TestOpenShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	assert.ErrorIs(t, err, ErrNoListings)
}

func TestCloseShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	require.NoError(t, err)

	err = p.CloseShop(mb)(m.Id(), 1000, CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
	assert.Equal(t, CloseReasonManualClose, closed.CloseReason())
}

// A personal shop closed by DISCONNECT (logout) must still return unsold items
// to the owner's inventory — the same AcceptAsset the manual close emits.
// Skipping it orphans the items on the closed shop (task-127 live bug: player
// logs out and their listed items never come back; personal shops have no
// Fredrick fallback).
func TestCloseShop_Disconnect_ReturnsPersonalShopItems(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)
	// itemId 2000004 = Elixir (a USE consumable) — mirrors the live case.
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000004, 2, 1, 100, 1000, asset2.AssetData{}, 2, 0)
	require.NoError(t, err)
	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	// Fresh buffer so we only observe the close's emissions.
	cmb := testBuffer()
	require.NoError(t, p.CloseShop(cmb)(m.Id(), 1000, CloseReasonDisconnect))

	assert.NotEmpty(t, cmb.GetAll()[compartment.EnvCommandTopic],
		"disconnect close of a personal shop must return unsold items to the owner")
}

func TestCloseShop_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	require.NoError(t, err)

	err = p.CloseShop(mb)(m.Id(), 1000, CloseReasonManualClose)
	require.NoError(t, err)

	// Closing an already-closed shop should fail.
	err = p.CloseShop(mb)(m.Id(), 1000, CloseReasonManualClose)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestAddListing(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	li, err := p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 5, 10, 1000, snapshot, 0, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	for i := 0; i < MaxListings; i++ {
		_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 1, 1000, snapshot, 0, 0)
		require.NoError(t, err)
	}

	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 1, 1000, snapshot, 0, 0)
	assert.ErrorIs(t, err, ErrListingLimitReached)
}

func TestPurchaseBundle(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 5, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	require.NoError(t, err)

	result, err := p.PurchaseBundle(mb)(2000, m.Id(), 0, 3, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 5, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	require.NoError(t, err)

	result, err := p.PurchaseBundle(mb)(2000, m.Id(), 0, 5, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 3, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	err = p.OpenShop(mb)(m.Id(), 1000)
	require.NoError(t, err)

	_, err = p.PurchaseBundle(mb)(2000, m.Id(), 0, 10, 0)
	assert.ErrorIs(t, err, ErrInsufficientBundles)
}

// --- State Transition Tests ---

func TestEnterMaintenance(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))
	require.NoError(t, p.EnterMaintenance(mb)(m.Id(), 1000))

	result, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Maintenance, result.State())
}

func TestEnterMaintenance_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	// Draft → Maintenance should fail.
	err = p.EnterMaintenance(mb)(m.Id(), 1000)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestExitMaintenance_Reopen(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))
	require.NoError(t, p.EnterMaintenance(mb)(m.Id(), 1000))
	require.NoError(t, p.ExitMaintenance(mb)(m.Id(), 1000))

	result, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Open, result.State())
}

func TestExitMaintenance_CloseWhenEmpty(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))
	require.NoError(t, p.EnterMaintenance(mb)(m.Id(), 1000))

	// Remove the only listing.
	_, err = p.RemoveListing(mb)(m.Id(), 1000, 0)
	require.NoError(t, err)

	require.NoError(t, p.ExitMaintenance(mb)(m.Id(), 1000))

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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	// Draft → ExitMaintenance should fail.
	err = p.ExitMaintenance(mb)(m.Id(), 1000)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

// --- Listing Operation Tests ---

func TestRemoveListing_DisplayOrderCollapse(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0) // index 0
	require.NoError(t, err)
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000001, 0, 1, 10, 2000, snapshot, 0, 0) // index 1
	require.NoError(t, err)
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000002, 0, 1, 10, 3000, snapshot, 0, 0) // index 2
	require.NoError(t, err)

	// Remove the first listing (index 0).
	removed, err := p.RemoveListing(mb)(m.Id(), 1000, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	// Open state should not allow adding listings.
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000001, 0, 1, 10, 1000, snapshot, 0, 0)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestAddListing_ZeroValues(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}

	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 1, 0, snapshot, 0, 0)
	assert.Error(t, err)

	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 0, 1, 1000, snapshot, 0, 0)
	assert.Error(t, err)

	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 0, 1000, snapshot, 0, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, HiredMerchant, "Hired Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, m.ExpiresAt())

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	// Purchase 3 bundles at 1000 each = 3000 total. Below 100k so no fee.
	result, err := p.PurchaseBundle(mb)(2000, m.Id(), 0, 3, 0)
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

	m, err := p.CreateShop(1000, CharacterShop, "Char Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)
	assert.Nil(t, m.ExpiresAt())
}

func TestCharacterShop_NoMesoAccumulation(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	_, err = p.PurchaseBundle(mb)(2000, m.Id(), 0, 3, 0)
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
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	_, err = p.PurchaseBundle(mb)(2000, m.Id(), 0, 0, 0)
	assert.Error(t, err)
}

func TestCloseShop_FromDraft(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	err = p.CloseShop(mb)(m.Id(), 1000, CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
}

// TestRetrieveFrederick_ClearFailure_SkipsOutbox asserts that when a
// Frederick-storage clear fails (ClearNotifications, the third and last of
// the three clear calls), RetrieveFrederick returns the error instead of
// swallowing it, and no outbox row gets enqueued for the asset/meso grant
// commands already buffered — i.e. a failed clear does not let the retrieve
// silently "succeed" and double-grant on a later retry.
//
// This uses table-drop as the failure-injection seam (frederick.Processor
// has no mock/fake in this module and shop.ProcessorImpl constructs it
// directly via frederick.NewProcessor(p.l, p.ctx, p.db), so there is no
// injectable interface at the shop-processor level). Note: because of the
// separately-tracked ExecuteTransaction-is-a-no-op bug (see
// bug_execute_transaction_noop in project memory / task-119), the ClearItems
// delete that already ran before ClearNotifications fails is NOT rolled
// back at the SQL level today — this test does not assert DB-row rollback,
// only the two things this fix pass actually delivers: error propagation
// and outbox-enqueue suppression.
func TestRetrieveFrederick_ClearFailure_SkipsOutbox(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, outboxlib.Migration(db))
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()

	fp := frederick.NewProcessor(l, ctx, db)
	require.NoError(t, fp.StoreItems(1000, []frederick.StoredItem{
		{ItemId: 2000000, ItemType: 0, Quantity: 5, ItemSnapshot: asset2.AssetData{}},
	}))

	// Force the third clear call (ClearNotifications) to fail.
	require.NoError(t, db.Migrator().DropTable("frederick_notifications"))

	p := NewProcessor(l, ctx, db)
	err := p.RetrieveFrederickAndEmit(1000, 0)
	require.Error(t, err)

	var outboxCount int64
	require.NoError(t, db.Model(&outboxlib.Entity{}).Count(&outboxCount).Error)
	assert.Equal(t, int64(0), outboxCount, "no outbox row should be enqueued when a Frederick clear fails")
}

// TestCloseShop_FrederickStoreFailure_SkipsOutbox asserts that when storing
// unsold listings to Frederick fails on shop close (storeToFrederick's
// StoreItems call), CloseShop returns the error instead of swallowing it,
// and no shop-closed outbox row gets enqueued — i.e. a failed store does not
// let the close silently "succeed" while the items vanish.
//
// Same table-drop injection seam and same task-119 caveat as above: the
// state-transition write inside CloseShop's own nested ExecuteTransaction
// call already committed (autocommit, no real tx) before storeToFrederick
// fails, so this test does not assert the shop stays Open — only error
// propagation and outbox-enqueue suppression.
func TestCloseShop_FrederickStoreFailure_SkipsOutbox(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, outboxlib.Migration(db))
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, HiredMerchant, "Hired Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))

	// Force Frederick item storage to fail.
	require.NoError(t, db.Migrator().DropTable("frederick_items"))

	err = p.CloseShopAndEmit(m.Id(), 1000, CloseReasonManualClose)
	require.Error(t, err)

	var outboxCount int64
	require.NoError(t, db.Model(&outboxlib.Entity{}).Count(&outboxCount).Error)
	assert.Equal(t, int64(0), outboxCount, "no outbox row should be enqueued when Frederick storage fails")
}

func TestCloseShop_FromMaintenance(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(1000, CharacterShop, "Test Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 0)
	require.NoError(t, err)

	snapshot := asset2.AssetData{}
	_, err = p.AddListing(mb)(m.Id(), 1000, 2000000, 0, 1, 10, 1000, snapshot, 0, 0)
	require.NoError(t, err)

	require.NoError(t, p.OpenShop(mb)(m.Id(), 1000))
	require.NoError(t, p.EnterMaintenance(mb)(m.Id(), 1000))

	err = p.CloseShop(mb)(m.Id(), 1000, CloseReasonManualClose)
	require.NoError(t, err)

	closed, err := p.GetById(m.Id())
	require.NoError(t, err)
	assert.Equal(t, Closed, closed.State())
}

// setupTestRegistries wires both the shop activeShops registry and the visitor
// registry to a per-test miniredis so occupancy resolution can be exercised.
func setupTestRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
	visitor.InitRegistry(client)
}

// GetShopForCharacter must resolve the OWNER of an owner-attached shop —
// otherwise every owner-side op (OPEN, PUT_ITEM, EXIT, CHAT) that the channel
// routes through /characters/{id}/visiting 404s on a freshly created Draft
// shop, the owner can neither stock nor open nor close it, and the stranded
// Draft blocks re-creation.
func TestGetShopForCharacter_OwnerOfDraftPersonalShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2000, CharacterShop, "Owner Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)

	got, err := p.GetShopForCharacter(2000)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got)
}

func TestGetShopForCharacter_OwnerOfDraftHiredMerchant(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2001, HiredMerchant, "Merch Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)

	got, err := p.GetShopForCharacter(2001)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got)
}

// A hired merchant running Open is owner-detached: the owner is NOT occupying
// it (they may be wandering, or visiting another shop — the visitor registry
// takes precedence then).
func TestGetShopForCharacter_OpenHiredMerchantOwnerDetached(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2002, HiredMerchant, "Merch Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)

	// Force the shop to Open (bypassing the listing requirement).
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Open)).Error)

	_, err = p.GetShopForCharacter(2002)
	assert.Error(t, err, "owner of an Open hired merchant is not occupying it")

	// But while visiting someone else's shop, the visitor registry resolves.
	other, err := p.CreateShop(2003, CharacterShop, "Other Shop", 0, 0, 910000001, uuid.Nil, 500, 0, 5140000)
	require.NoError(t, err)
	require.NoError(t, visitor.GetRegistry().AddVisitor(ctx, tenant.MustFromContext(ctx), other.Id(), 2002))
	got, err := p.GetShopForCharacter(2002)
	require.NoError(t, err)
	assert.Equal(t, other.Id(), got)
}

// A hired merchant in Maintenance is owner-attached again (management view).
func TestGetShopForCharacter_MaintenanceHiredMerchantOwnerAttached(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2004, HiredMerchant, "Merch Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Maintenance)).Error)

	got, err := p.GetShopForCharacter(2004)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got)
}

// A personal-shop owner must resolve their own shop even when the Redis
// occupancy entry is unavailable at read time (eviction, an uncommitted
// CreateShop Put, a close-desync). The DB is authoritative; otherwise every
// owner-side op 404s on /characters/{id}/visiting and the client freezes
// (task-127 live bug: add-item hangs, close never fires, re-create blocked).
func TestGetShopForCharacter_PersonalShop_OccupancyEvicted_DBFallback(t *testing.T) {
	db := setupTestDB(t)
	ctx, ten := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2200, CharacterShop, "Owner Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)

	// Simulate the occupancy entry being gone when the owner acts.
	require.NoError(t, GetRegistry().activeShops.Remove(ctx, ten, 2200))

	got, err := p.GetShopForCharacter(2200)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got)

	// The DB fallback re-seeds Redis so the fast path is restored.
	entry, err := GetRegistry().activeShops.Get(ctx, ten, 2200)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), entry.ShopId)
}

// Occupancy eviction must not accidentally resolve an owner-detached Open
// hired merchant — the DB fallback honors the same owner-attached rule
// (personal: any non-Closed; hired merchant: only Draft/Maintenance).
func TestGetShopForCharacter_OpenHiredMerchant_OccupancyEvicted_StillDetached(t *testing.T) {
	db := setupTestDB(t)
	ctx, ten := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(2201, HiredMerchant, "Merch", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Open)).Error)
	require.NoError(t, GetRegistry().activeShops.Remove(ctx, ten, 2201))

	_, err = p.GetShopForCharacter(2201)
	assert.Error(t, err, "owner of an Open hired merchant is not occupying it, even without a cache entry")
}

// A stale occupancy entry pointing at a now-Closed shop must not 404 the owner
// when they actually have a fresh active shop — the DB fallback finds the real
// one (and a Closed shop is never returned as occupied).
func TestGetShopForCharacter_StaleOccupancy_ResolvesActiveShop(t *testing.T) {
	db := setupTestDB(t)
	ctx, ten := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	old, err := p.CreateShop(2202, CharacterShop, "Old", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", old.Id()).Update("state", byte(Closed)).Error)

	fresh, err := p.CreateShop(2202, CharacterShop, "Fresh", 0, 0, 910000002, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)

	// Force occupancy to point at the stale (Closed) shop.
	require.NoError(t, GetRegistry().activeShops.Put(ctx, ten, 2202, ActiveShopEntry{ShopId: old.Id(), ShopType: CharacterShop}))

	got, err := p.GetShopForCharacter(2202)
	require.NoError(t, err)
	assert.Equal(t, fresh.Id(), got)
}

// AddListing must emit SHOP_UPDATED so the channel refreshes the owner's store
// view (UPDATE_MERCHANT). Without it the client that just dropped an item into a
// slot gets no reply and freezes (task-127 live bug).
func TestAddListing_EmitsShopUpdated(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(3000, CharacterShop, "Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)

	_, err = p.AddListing(mb)(m.Id(), 3000, 2000000, 0, 1, 10, 1000, asset2.AssetData{}, 0, 0)
	require.NoError(t, err)

	assert.NotEmpty(t, mb.GetAll()[merchantmsg.EnvStatusEventTopic], "AddListing must emit SHOP_UPDATED")
}

// RemoveListing must likewise refresh the view after pulling an item back.
func TestRemoveListing_EmitsShopUpdated(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	mb := testBuffer()

	m, err := p.CreateShop(3001, CharacterShop, "Shop", 0, 0, 910000001, uuid.Nil, 0, 0, 5140000)
	require.NoError(t, err)
	_, err = p.AddListing(mb)(m.Id(), 3001, 2000000, 0, 1, 10, 1000, asset2.AssetData{}, 0, 0)
	require.NoError(t, err)

	rmb := testBuffer()
	_, err = p.RemoveListing(rmb)(m.Id(), 3001, 0)
	require.NoError(t, err)

	assert.NotEmpty(t, rmb.GetAll()[merchantmsg.EnvStatusEventTopic], "RemoveListing must emit SHOP_UPDATED")
}

// A hired merchant abandoned in Draft (owner created the shop, closed the
// window, never opened) must still be reaped by the expiry task — otherwise
// it never leaves Draft and permanently blocks the character from creating
// another shop of that type.
func TestGetExpired_IncludesDraftHiredMerchant(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db)

	m, err := p.CreateShop(3000, HiredMerchant, "Stale Draft", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)

	// Age the shop past its 24h expiry.
	past := time.Now().Add(-25 * time.Hour)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("expires_at", past).Error)

	expired, err := p.GetExpired()
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, m.Id(), expired[0].Id())
	assert.Equal(t, Draft, expired[0].State())
}
