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
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

func setupRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := commodities.InitResource(&testServerInformation{})(db)
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

func seedCommodity(t *testing.T, db *gorm.DB, tenantId uuid.UUID, templateId uint32, npcId uint32, mesoPrice uint32) {
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	entity := commodities.Entity{
		Id:         uuid.New(),
		TenantId:   tenantId,
		NpcId:      npcId,
		TemplateId: templateId,
		MesoPrice:  mesoPrice,
	}
	require.NoError(t, db.WithContext(ctx).Create(&entity).Error)
}

func TestGetCommoditiesByItem(t *testing.T) {
	db := test.SetupTestDB(t, commodities.Migration)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	otherTenantId := uuid.New()

	seedCommodity(t, db, tenantId, 1002357, 9200000, 50000)
	seedCommodity(t, db, tenantId, 1002357, 2040000, 0)
	seedCommodity(t, db, tenantId, 9999999, 9200000, 100)
	seedCommodity(t, db, otherTenantId, 1002357, 9200000, 99999)

	server := httptest.NewServer(setupRouter(db))
	defer server.Close()

	t.Run("ReturnsMatchingRows", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/1002357", server.URL)
		req := requestWithTenant("GET", url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

		data := response["data"].([]interface{})
		assert.Len(t, data, 2)
		for _, item := range data {
			entry := item.(map[string]interface{})
			assert.Equal(t, "commodities", entry["type"])
			attrs := entry["attributes"].(map[string]interface{})
			assert.Equal(t, float64(1002357), attrs["templateId"])
		}
	})

	t.Run("EmptyArrayOnNoMatches", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/4242", server.URL)
		req := requestWithTenant("GET", url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		data := response["data"].([]interface{})
		assert.Len(t, data, 0)
	})

	t.Run("BadIdReturns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/bogus", server.URL)
		req := requestWithTenant("GET", url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("TenantIsolation", func(t *testing.T) {
		url := fmt.Sprintf("%s/commodities/items/1002357", server.URL)
		req := requestWithTenant("GET", url, otherTenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		data := response["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}

// Ensure unused imports are kept
var _ jsonapi.ServerInformation = &testServerInformation{}
