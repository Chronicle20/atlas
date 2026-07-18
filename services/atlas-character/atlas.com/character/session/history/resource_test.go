package history

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type sessionResourceTestServerInfo struct{}

func (t *sessionResourceTestServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t *sessionResourceTestServerInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &sessionResourceTestServerInfo{}

func setupSessionResourceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&sessionResourceTestServerInfo{})(db)
	ri(r, l)
	return r
}

func sessionResourceRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedSession(t *testing.T, db *gorm.DB, tenantId uuid.UUID, characterId uint32, login time.Time) {
	t.Helper()
	logout := login.Add(30 * time.Minute)
	require.NoError(t, db.Create(&entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(0),
		LoginTime:   login,
		LogoutTime:  &logout,
	}).Error)
}

// TestGetSessionsPaginates drives GET /characters/{id}/sessions through the
// real resource router (InitResource) against an in-memory tenant-scoped DB,
// verifying the JSON:API paginated envelope for the session-history growing
// log: page-size slicing, meta.total/meta.page.last, links.next/links.prev,
// and 400 on invalid paging params. Mirrors task-117 Task 9's
// TestGetAccountsPaginates / Task 10's TestGetCharactersPaginates.
func TestGetSessionsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	characterId := uint32(42)

	now := time.Now()
	seedSession(t, db, tenantId, characterId, now.Add(-3*time.Hour))
	seedSession(t, db, tenantId, characterId, now.Add(-2*time.Hour))
	seedSession(t, db, tenantId, characterId, now.Add(-1*time.Hour))

	srv := httptest.NewServer(setupSessionResourceRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/sessions?page[number]=1&page[size]=2", srv.URL, characterId)
		req := sessionResourceRequestWithTenant(http.MethodGet, url, tenantId)

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
		url := fmt.Sprintf("%s/characters/%d/sessions?page[size]=0", srv.URL, characterId)
		req := sessionResourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/sessions?limit=5", srv.URL, characterId)
		req := sessionResourceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/sessions?page[number]=99&page[size]=2", srv.URL, characterId)
		req := sessionResourceRequestWithTenant(http.MethodGet, url, tenantId)

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

// TestComputePlaytimeSinceUnaffectedByPageSize is a regression guard for the
// hidden-decoration/internal-caller lesson: ComputePlaytimeSince sums ALL
// matching sessions (not one page). With 3 sessions of 30 minutes each, and
// a drain that would (incorrectly) cap at page size 2 if GetSessionsSince
// were repointed at the paged provider, playtime would read 60m instead of
// 90m. Proves the unpaged GetSessionsSince/ComputePlaytimeSince path was not
// disturbed by adding the paged sibling.
func TestComputePlaytimeSinceUnaffectedByPageSize(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	characterId := uint32(7)

	now := time.Now()
	seedSession(t, db, tenantId, characterId, now.Add(-3*time.Hour))
	seedSession(t, db, tenantId, characterId, now.Add(-2*time.Hour))
	seedSession(t, db, tenantId, characterId, now.Add(-1*time.Hour))

	p := NewProcessor(logrus.New(), databasetest.TenantContext(tenantId), db)
	playtime, err := p.ComputePlaytimeSince(characterId, now.Add(-4*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 90*time.Minute, playtime, "must sum all 3 sessions, not just one page's worth")
}
