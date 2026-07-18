package instance

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
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type instanceTestServerInformation struct{}

func (t *instanceTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *instanceTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &instanceTestServerInformation{}

func setupInstanceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&instanceTestServerInformation{})(db)(r, l)
	return r
}

func instanceRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedInstance(t *testing.T, ten tenant.Model, id uuid.UUID) {
	t.Helper()
	now := time.Now()
	m := Model{
		id:           id,
		tenantId:     ten.Id(),
		definitionId: uuid.New(),
		questId:      "test-quest",
		state:        StateActive,
		worldId:      world.Id(1),
		channelId:    channel.Id(1),
		startedAt:    now,
		registeredAt: now,
	}
	GetRegistry().Create(ten, m)
}

// TestGetAllInstancesPaginates drives GET /party-quests/instances through
// the real resource router against the in-memory instance registry (a Go
// map, whose iteration order is explicitly randomized at runtime),
// verifying the JSON:API paginated envelope AND that page 1's items come
// back in ascending instance-id-string order (the stable-sort fix) rather
// than map iteration order.
func TestGetAllInstancesPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Deliberately fixed, out-of-lexicographic-order UUIDs so the sort is
	// the only thing that can put them in order.
	idHigh := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	idLow := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	idMid := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	seedInstance(t, ten, idHigh)
	seedInstance(t, ten, idLow)
	seedInstance(t, ten, idMid)

	srv := httptest.NewServer(setupInstanceRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/party-quests/instances?page[number]=1&page[size]=2", srv.URL)
		req := instanceRequestWithTenant(http.MethodGet, url, tenantId)

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

		// seeded high, low, mid: page 1 must return low then mid in
		// ascending lexicographic order, not seed/map-iteration order.
		assert.Equal(t, idLow.String(), doc.Data.DataArray[0].ID)
		assert.Equal(t, idMid.String(), doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/party-quests/instances?page[size]=0", srv.URL)
		req := instanceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/party-quests/instances?limit=5", srv.URL)
		req := instanceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/party-quests/instances?page[number]=99&page[size]=2", srv.URL)
		req := instanceRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		assert.NotContains(t, doc.Links, "next")
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
	})
}
