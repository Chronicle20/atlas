package chair

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-chairs/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

func setupChairsRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	character.InitRegistry(rc)
}

type chairsTestServerInformation struct{}

func (t *chairsTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *chairsTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &chairsTestServerInformation{}

func setupChairsRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&chairsTestServerInformation{})(r, l)
	return r
}

func chairsRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetChairsInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/chairs through the real
// resource router against the Redis-backed character-in-map and chair
// registries, verifying the JSON:API paginated envelope AND that page 1's
// items come back in ascending characterId order (the stable-sort fix)
// rather than the registry's unordered Redis SET membership order.
func TestGetChairsInMapPaginates(t *testing.T) {
	setupChairsRegistries(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	for _, characterId := range []uint32{300, 100, 200} {
		cp.Enter(f, characterId)
		GetRegistry().Set(ctx, characterId, Model{})
	}
	// A character present in the map but with no active chair must NOT
	// appear in the list (the existing GetById-error filter, unchanged).
	cp.Enter(f, 400)

	srv := httptest.NewServer(setupChairsRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chairs?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := chairsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "meta.total must reflect the filtered count (3), not the 4 characters present in the map")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")

		var first, second struct {
			CharacterId uint32 `json:"characterId"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, 100, first.CharacterId)
		assert.EqualValues(t, 200, second.CharacterId)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chairs?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := chairsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chairs?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := chairsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
