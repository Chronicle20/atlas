package asset

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
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

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupAssetRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
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

func seedAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, slot int16, templateId uint32) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		Slot:          slot,
		TemplateId:    templateId,
		Expiration:    time.Time{},
		CreatedAt:     time.Now(),
	}).Error)
}

// TestGetAssetsPaginates drives GET /characters/{characterId}/inventory/compartments/{compartmentId}/assets
// through the real resource router, verifying the JSON:API paginated envelope, 400 on invalid
// paging params, and empty-page handling past the last page. Also confirms a different
// compartment's assets are excluded from the total (the CompartmentId filter survives the
// pagination conversion).
func TestGetAssetsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	compartmentId := uuid.New()
	otherCompartmentId := uuid.New()
	seedAsset(t, db, tenantId, compartmentId, 1, 2000000)
	seedAsset(t, db, tenantId, compartmentId, 2, 2000001)
	seedAsset(t, db, tenantId, compartmentId, 3, 2000002)
	seedAsset(t, db, tenantId, otherCompartmentId, 1, 9999999)

	srv := httptest.NewServer(setupAssetRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/inventory/compartments/%s/assets?page[number]=1&page[size]=2", srv.URL, compartmentId)
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
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude the other compartment's asset")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/inventory/compartments/%s/assets?page[size]=0", srv.URL, compartmentId)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/inventory/compartments/%s/assets?limit=5", srv.URL, compartmentId)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/inventory/compartments/%s/assets?page[number]=99&page[size]=2", srv.URL, compartmentId)
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

	t.Run("DefaultPageSizeIsGameCap250", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/inventory/compartments/%s/assets", srv.URL, compartmentId)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 3, "all 3 assets should fit on the default game-capped page")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 250, page["size"])
	})
}
