package character

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-buffs/buff/stat"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string   { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupBuffRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupBuffRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&testServerInformation{})(r, l)
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

// TestGetBuffsPaginates drives GET /characters/{characterId}/buffs through
// the real resource router (InitResource) against a miniredis-backed
// registry (buffs have no DB table at all — this is a live TTL cache),
// verifying the JSON:API paginated envelope AND that the sort-by-map-key
// determinism fix is load-bearing: Go map iteration order is randomized, so
// without the sort this test would flake across runs.
func TestGetBuffsPaginates(t *testing.T) {
	setupBuffRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	characterId := uint32(900)
	// srcKey(sourceId) = strconv.FormatInt(sourceId,10); with these 3 values
	// the map keys "100" < "200" < "300" sort both lexicographically and
	// numerically, so the expected page order is unambiguous. Applied in
	// reverse (300, then 100, then 200) so map insertion order can never be
	// mistaken for the sort that actually determines page contents.
	for _, sourceId := range []int32{300, 100, 200} {
		_, err := GetRegistry().Apply(ctx, 0, 0, characterId, sourceId, 1, 60000, []stat.Model{stat.NewStat("HP", 10)}, false)
		require.NoError(t, err)
	}

	srv := httptest.NewServer(setupBuffRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/900/buffs?page[number]=1&page[size]=2", srv.URL)
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

		var first, second struct {
			SourceId int32 `json:"sourceId"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, 100, first.SourceId)
		assert.EqualValues(t, 200, second.SourceId)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/900/buffs?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/900/buffs?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/900/buffs?page[number]=99&page[size]=2", srv.URL)
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
