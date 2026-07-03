package commodities_test

import (
	"atlas-npc/commodities"
	"atlas-npc/test"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedCommodityForPaging(t *testing.T, db *gorm.DB, tenantId uuid.UUID, templateId uint32, npcId uint32) {
	t.Helper()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	entity := commodities.Entity{
		Id:         uuid.New(),
		TenantId:   tenantId,
		NpcId:      npcId,
		TemplateId: templateId,
	}
	require.NoError(t, db.WithContext(ctx).Create(&entity).Error)
}

// TestGetCommoditiesByItemPaginates exercises the paginate.ParseParams /
// server.MarshalPaginatedResponse envelope on GET /commodities/items/{itemId}.
func TestGetCommoditiesByItemPaginates(t *testing.T) {
	db := test.SetupTestDB(t, commodities.Migration)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	for i := 0; i < 3; i++ {
		seedCommodityForPaging(t, db, tenantId, 1002357, uint32(9200000+i))
	}

	server := httptest.NewServer(setupRouter(db))
	defer server.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/1002357?page[number]=1&page[size]=2", server.URL)
		req := requestWithTenant("GET", url, tenantId)
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
		url := fmt.Sprintf("%s/commodities/items/1002357?page[size]=0", server.URL)
		req := requestWithTenant("GET", url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/1002357?limit=5", server.URL)
		req := requestWithTenant("GET", url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/1002357?page[number]=99&page[size]=2", server.URL)
		req := requestWithTenant("GET", url, tenantId)
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
