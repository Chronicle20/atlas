package _map

import (
	"atlas-maps/map/character"
	"context"
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

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type mapTestServerInformation struct{}

func (t *mapTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *mapTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &mapTestServerInformation{}

func setupMapRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&mapTestServerInformation{})(r, l)
	return r
}

func mapRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetCharactersInMapPaginates drives GET
// /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters
// through the real resource router against the in-memory character registry
// (no DB table at all), verifying the JSON:API paginated envelope AND that
// the stable-sort-by-character-id fix is load-bearing: characters are
// entered out of id order (300, then 100, then 200), so without the sort
// page 1 would return them in registry-append order, not ascending id order.
func TestGetCharactersInMapPaginates(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	for _, characterId := range []uint32{300, 100, 200} {
		cp.Enter(uuid.New(), f, characterId)
	}

	srv := httptest.NewServer(setupMapRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/characters?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := mapRequestWithTenant(http.MethodGet, url, tenantId)

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

		assert.Equal(t, "100", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/characters?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := mapRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/characters?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := mapRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetCharactersInMapAllInstancesPaginates covers the sibling
// all-instances arm (map/resource.go handleGetCharactersInMapAllInstances),
// which fans out over every instance key in the registry via Go's
// intrinsically-randomized map iteration (character/registry.go
// GetInMapAllInstances) — the stable sort is the ONLY thing standing between
// this endpoint and a flaky page order.
func TestGetCharactersInMapAllInstancesPaginates(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(2)
	channelId := channel.Id(2)
	mapId := _map.Id(200000000)

	cp := character.NewProcessor(logrus.New(), ctx)
	// Spread across two distinct instances of the same map so
	// GetInMapAllInstances' map-iteration fan-out is actually exercised.
	instanceA := uuid.New()
	instanceB := uuid.New()
	fA := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceA).Build()
	fB := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceB).Build()
	cp.Enter(uuid.New(), fA, 300)
	cp.Enter(uuid.New(), fB, 100)
	cp.Enter(uuid.New(), fA, 200)

	srv := httptest.NewServer(setupMapRouter())
	defer srv.Close()

	url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/characters?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId)
	req := mapRequestWithTenant(http.MethodGet, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 2)
	assert.Equal(t, "100", doc.Data.DataArray[0].ID)
	assert.Equal(t, "200", doc.Data.DataArray[1].ID)

	require.NotNil(t, doc.Meta)
	assert.EqualValues(t, 3, doc.Meta["total"])
}
