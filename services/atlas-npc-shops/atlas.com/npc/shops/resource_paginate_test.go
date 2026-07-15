package shops_test

import (
	"atlas-npc/commodities"
	"atlas-npc/data/consumable"
	"atlas-npc/shops"
	"atlas-npc/test"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupShopsRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := shops.InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenantShops(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedShop(t *testing.T, db *gorm.DB, tenantId uuid.UUID, npcId uint32, recharger bool) {
	t.Helper()
	require.NoError(t, db.Create(&shops.Entity{
		Id:        uuid.New(),
		TenantId:  tenantId,
		NpcId:     npcId,
		Recharger: recharger,
	}).Error)
}

// TestGetAllShopsPaginates drives GET /shops through the real resource
// router (InitResource) against an in-memory tenant-scoped DB. /shops is
// content full-table (no per-request WHERE filter), so its default page
// size is 50 (paginate.DefaultPageSize), not the 250 game-cap used by the
// other two routes in this service.
func TestGetAllShopsPaginates(t *testing.T) {
	db := test.SetupTestDB(t, shops.Migration, commodities.Migration)
	defer test.CleanupTestDB(t, db)

	// Consumable cache must be initialized before any AllShopsProvider call
	// (RechargeableConsumablesDecorator is unconditional, not include-gated).
	mockCache := &mockConsumableCache{consumables: map[uuid.UUID][]consumable.Model{}}
	original := shops.GetConsumableCache()
	shops.SetConsumableCacheForTesting(mockCache)
	defer shops.SetConsumableCacheForTesting(original)

	tenantId := uuid.New()
	seedShop(t, db, tenantId, 9000001, false)
	seedShop(t, db, tenantId, 9000002, false)
	seedShop(t, db, tenantId, 9000003, false)

	srv := httptest.NewServer(setupShopsRouter(db))
	defer srv.Close()

	t.Run("DefaultPageSizeIs50NotGameCap250", func(t *testing.T) {
		url := fmt.Sprintf("%s/shops", srv.URL)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 50, page["size"])
	})

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/shops?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])
		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/shops?page[size]=0", srv.URL)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/shops?limit=5", srv.URL)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/shops?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
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

// TestGetAllShopsAppliesMandatoryDecorationAfterPaging proves
// RechargeableConsumablesDecorator (unconditional, not include-gated) still
// runs on the paged result — not the pre-pagination full set — by seeding a
// recharger shop with one commodity and a mock consumable cache entry for
// the same templateId with a distinct SlotMax/UnitPrice, then asserting the
// response's included commodity carries the DECORATED values, not the raw
// DB-stored ones.
func TestGetAllShopsAppliesMandatoryDecorationAfterPaging(t *testing.T) {
	db := test.SetupTestDB(t, shops.Migration, commodities.Migration)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	npcId := uint32(9100001)
	seedShop(t, db, tenantId, npcId, true)
	require.NoError(t, db.WithContext(ctx).Create(&commodities.Entity{
		Id:         uuid.New(),
		TenantId:   tenantId,
		NpcId:      npcId,
		TemplateId: 2330000,
		MesoPrice:  100,
	}).Error)

	raw := `{"id":2330000,"slotMax":777,"unitPrice":12.5}`
	var rechargeable consumable.Model
	require.NoError(t, json.Unmarshal([]byte(raw), &rechargeable))

	mockCache := &mockConsumableCache{consumables: map[uuid.UUID][]consumable.Model{tenantId: {rechargeable}}}
	original := shops.GetConsumableCache()
	shops.SetConsumableCacheForTesting(mockCache)
	defer shops.SetConsumableCacheForTesting(original)

	srv := httptest.NewServer(setupShopsRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/shops?include=commodities", srv.URL)
	req := requestWithTenantShops(http.MethodGet, url, tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.Len(t, doc.Included, 1)

	var attrs struct {
		SlotMax   uint32  `json:"slotMax"`
		UnitPrice float64 `json:"unitPrice"`
	}
	require.NoError(t, json.Unmarshal(doc.Included[0].Attributes, &attrs))
	assert.EqualValues(t, 777, attrs.SlotMax)
	assert.EqualValues(t, 12.5, attrs.UnitPrice)
}

// TestGetShopCharactersPaginates drives GET /npcs/{npcId}/shop/characters
// through the real resource router against a miniredis-backed registry
// (GetCharactersInShop is a Redis set, not a DB table).
func TestGetShopCharactersPaginates(t *testing.T) {
	db := test.SetupTestDB(t, shops.Migration, commodities.Migration)
	defer test.CleanupTestDB(t, db)

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	shops.InitRegistry(client)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	npcId := uint32(9200001)
	// Added out of ascending order; Redis set membership order is not
	// guaranteed either way, so the handler's sort is what must determine
	// page contents.
	for _, characterId := range []uint32{300, 100, 200} {
		shops.GetRegistry().AddCharacter(ctx, characterId, npcId)
	}

	srv := httptest.NewServer(setupShopsRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/npcs/%d/shop/characters?page[number]=1&page[size]=2", srv.URL, npcId)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])
		assert.EqualValues(t, 2, page["size"])

		// Stable-sorted ascending: page 1 must be characterId 100 then 200.
		assert.Equal(t, "100", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/npcs/%d/shop/characters?page[size]=0", srv.URL, npcId)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/npcs/%d/shop/characters?limit=5", srv.URL, npcId)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/npcs/%d/shop/characters?page[number]=99&page[size]=2", srv.URL, npcId)
		req := requestWithTenantShops(http.MethodGet, url, tenantId)
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
