package continent

import (
	"atlas-drops-information/continent/drop"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func setupContinentRouter(db *gorm.DB) *mux.Router {
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

func seedContinentDrop(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, continentId int32, itemId uint32) {
	t.Helper()
	m, err := drop.NewContinentDropBuilder(tenantId, id).
		SetContinentId(continentId).
		SetItemId(itemId).
		SetMinimumQuantity(1).
		SetMaximumQuantity(1).
		SetChance(50000).
		Build()
	require.NoError(t, err)
	require.NoError(t, drop.BulkCreateContinentDrop(db, []drop.Model{m}))
}

// TestGetContinentsPaginates drives GET /continents/drops through the real
// resource router (InitResource) against an in-memory tenant-scoped DB.
// GetAll() is a computed in-memory aggregation (drops grouped by
// continentId via a Go map, so it has no natural order) rather than a
// single Where-filtered query; the handler stable-sorts by continent id
// before paginate.Slice, so this test also proves the sort makes paging
// deterministic across requests, not just correct on page 1.
func TestGetContinentsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, drop.Migration)
	tenantId := uuid.New()
	// Three distinct continents, each with a drop row (grouping key is
	// continentId); seeded out of numeric order to exercise the stable sort.
	seedContinentDrop(t, db, tenantId, 1, 3, 2000000)
	seedContinentDrop(t, db, tenantId, 2, 1, 2000001)
	seedContinentDrop(t, db, tenantId, 3, 2, 2000002)

	srv := httptest.NewServer(setupContinentRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoOrderedByContinentId", func(t *testing.T) {
		url := fmt.Sprintf("%s/continents/drops?page[number]=1&page[size]=2", srv.URL)
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

		// Stable sort by continent id: continents 1 then 2 must come first,
		// even though they were seeded (and map-iterated) out of order.
		assert.Equal(t, "1", doc.Data.DataArray[0].ID)
		assert.Equal(t, "2", doc.Data.DataArray[1].ID)

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("SecondPageIsContinent3", func(t *testing.T) {
		url := fmt.Sprintf("%s/continents/drops?page[number]=2&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1)
		assert.Equal(t, "3", doc.Data.DataArray[0].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/continents/drops?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
