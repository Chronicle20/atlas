package guild

import (
	"atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
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

// resourceTestDB mirrors newGuildsDB's sqlite workaround (title.Migration uses
// PostgreSQL-specific uuid_generate_v4()) but seeds nothing, leaving that to
// each test.
func resourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	return databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, character.Migration, titlesMigration)
}

func setupGuildRouter(db *gorm.DB) *mux.Router {
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

func seedGuild(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, name string, leaderId uint32) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{Id: id, TenantId: tenantId, WorldId: 0, Name: name, LeaderId: leaderId, Capacity: 30}).Error)
}

// TestGetGuildsResourcePaginates drives GET /guilds through the real
// resource router (InitResource) against an in-memory tenant-scoped DB,
// verifying the JSON:API paginated envelope on the bare list form.
func TestGetGuildsResourcePaginates(t *testing.T) {
	db := resourceTestDB(t)
	tenantId := uuid.New()
	seedGuild(t, db, tenantId, 1, "GuildOne", 100)
	seedGuild(t, db, tenantId, 2, "GuildTwo", 101)
	seedGuild(t, db, tenantId, 3, "GuildThree", 102)

	srv := httptest.NewServer(setupGuildRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?page[number]=1&page[size]=2", srv.URL)
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

	t.Run("BadPageSizeIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetGuildsByNameFilterResource exercises the filter[name] route,
// including the mux-order-dependent empty-value case: gorilla/mux compiles
// query templates with the ".*" default pattern (unlike ".+" for path
// segments), so "filter[name]=" (present, empty) DOES match this route
// rather than falling through to the bare form — the handler's own
// name == "" check is what turns it into a 400.
func TestGetGuildsByNameFilterResource(t *testing.T) {
	db := resourceTestDB(t)
	tenantId := uuid.New()
	seedGuild(t, db, tenantId, 1, "Alpha", 100)
	seedGuild(t, db, tenantId, 2, "alphabet", 101)
	seedGuild(t, db, tenantId, 3, "Beta", 102)

	srv := httptest.NewServer(setupGuildRouter(db))
	defer srv.Close()

	t.Run("MatchesSubstringCaseInsensitive", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?filter[name]=alpha", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)
		assert.EqualValues(t, 2, doc.Meta["total"])
	})

	t.Run("ComposesWithPaging", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?filter[name]=alpha&page[number]=1&page[size]=1", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)
		assert.EqualValues(t, 2, doc.Meta["total"])
	})

	t.Run("EmptyValueIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?filter[name]=", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetGuildsByMemberIdFilterResource verifies the ?filter[members.id]=
// form keeps its prior shape (single guild for the member) and additionally
// accepts page[*].
func TestGetGuildsByMemberIdFilterResource(t *testing.T) {
	db := resourceTestDB(t)
	tenantId := uuid.New()
	seedGuild(t, db, tenantId, 1, "GuildOne", 100)
	require.NoError(t, db.Create(&member.Entity{TenantId: tenantId, GuildId: 1, CharacterId: 500, Name: "hero", Level: 10}).Error)
	require.NoError(t, db.Create(&character.Entity{TenantId: tenantId, CharacterId: 500, GuildId: 1}).Error)

	srv := httptest.NewServer(setupGuildRouter(db))
	defer srv.Close()

	t.Run("ReturnsGuildForMember", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?filter[members.id]=500", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)
	})

	t.Run("InvalidMemberIdIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds?filter[members.id]=bogus", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
