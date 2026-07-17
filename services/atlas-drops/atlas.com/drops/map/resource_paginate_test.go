package _map

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-drops/drop"

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

func setupDropsRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	drop.InitRegistry(rc)
}

type dropsTestServerInformation struct{}

func (t *dropsTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *dropsTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &dropsTestServerInformation{}

func setupDropsRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&dropsTestServerInformation{})(r, l)
	return r
}

func dropsRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetDropsInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/drops through the real
// resource router against the Redis-backed drop registry, verifying the
// JSON:API paginated envelope AND that page 1's items come back in
// ascending drop-id order (the stable-sort-by-id fix) rather than the
// registry's unordered Redis SET membership order.
func TestGetDropsInMapPaginates(t *testing.T) {
	setupDropsRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	reg := drop.GetRegistry()
	var ids []uint32
	for i := 0; i < 3; i++ {
		mb := drop.NewModelBuilder(ten, f).
			SetItem(1000000, 1).
			SetPosition(100, 200).
			SetOwner(12345, 0).
			SetDropper(99999, 50, 150).
			SetType(0)
		m, err := reg.CreateDrop(mb)
		require.NoError(t, err)
		ids = append(ids, m.Id())
	}

	srv := httptest.NewServer(setupDropsRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/drops?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := dropsRequestWithTenant(http.MethodGet, url, tenantId)

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

		assert.Equal(t, fmt.Sprintf("%d", ids[0]), doc.Data.DataArray[0].ID)
		assert.Equal(t, fmt.Sprintf("%d", ids[1]), doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/drops?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := dropsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/drops?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := dropsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
