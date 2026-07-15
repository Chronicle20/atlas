package chalkboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-chalkboards/character"

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

func setupChalkboardsRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	character.InitRegistry(rc)
}

type chalkboardsTestServerInformation struct{}

func (t *chalkboardsTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *chalkboardsTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &chalkboardsTestServerInformation{}

func setupChalkboardsRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&chalkboardsTestServerInformation{})(r, l)
	return r
}

func chalkboardsRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetChalkboardsInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/chalkboards through the
// real resource router against the Redis-backed character-in-map and
// chalkboard-message registries, verifying the JSON:API paginated envelope
// AND that page 1's items come back in ascending characterId order (the
// stable-sort fix) rather than the registry's unordered Redis SET
// membership order.
func TestGetChalkboardsInMapPaginates(t *testing.T) {
	setupChalkboardsRegistries(t)

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
		getRegistry().Set(ctx, characterId, "hello")
	}
	// A character present in the map but with no active chalkboard message
	// must NOT appear in the list (the existing GetById-error filter,
	// unchanged).
	cp.Enter(f, 400)

	srv := httptest.NewServer(setupChalkboardsRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chalkboards?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := chalkboardsRequestWithTenant(http.MethodGet, url, tenantId)

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

		assert.Equal(t, "100", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chalkboards?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := chalkboardsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/chalkboards?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := chalkboardsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
