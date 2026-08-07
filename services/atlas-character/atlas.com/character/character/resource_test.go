package character

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type resourceTestServerInfo struct{}

func (t *resourceTestServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t *resourceTestServerInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &resourceTestServerInfo{}

func setupResourceTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitTemporalRegistry(client)
}

func setupCharacterResourceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&resourceTestServerInfo{})(db)
	ri(r, l)
	return r
}

func resourceRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedResourceCharacter(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, accountId uint32, worldId world.Id, name string) {
	t.Helper()
	require.NoError(t, db.Create(&entity{ID: id, TenantId: tenantId, AccountId: accountId, World: worldId, Name: name, Level: 1, JobId: 0}).Error)
}

// TestGetCharactersPaginates drives GET /characters through the real
// resource router (InitResource) against an in-memory tenant-scoped DB,
// verifying the JSON:API paginated envelope: page-size slicing,
// meta.total/meta.page.last, links.next/links.prev, and 400 on invalid
// paging params. Mirrors task-117 Task 9's TestGetAccountsPaginates.
func TestGetCharactersPaginates(t *testing.T) {
	setupResourceTestRegistry(t)

	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedResourceCharacter(t, db, tenantId, 1, 100, world.Id(0), "hero1")
	seedResourceCharacter(t, db, tenantId, 2, 100, world.Id(0), "hero2")
	seedResourceCharacter(t, db, tenantId, 3, 100, world.Id(0), "hero3")

	srv := httptest.NewServer(setupCharacterResourceRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters?page[number]=1&page[size]=2", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

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
		url := fmt.Sprintf("%s/characters?page[size]=0", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters?limit=5", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters?page[number]=99&page[size]=2", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

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

// TestGetCharactersByNamePaginates drives GET /characters?name=X through the
// real resource router, verifying the name filter is preserved (only
// matching characters returned) while page[*] params are still honored.
func TestGetCharactersByNamePaginates(t *testing.T) {
	setupResourceTestRegistry(t)

	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	// Same name across different worlds — getForName filters by name only
	// (not world), so this exercises multiple matches for one name filter.
	seedResourceCharacter(t, db, tenantId, 1, 100, world.Id(0), "Hero")
	seedResourceCharacter(t, db, tenantId, 2, 200, world.Id(1), "Hero")
	seedResourceCharacter(t, db, tenantId, 3, 300, world.Id(2), "Hero")
	seedResourceCharacter(t, db, tenantId, 4, 400, world.Id(0), "OtherName")

	srv := httptest.NewServer(setupCharacterResourceRouter(db))
	defer srv.Close()

	t.Run("FiltersByNameAndPaginates", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters?name=Hero&page[number]=1&page[size]=2", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		// Only the 3 "Hero" rows match the name filter; "OtherName" is excluded.
		assert.Len(t, doc.Data.DataArray, 2, "page size 2 of 3 matching rows")

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "name filter must exclude OtherName")

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters?name=Hero&page[size]=0", srv.URL)
		req := resourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
