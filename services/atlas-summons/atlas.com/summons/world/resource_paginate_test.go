package world

import (
	"atlas-summons/summon"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupSummonsRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	summon.InitRegistry(rc)
}

type summonsTestServerInformation struct{}

func (t *summonsTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *summonsTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &summonsTestServerInformation{}

func setupSummonsRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&summonsTestServerInformation{})(r, l)
	return r
}

func summonsRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetSummonsInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/summons through the real
// resource router against the Redis-backed summon registry, verifying the
// JSON:API paginated envelope AND that page 1's items come back in
// ascending summon-id order (the stable-sort-by-id fix) rather than the
// registry's unordered Redis SET membership order.
func TestGetSummonsInMapPaginates(t *testing.T) {
	setupSummonsRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	now := time.Now()
	for _, id := range []uint32{300, 100, 200} {
		m := summon.NewBuilder().
			SetId(id).
			SetOwnerCharacterId(42).
			SetSkillId(3111002).
			SetSkillLevel(20).
			SetSummonType(summon.SummonTypePuppet).
			SetMovementType(summon.MovementStationary).
			SetField(f).
			SetX(100).SetY(-50).
			SetHp(800).SetMaxHp(800).
			SetSpawnTime(now).
			SetExpiresAt(now.Add(60 * time.Second)).
			Build()
		require.NoError(t, summon.GetRegistry().Put(ctx, ten, m))
	}

	srv := httptest.NewServer(setupSummonsRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/summons?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := summonsRequestWithTenant(http.MethodGet, url, tenantId)

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

		// seeded out of ascending-id order (300, 100, 200): page 1 must
		// return 100 then 200, not registry-insertion order.
		assert.Equal(t, "100", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/summons?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := summonsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/summons?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := summonsRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
