package shop

import (
	"testing"
	"time"

	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/listing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedSearchData creates, for one tenant: an Open shop in world 0, an Open
// shop in world 1, and a Maintenance shop in world 0 — each with one listing
// for item 2060000 at ascending prices — plus a second tenant's world-0 shop
// with the same item.
func seedSearchData(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()

	mkShop := func(tid uuid.UUID, characterId uint32, worldId world.Id, state State, title string) uuid.UUID {
		id := uuid.New()
		require.NoError(t, db.Create(&Entity{
			Model:        gorm.Model{CreatedAt: now, UpdatedAt: now},
			Id:           id,
			TenantId:     tid,
			CharacterId:  characterId,
			ShopType:     byte(CharacterShop),
			State:        byte(state),
			Title:        title,
			WorldId:      worldId,
			ChannelId:    1,
			MapId:        910000004,
			InstanceId:   uuid.Nil,
			PermitItemId: 5140000,
		}).Error)
		return id
	}
	mkListing := func(tid uuid.UUID, shopId uuid.UUID, itemId uint32, price uint32) {
		require.NoError(t, db.Create(&listing.Entity{
			Model:            gorm.Model{CreatedAt: now, UpdatedAt: now},
			Id:               uuid.New(),
			TenantId:         tid,
			ShopId:           shopId,
			ItemId:           itemId,
			ItemType:         2,
			Quantity:         100,
			BundleSize:       100,
			BundlesRemaining: 1,
			PricePerBundle:   price,
			ItemSnapshot:     asset.AssetData{Quantity: 100},
			ListedAt:         now,
		}).Error)
	}

	sA0 := mkShop(tidA, 1001, 0, Open, "w0 open")
	sA1 := mkShop(tidA, 1002, 1, Open, "w1 open")
	sAM := mkShop(tidA, 1003, 0, Maintenance, "w0 maint")
	sB0 := mkShop(tidB, 2001, 0, Open, "tenantB w0")

	mkListing(tidA, sA0, 2060000, 1000)
	mkListing(tidA, sA1, 2060000, 2000)
	mkListing(tidA, sAM, 2060000, 3000)
	mkListing(tidB, sB0, 2060000, 500)

	return db, tidA, tidB
}

func TestSearchListings_WorldScopedAndTenantScoped(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	w0 := world.Id(0)
	paged, err := searchListingsByItemIdPaged(tidA, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0}, model.Page{Number: 1, Size: MaxSearchResults})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	results := paged.Items
	require.Len(t, results, 2) // w0 open + w0 maintenance; w1 and tenantB excluded
	require.Equal(t, 2, paged.Total)
	require.Equal(t, uint32(1000), results[0].Listing.PricePerBundle())
	require.Equal(t, uint32(3000), results[1].Listing.PricePerBundle())
	require.Equal(t, uint32(1001), results[0].ShopOwnerId)
	require.Equal(t, CharacterShop, results[0].ShopType)
	require.Equal(t, Open, results[0].State)
	require.Equal(t, Maintenance, results[1].State)
}

func TestSearchListings_NoWorldFilterKeepsOldBehavior(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	paged, err := searchListingsByItemIdPaged(tidA, ListingSearchCriteria{ItemId: 2060000}, model.Page{Number: 1, Size: MaxSearchResults})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 3) // all tenant-A worlds, tenant B still excluded
	require.Equal(t, 3, paged.Total)
}

func TestSearchListings_DescendingOrder(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	w0 := world.Id(0)
	paged, err := searchListingsByItemIdPaged(tidA, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0, Descending: true}, model.Page{Number: 1, Size: MaxSearchResults})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 2)
	require.Equal(t, uint32(3000), paged.Items[0].Listing.PricePerBundle())
}

func TestSearchListings_PageCappedAt200(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tid := uuid.New()
	now := time.Now()
	shopId := uuid.New()
	require.NoError(t, db.Create(&Entity{
		Model: gorm.Model{CreatedAt: now, UpdatedAt: now}, Id: shopId, TenantId: tid,
		CharacterId: 1001, ShopType: byte(CharacterShop), State: byte(Open),
		Title: "bulk", WorldId: 0, ChannelId: 1, MapId: 910000004,
		InstanceId: uuid.Nil, PermitItemId: 5140000,
	}).Error)
	for i := 0; i < 205; i++ {
		require.NoError(t, db.Create(&listing.Entity{
			Model: gorm.Model{CreatedAt: now, UpdatedAt: now}, Id: uuid.New(), TenantId: tid,
			ShopId: shopId, ItemId: 2060000, ItemType: 2, Quantity: 1, BundleSize: 1,
			BundlesRemaining: 1, PricePerBundle: uint32(1000 + i),
			ItemSnapshot: asset.AssetData{Quantity: 1}, ListedAt: now,
		}).Error)
	}
	w0 := world.Id(0)
	// The game cap is now the route's max page size (task-117): one page
	// never exceeds MaxSearchResults, while Total reports the full match.
	paged, err := searchListingsByItemIdPaged(tid, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0}, model.Page{Number: 1, Size: MaxSearchResults})(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	results := paged.Items
	require.Len(t, results, MaxSearchResults)
	require.Equal(t, 205, paged.Total)
	// ascending truncates the most expensive tail
	require.Equal(t, uint32(1000), results[0].Listing.PricePerBundle())
	require.Equal(t, uint32(1000+MaxSearchResults-1), results[len(results)-1].Listing.PricePerBundle())
}
