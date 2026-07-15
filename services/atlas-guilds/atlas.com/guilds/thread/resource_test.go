package thread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atlas-guilds/thread/reply"

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

func resourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration, reply.Migration)
}

func setupThreadRouter(db *gorm.DB) *mux.Router {
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

func seedThread(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, guildId uint32, title string, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{
		Id:         id,
		TenantId:   tenantId,
		GuildId:    guildId,
		PosterId:   100,
		Title:      title,
		Message:    "message",
		EmoticonId: 0,
		Notice:     false,
		CreatedAt:  createdAt,
	}).Error)
}

// TestGetGuildThreadsResourcePaginates drives GET /guilds/{guildId}/threads
// through the real resource router (InitResource) against an in-memory
// tenant-scoped DB, verifying the JSON:API paginated envelope, that the
// route is guild-scoped, and that a bad page param is a 400.
func TestGetGuildThreadsResourcePaginates(t *testing.T) {
	db := resourceTestDB(t)
	tenantId := uuid.New()
	base := time.Now().Add(-time.Hour)
	seedThread(t, db, tenantId, 1, 1, "ThreadOne", base.Add(1*time.Minute))
	seedThread(t, db, tenantId, 2, 1, "ThreadTwo", base.Add(2*time.Minute))
	seedThread(t, db, tenantId, 3, 1, "ThreadThree", base.Add(3*time.Minute))
	// Different guild -- must not appear in guild 1's list.
	seedThread(t, db, tenantId, 4, 2, "OtherGuildThread", base.Add(4*time.Minute))

	srv := httptest.NewServer(setupThreadRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds/1/threads?page[number]=1&page[size]=2", srv.URL)
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

		// Most-recently-created thread (ThreadThree) sorts first (created_at
		// desc), matching the pre-pagination ordering the guild BBS writer
		// relies on for notice-thread placement.
		var attrs struct {
			Title string `json:"title"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &attrs))
		assert.Equal(t, "ThreadThree", attrs.Title)
	})

	t.Run("SecondPageHasRemainder", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds/1/threads?page[number]=2&page[size]=2", srv.URL)
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

	t.Run("BadPageSizeIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/guilds/1/threads?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
