package item

import (
	"context"
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
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupItemRouter(db *gorm.DB) *mux.Router {
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

func seedGachaponItem(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, gachaponId string, itemId uint32, tier string) {
	t.Helper()
	m, err := NewBuilder(tenantId, id).
		SetGachaponId(gachaponId).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier(tier).
		Build()
	require.NoError(t, err)
	require.NoError(t, CreateItem(db, m))
}

// TestCreateItemWithoutClientSuppliedId drives
// POST /gachapons/{gachaponId}/items with the exact JSON:API creation payload
// the UI sends: `data` carries a type and attributes but NO `id`, because the
// row's id is server-generated. api2go's Unmarshal calls SetID(data.ID)
// unconditionally, so an id-less create reaches SetID with "" — which must not
// be treated as a malformed id.
func TestCreateItemWithoutClientSuppliedId(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupItemRouter(db))
	defer srv.Close()

	body := `{"data":{"type":"gachapon-items","attributes":{"itemId":2000000,"quantity":1,"tier":"common","weight":50}}}`
	url := fmt.Sprintf("%s/gachapons/henesys/items", srv.URL)
	req := requestWithTenantBody(http.MethodPost, url, tenantId, strings.NewReader(body))

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	items, err := NewProcessor(logrus.New(), ctx, db).GetByGachaponId("henesys")()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.EqualValues(t, 2000000, items[0].ItemId())
	assert.EqualValues(t, 50, items[0].Weight())
}

// TestSetIDAcceptsEmptyId pins the unmarshal-level contract directly: a
// creation payload has no id, and RestModel must accept that rather than
// failing the whole request.
func TestSetIDAcceptsEmptyId(t *testing.T) {
	var rm RestModel
	require.NoError(t, rm.SetID(""))
	assert.EqualValues(t, 0, rm.Id)

	require.NoError(t, rm.SetID("7"))
	assert.EqualValues(t, 7, rm.Id)

	assert.Error(t, rm.SetID("not-a-number"), "a genuinely malformed id must still be rejected")
}

// TestGetItemsByGachaponIdPaginates drives GET /gachapons/{gachaponId}/items
// (bare, no tier filter) through the real resource router, verifying the
// JSON:API paginated envelope, 400 on invalid paging params, and
// tenant/gachaponId scoping of meta.total.
func TestGetItemsByGachaponIdPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedGachaponItem(t, db, tenantId, 1, "henesys", 2000000, "common")
	seedGachaponItem(t, db, tenantId, 2, "henesys", 2000001, "uncommon")
	seedGachaponItem(t, db, tenantId, 3, "henesys", 2000002, "rare")
	seedGachaponItem(t, db, tenantId, 4, "ellinia", 2000003, "common")

	srv := httptest.NewServer(setupItemRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/items?page[number]=1&page[size]=2", srv.URL)
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
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude the other gachaponId's item")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/items?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetItemsByGachaponIdAndTierPaginates drives
// GET /gachapons/{gachaponId}/items?tier=..., verifying meta.total excludes
// other tiers.
func TestGetItemsByGachaponIdAndTierPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedGachaponItem(t, db, tenantId, 1, "henesys", 2000000, "common")
	seedGachaponItem(t, db, tenantId, 2, "henesys", 2000001, "common")
	seedGachaponItem(t, db, tenantId, 3, "henesys", 2000002, "rare")

	srv := httptest.NewServer(setupItemRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/items?tier=common&page[number]=1&page[size]=2", srv.URL)
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
}
