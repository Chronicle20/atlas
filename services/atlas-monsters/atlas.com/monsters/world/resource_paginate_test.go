package world

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"atlas-monsters/monster"

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

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	monster.InitIdAllocator(rc)
	monster.InitMonsterRegistry(rc)

	os.Exit(m.Run())
}

type worldTestServerInformation struct{}

func (t *worldTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *worldTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &worldTestServerInformation{}

func setupWorldRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&worldTestServerInformation{})(r, l)
	return r
}

func worldRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetMonstersInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters through the real
// resource router against the Redis-backed monster registry, verifying the
// JSON:API paginated envelope and that page 1's items come back in
// ascending unique-id order (the stable-sort-by-unique-id fix).
func TestGetMonstersInMapPaginates(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	reg := monster.GetMonsterRegistry()
	var ids []uint32
	for i := 0; i < 3; i++ {
		m := reg.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 0, 0, 100, 100)
		ids = append(ids, m.UniqueId())
	}

	srv := httptest.NewServer(setupWorldRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/monsters?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := worldRequestWithTenant(http.MethodGet, url, tenantId)

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

		firstId, err := strconv.ParseUint(doc.Data.DataArray[0].ID, 10, 32)
		require.NoError(t, err)
		secondId, err := strconv.ParseUint(doc.Data.DataArray[1].ID, 10, 32)
		require.NoError(t, err)
		assert.Less(t, firstId, secondId, "page 1 items must be in ascending unique-id order")
		assert.Contains(t, ids, uint32(firstId))
		assert.Contains(t, ids, uint32(secondId))
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/monsters?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := worldRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/monsters?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := worldRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetMonstersInMapRectPaginates proves the /monsters/in-rect arm keeps
// its ascending-distance-from-center ordering (server-authoritative
// closest-first target selection) intact under pagination -- the paginated
// envelope wraps GetInFieldRect's existing sort, it does not replace it with
// a stable sort by unique id.
func TestGetMonstersInMapRectPaginates(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(2)
	channelId := channel.Id(2)
	mapId := mapconst.Id(200000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	reg := monster.GetMonsterRegistry()
	far := reg.CreateMonster(ctx, ten, f, 9300018, 100, 100, 0, 0, 0, 100, 100)
	near := reg.CreateMonster(ctx, ten, f, 9300018, 10, 10, 0, 0, 0, 100, 100)
	mid := reg.CreateMonster(ctx, ten, f, 9300018, 50, 50, 0, 0, 0, 100, 100)

	srv := httptest.NewServer(setupWorldRouter())
	defer srv.Close()

	url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/monsters/in-rect?x1=0&y1=0&x2=100&y2=100&page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
	req := worldRequestWithTenant(http.MethodGet, url, tenantId)

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

	// Rect center is (50,50): mid(50,50) has d2=0, near(10,10) has d2=3200,
	// far(100,100) has d2=5000 -- ascending-distance order is
	// [mid, near, far], so page 1 of size 2 must return mid then near.
	assert.Equal(t, fmt.Sprintf("%d", mid.UniqueId()), doc.Data.DataArray[0].ID)
	assert.Equal(t, fmt.Sprintf("%d", near.UniqueId()), doc.Data.DataArray[1].ID)
	_ = far
}
