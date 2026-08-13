package global

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupGlobalRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	return requestWithTenantBody(method, url, tenantId, nil)
}

func requestWithTenantBody(method, url string, tenantId uuid.UUID, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
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

func seedGlobalItem(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, itemId uint32, tier string) {
	t.Helper()
	m, err := NewBuilder(tenantId, id).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier(tier).
		Build()
	require.NoError(t, err)
	require.NoError(t, CreateItem(db, m))
}

// TestCreateGlobalItemWithoutClientSuppliedId drives POST /global-items with
// the UI's JSON:API creation payload, which carries no `data.id` — the row id
// is server-generated. api2go calls SetID("") for it, which must not fail the
// request. Mirrors item.TestCreateItemWithoutClientSuppliedId.
func TestCreateGlobalItemWithoutClientSuppliedId(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupGlobalRouter(db))
	defer srv.Close()

	body := `{"data":{"type":"global-gachapon-items","attributes":{"itemId":2000000,"quantity":1,"tier":"rare"}}}`
	req := requestWithTenantBody(http.MethodPost, srv.URL+"/global-items", tenantId, strings.NewReader(body))

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Read it back through the router so the assertion covers the persisted
	// row, not just the status code.
	getResp, err := (&http.Client{}).Do(requestWithTenant(http.MethodGet, srv.URL+"/global-items", tenantId))
	require.NoError(t, err)
	defer func() { _ = getResp.Body.Close() }()

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 1)

	var attrs JSONModel
	require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &attrs))
	assert.EqualValues(t, 2000000, attrs.ItemId)
	assert.Equal(t, "rare", attrs.Tier)
}

// TestSetIDAcceptsEmptyId pins the unmarshal-level contract directly.
func TestSetIDAcceptsEmptyId(t *testing.T) {
	var rm RestModel
	require.NoError(t, rm.SetID(""))
	assert.EqualValues(t, 0, rm.Id)

	require.NoError(t, rm.SetID("7"))
	assert.EqualValues(t, 7, rm.Id)

	assert.Error(t, rm.SetID("not-a-number"), "a genuinely malformed id must still be rejected")
}

// TestGetAllGlobalItemsPaginates drives GET /global-items (bare, no tier
// filter) through the real resource router (InitResource) against an
// in-memory tenant-scoped DB, verifying the JSON:API paginated envelope,
// 400 on invalid paging params, and past-end-page handling.
func TestGetAllGlobalItemsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedGlobalItem(t, db, tenantId, 1, 2000000, "common")
	seedGlobalItem(t, db, tenantId, 2, 2000001, "uncommon")
	seedGlobalItem(t, db, tenantId, 3, 2000002, "rare")

	srv := httptest.NewServer(setupGlobalRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/global-items?page[number]=1&page[size]=2", srv.URL)
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
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/global-items?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/global-items?page[number]=99&page[size]=2", srv.URL)
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
		assert.NotContains(t, doc.Links, "next")
	})
}

// TestGetGlobalItemsByTierPaginates drives GET /global-items?tier=..., the
// filtered arm backed by GetByTierPaged, verifying the pagination envelope
// excludes other tiers from the total.
func TestGetGlobalItemsByTierPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedGlobalItem(t, db, tenantId, 1, 2000000, "common")
	seedGlobalItem(t, db, tenantId, 2, 2000001, "common")
	seedGlobalItem(t, db, tenantId, 3, 2000002, "rare")

	srv := httptest.NewServer(setupGlobalRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/global-items?tier=common&page[number]=1&page[size]=2", srv.URL)
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
		assert.EqualValues(t, 2, doc.Meta["total"], "must exclude the rare-tier row")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/global-items?tier=common&page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
