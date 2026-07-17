package list

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-buddies/buddy"

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

// listsMigrationSqlite creates the lists table directly (list.Migration uses
// PostgreSQL-specific uuid_generate_v4() and cannot run on sqlite), matching
// the pattern in provider_test.go.
func listsMigrationSqlite(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS lists (
		tenant_id TEXT NOT NULL,
		id TEXT PRIMARY KEY,
		character_id INTEGER NOT NULL,
		capacity INTEGER NOT NULL
	)`).Error
}

func setupListRouter(db *gorm.DB) *mux.Router {
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

func seedBuddyList(t *testing.T, db *gorm.DB, tenantId uuid.UUID, listId uuid.UUID, characterId uint32, capacity byte) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{Id: listId, TenantId: tenantId, CharacterId: characterId, Capacity: capacity}).Error)
}

func seedBuddy(t *testing.T, db *gorm.DB, tenantId uuid.UUID, listId uuid.UUID, characterId uint32, name string) {
	t.Helper()
	require.NoError(t, db.Create(&buddy.Entity{
		CharacterId:   characterId,
		ListId:        listId,
		TenantId:      tenantId,
		Group:         "Default Group",
		CharacterName: name,
		ChannelId:     -1,
	}).Error)
}

// TestGetBuddiesInBuddyListPaginates drives GET
// /characters/{characterId}/buddy-list/buddies through the real resource
// router, verifying the JSON:API paginated envelope, 400 on invalid paging
// params, empty-page handling past the last page, and that ordering is
// deterministic (buddies sorted by CharacterId) across repeated calls since
// GORM's Preload has no explicit ORDER BY.
func TestGetBuddiesInBuddyListPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, listsMigrationSqlite, buddy.Migration)
	tenantId := uuid.New()
	listId := uuid.New()
	seedBuddyList(t, db, tenantId, listId, 42, 100)
	seedBuddy(t, db, tenantId, listId, 3, "charlie")
	seedBuddy(t, db, tenantId, listId, 1, "alpha")
	seedBuddy(t, db, tenantId, listId, 2, "bravo")

	srv := httptest.NewServer(setupListRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/buddy-list/buddies?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)
		// Deterministic order: CharacterId 1 then 2 (not insertion order 3,1,2).
		assert.Equal(t, "1", doc.Data.DataArray[0].ID)
		assert.Equal(t, "2", doc.Data.DataArray[1].ID)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/buddy-list/buddies?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/buddy-list/buddies?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/42/buddy-list/buddies?page[number]=99&page[size]=2", srv.URL)
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
