package shop

import (
	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/listing"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// seedSearchFixture creates one open shop and n listings for itemId on it,
// with strictly increasing PricePerBundle (1000, 2000, 3000, ...), directly
// at the entity layer (bypassing CreateShop's portal/proximity validation,
// which is irrelevant to the paged-query math under test here).
func seedSearchFixture(t *testing.T, db *gorm.DB, tenantId uuid.UUID, itemId uint32, n int) {
	t.Helper()
	shopId := uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		Model:        gorm.Model{CreatedAt: now, UpdatedAt: now},
		Id:           shopId,
		TenantId:     tenantId,
		CharacterId:  9001,
		ShopType:     byte(CharacterShop),
		State:        byte(Open),
		Title:        "search fixture shop",
		MapId:        910000001,
		InstanceId:   uuid.Nil,
		PermitItemId: 0,
	}).Error)

	for i := 0; i < n; i++ {
		require.NoError(t, db.Create(&listing.Entity{
			Model:            gorm.Model{CreatedAt: now, UpdatedAt: now},
			Id:               uuid.New(),
			TenantId:         tenantId,
			ShopId:           shopId,
			ItemId:           itemId,
			ItemType:         0,
			Quantity:         10,
			BundleSize:       1,
			BundlesRemaining: 10,
			PricePerBundle:   uint32(1000 * (i + 1)),
			ItemSnapshot:     asset.AssetData{},
			DisplayOrder:     uint16(i),
			Version:          1,
			ListedAt:         now,
		}).Error)
	}
}

// TestSearchListingsByItemIdPaged_OrdersByPriceAndPaginates guards the
// hand-rolled pagination in searchListingsByItemIdPaged (task-117): it does
// not use database.PagedQuery (see the function's doc comment on the
// ambiguous unqualified `id` join hazard), so its Total/Page math and
// price-ascending ordering need direct coverage.
func TestSearchListingsByItemIdPaged_OrdersByPriceAndPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)

	seedSearchFixture(t, db, tenantId, 7000000, 5)

	firstPage, err := searchListingsByItemIdPaged(tenantId, ListingSearchCriteria{ItemId: 7000000}, model.Page{Number: 1, Size: 2})(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, 5, firstPage.Total)
	require.Len(t, firstPage.Items, 2)
	assert.EqualValues(t, 1000, firstPage.Items[0].Listing.PricePerBundle())
	assert.EqualValues(t, 2000, firstPage.Items[1].Listing.PricePerBundle())

	secondPage, err := searchListingsByItemIdPaged(tenantId, ListingSearchCriteria{ItemId: 7000000}, model.Page{Number: 2, Size: 2})(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, 5, secondPage.Total)
	require.Len(t, secondPage.Items, 2)
	assert.EqualValues(t, 3000, secondPage.Items[0].Listing.PricePerBundle())
	assert.EqualValues(t, 4000, secondPage.Items[1].Listing.PricePerBundle())

	thirdPage, err := searchListingsByItemIdPaged(tenantId, ListingSearchCriteria{ItemId: 7000000}, model.Page{Number: 3, Size: 2})(db.WithContext(ctx))()
	require.NoError(t, err)
	assert.Equal(t, 5, thirdPage.Total)
	require.Len(t, thirdPage.Items, 1)
	assert.EqualValues(t, 5000, thirdPage.Items[0].Listing.PricePerBundle())
}

// TestSearchListingsByItemIdPaged_FiltersByTenant guards that the joined
// query is still tenant-scoped after the pagination rewrite.
func TestSearchListingsByItemIdPaged_FiltersByTenant(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tidA, tidB := uuid.New(), uuid.New()

	seedSearchFixture(t, db, tidA, 7000001, 1)
	seedSearchFixture(t, db, tidB, 7000001, 1)

	gotA, err := searchListingsByItemIdPaged(tidA, ListingSearchCriteria{ItemId: 7000001}, model.Page{Number: 1, Size: 10})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, 1, gotA.Total)
	require.Len(t, gotA.Items, 1)
}
