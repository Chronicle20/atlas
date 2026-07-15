package shop_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	blacklistpkg "atlas-merchant/blacklist"
	"atlas-merchant/frederick"
	message "atlas-merchant/kafka/message"
	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/listing"
	searchcountpkg "atlas-merchant/searchcount"
	"atlas-merchant/shop"
	visitpkg "atlas-merchant/visit"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupMerchantRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := shop.InitializeRoutes(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// merchantTestContext builds a tenant-scoped context matching the tenant
// header combination requestWithTenant sends, so a processor seeding via
// this context writes rows the HTTP layer's tenant scoping can see.
func merchantTestContext(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func shopTestBuffer() *message.Buffer {
	return message.NewBuffer()
}

// seedOpenShop creates a shop, adds numListings listings to it (item ids
// itemId, itemId+1, ...), and opens it (via the real shop.Processor,
// mirroring processor_test.go's seeding convention) so it satisfies
// GetAllOpenPaged's `state IN (Open, Maintenance)` filter. x/y are derived
// from characterId so shops seeded on the same map land far enough apart to
// pass CreateShop's shop-proximity validation (threshold 100).
func seedOpenShop(t *testing.T, db *gorm.DB, ctx context.Context, characterId uint32, mapId uint32, itemId uint32, numListings int, title string) shop.Model {
	t.Helper()
	l, _ := test.NewNullLogger()
	p := shop.NewProcessor(l, ctx, db)
	mb := shopTestBuffer()

	x := int16((characterId % 20) * 500)
	m, err := p.CreateShop(characterId, shop.CharacterShop, title, 0, 0, mapId, uuid.Nil, x, 0, 0)
	require.NoError(t, err)

	snapshot := asset.AssetData{}
	for i := 0; i < numListings; i++ {
		_, err = p.AddListing(mb)(m.Id(), characterId, itemId+uint32(i), 0, 1, 10, 1000, snapshot, 0, 0)
		require.NoError(t, err)
	}

	require.NoError(t, p.OpenShop(mb)(m.Id(), characterId))

	opened, err := p.GetById(m.Id())
	require.NoError(t, err)
	return opened
}

// TestGetMerchantsPaginates drives the bare GET /merchants full-table list
// route (50/250) through the real resource router.
func TestGetMerchantsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	seedOpenShop(t, db, ctx, 1001, 910000001, 2000000, 1, "shop1")
	seedOpenShop(t, db, ctx, 1002, 910000002, 2000000, 1, "shop2")
	seedOpenShop(t, db, ctx, 1003, 910000003, 2000000, 1, "shop3")

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}

// TestGetCharacterMerchantsPaginates drives
// GET /characters/{characterId}/merchants (game-capped 250/250).
func TestGetCharacterMerchantsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	seedOpenShop(t, db, ctx, 2001, 910000001, 2000000, 1, "character shop")
	// A different character's shop must not appear in the filtered results.
	seedOpenShop(t, db, ctx, 2002, 910000002, 2000000, 1, "other character shop")

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FiltersByCharacterAndPaginates", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/2001/merchants", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 1, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 250, page["size"])
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/2001/merchants?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/2001/merchants?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetFieldMerchantsPaginates drives
// GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/merchants
// (game-capped 250/250), including the listing-count decoration.
func TestGetFieldMerchantsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	// Both field shops carry the same listing count so the ListingCount
	// decoration assertion below doesn't depend on which one lands on page 1
	// (PagedQuery orders by primary key, i.e. a random UUID, not insertion
	// order).
	seedOpenShop(t, db, ctx, 3001, 910000001, 2000000, 1, "field shop 1")
	seedOpenShop(t, db, ctx, 3002, 910000001, 2000000, 1, "field shop 2")
	// A different map must not appear in the filtered results.
	seedOpenShop(t, db, ctx, 3003, 910000002, 2000000, 1, "other field shop")

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfOneFiltersByFieldAndDecoratesListingCount", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/0/channels/0/maps/910000001/instances/%s/merchants?page[number]=1&page[size]=1", srv.URL, uuid.Nil.String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 2, doc.Meta["total"])

		var attrs struct {
			ListingCount int64 `json:"listingCount"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &attrs))
		assert.EqualValues(t, 1, attrs.ListingCount)
	})
}

// TestSearchListingsPaginates drives GET /merchants/search/listings?itemId=X
// (game-capped 250/250, joined query keeping its item/state Where and
// price-ascending order) through the real resource router.
func TestSearchListingsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	// Two shops each list the searched item; a third shop lists a different
	// item and must not appear in the results.
	seedOpenShop(t, db, ctx, 4001, 910000001, 3000000, 1, "search shop 1")
	seedOpenShop(t, db, ctx, 4002, 910000002, 3000000, 1, "search shop 2")
	seedOpenShop(t, db, ctx, 4003, 910000003, 3000001, 1, "other item shop")

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfOneFiltersByItemId", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/search/listings?itemId=3000000&page[number]=1&page[size]=1", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 2, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 1, page["size"])
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("MissingItemIdIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/search/listings", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/search/listings?itemId=3000000&page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/search/listings?itemId=3000000&limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetMerchantListingsPaginates drives
// GET /merchants/{shopId}/relationships/listings (game-capped 250/250,
// bounded in practice by shop.MaxListings=16) through the real resource
// router.
func TestGetMerchantListingsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	m := seedOpenShop(t, db, ctx, 5001, 910000001, 6000000, 3, "listings shop")

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/relationships/listings?page[number]=1&page[size]=2", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/relationships/listings?page[size]=0", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/relationships/listings?limit=5", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetMerchantBlacklistPaginates drives GET /merchants/{shopId}/blacklist
// (250/250, task-117) through the real resource router. The route was added
// unpaged by task-127 (owl/mini-room work) and folded into the pagination
// convention when task-117 rebased over it.
func TestGetMerchantBlacklistPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration, blacklistpkg.Migration, visitpkg.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	m := seedOpenShop(t, db, ctx, 1001, 910000001, 2000000, 1, "shop1")
	l, _ := test.NewNullLogger()
	bp := blacklistpkg.NewProcessor(l, ctx, db)
	require.NoError(t, bp.Add(m.Id(), "Alice"))
	require.NoError(t, bp.Add(m.Id(), "Bob"))
	require.NoError(t, bp.Add(m.Id(), "Carol"))

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/blacklist?page[number]=1&page[size]=2", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])
	})

	t.Run("DefaultIsWholeSet", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/blacklist", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 3)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/blacklist?page[size]=0", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetMerchantVisitsPaginates drives GET /merchants/{shopId}/visits
// (250/250, task-117) through the real resource router.
func TestGetMerchantVisitsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration, blacklistpkg.Migration, visitpkg.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	m := seedOpenShop(t, db, ctx, 1001, 910000001, 2000000, 1, "shop1")
	l, _ := test.NewNullLogger()
	vp := visitpkg.NewProcessor(l, ctx, db)
	require.NoError(t, vp.Record(m.Id(), "Alice"))
	require.NoError(t, vp.Record(m.Id(), "Alice"))
	require.NoError(t, vp.Record(m.Id(), "Bob"))
	require.NoError(t, vp.Record(m.Id(), "Carol"))

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoOrderedByCount", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/visits?page[number]=1&page[size]=2", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		// Alice has 2 visits and must lead the count-DESC ordering.
		var first map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		assert.Equal(t, "Alice", first["name"])
		assert.EqualValues(t, 2, first["count"])

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/merchants/%s/visits?page[size]=0", srv.URL, m.Id().String())
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetTopShopSearchesPaginates drives GET /worlds/{worldId}/shop-searches/top
// (bounded top-N via paginate.Slice, task-117) through the real resource
// router.
func TestGetTopShopSearchesPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, shop.Migration, listing.Migration, frederick.Migration, searchcountpkg.Migration)
	tenantId := uuid.New()
	ctx := merchantTestContext(t, tenantId)

	l, _ := test.NewNullLogger()
	sp := searchcountpkg.NewProcessor(l, ctx, db)
	// item 2000001 searched twice, 2000000 once — count DESC puts 2000001 first.
	require.NoError(t, sp.RecordSearch(0, 2000001))
	require.NoError(t, sp.RecordSearch(0, 2000001))
	require.NoError(t, sp.RecordSearch(0, 2000000))

	srv := httptest.NewServer(setupMerchantRouter(db))
	defer srv.Close()

	t.Run("EnvelopeWithWholeTopList", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/0/shop-searches/top", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 2, doc.Meta["total"])
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/0/shop-searches/top?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
