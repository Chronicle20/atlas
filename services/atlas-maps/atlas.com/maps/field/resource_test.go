package field

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
	cfield "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fieldTestServerInformation struct{}

func (t *fieldTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *fieldTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &fieldTestServerInformation{}

func setupFieldRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&fieldTestServerInformation{})(r, l)
	return r
}

func fieldRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func doGetFields(t *testing.T, tenantId uuid.UUID, url string) (*http.Response, jsonapi.Document) {
	t.Helper()
	req := fieldRequestWithTenant(http.MethodGet, url, tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	return resp, doc
}

// TestGetFieldsDrainedFieldExcluded is the PRD's named acceptance test: a
// field that has been fully drained (entered then exited) must not appear
// in the listing, even though its registry key lingers as a zero-length
// entry.
func TestGetFieldsDrainedFieldExcluded(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(0)
	channelId := channel.Id(1)

	drainedMap := _map.Id(100000000)
	liveMap := _map.Id(200000000)

	drainedField := cfield.NewBuilder(worldId, channelId, drainedMap).SetInstance(uuid.Nil).Build()
	liveField := cfield.NewBuilder(worldId, channelId, liveMap).SetInstance(uuid.Nil).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	cp.Enter(uuid.New(), drainedField, 100)
	cp.Exit(uuid.New(), drainedField, 100)
	cp.Enter(uuid.New(), liveField, 200)

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields", srv.URL))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 1)

	assert.Equal(t, "0:1:200000000:00000000-0000-0000-0000-000000000000", doc.Data.DataArray[0].ID)
	for _, item := range doc.Data.DataArray {
		assert.NotContains(t, item.ID, "0:1:100000000:")
	}
}

func TestGetFieldsTenantIsolation(t *testing.T) {
	tenantAId := uuid.New()
	tenA, err := tenant.Create(tenantAId, "GMS", 83, 1)
	require.NoError(t, err)
	ctxA := tenant.WithContext(context.Background(), tenA)

	tenantBId := uuid.New()
	tenB, err := tenant.Create(tenantBId, "GMS", 83, 1)
	require.NoError(t, err)
	ctxB := tenant.WithContext(context.Background(), tenB)

	worldId := world.Id(0)
	channelId := channel.Id(1)
	mapA := _map.Id(300000000)
	mapB := _map.Id(400000000)

	fA := cfield.NewBuilder(worldId, channelId, mapA).SetInstance(uuid.Nil).Build()
	fB := cfield.NewBuilder(worldId, channelId, mapB).SetInstance(uuid.Nil).Build()

	character.NewProcessor(logrus.New(), ctxA).Enter(uuid.New(), fA, 1)
	character.NewProcessor(logrus.New(), ctxB).Enter(uuid.New(), fB, 2)

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	respA, docA := doGetFields(t, tenantAId, fmt.Sprintf("%s/fields", srv.URL))
	require.Equal(t, http.StatusOK, respA.StatusCode)
	require.NotNil(t, docA.Data)
	require.Len(t, docA.Data.DataArray, 1)
	var attrsA map[string]interface{}
	require.NoError(t, json.Unmarshal(docA.Data.DataArray[0].Attributes, &attrsA))
	assert.EqualValues(t, 300000000, attrsA["mapId"])

	respB, docB := doGetFields(t, tenantBId, fmt.Sprintf("%s/fields", srv.URL))
	require.Equal(t, http.StatusOK, respB.StatusCode)
	require.NotNil(t, docB.Data)
	require.Len(t, docB.Data.DataArray, 1)
	var attrsB map[string]interface{}
	require.NoError(t, json.Unmarshal(docB.Data.DataArray[0].Attributes, &attrsB))
	assert.EqualValues(t, 400000000, attrsB["mapId"])
}

func TestGetFieldsFilters(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f1 := cfield.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f2 := cfield.NewBuilder(world.Id(0), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f3 := cfield.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(uuid.Nil).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	cp.Enter(uuid.New(), f1, 1)
	cp.Enter(uuid.New(), f2, 2)
	cp.Enter(uuid.New(), f3, 3)

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	t.Run("no filter", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 3)
	})

	t.Run("world only", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[worldId]=0", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 2)
		for _, item := range doc.Data.DataArray {
			var attrs map[string]interface{}
			require.NoError(t, json.Unmarshal(item.Attributes, &attrs))
			assert.EqualValues(t, 100000000, attrs["mapId"])
		}
	})

	t.Run("channel only", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[channelId]=1", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 2)
		ids := []string{doc.Data.DataArray[0].ID, doc.Data.DataArray[1].ID}
		assert.Contains(t, ids, "0:1:100000000:00000000-0000-0000-0000-000000000000")
		assert.Contains(t, ids, "1:1:200000000:00000000-0000-0000-0000-000000000000")
	})

	t.Run("map only", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[mapId]=200000000", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 1)
		assert.Equal(t, "1:1:200000000:00000000-0000-0000-0000-000000000000", doc.Data.DataArray[0].ID)
	})

	t.Run("all three", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[worldId]=0&filter[channelId]=2&filter[mapId]=100000000", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 1)
		assert.Equal(t, "0:2:100000000:00000000-0000-0000-0000-000000000000", doc.Data.DataArray[0].ID)
	})

	t.Run("unknown filter ignored", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[nope]=7", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Len(t, doc.Data.DataArray, 3)
	})

	t.Run("no match", func(t *testing.T) {
		resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?filter[worldId]=9", srv.URL))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 0)
	})
}

func TestGetFieldsMalformedFilter(t *testing.T) {
	tenantId := uuid.New()
	_, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	tests := []struct {
		name  string
		query string
	}{
		{"worldId", "?filter[worldId]=abc"},
		{"channelId", "?filter[channelId]=abc"},
		{"mapId", "?filter[mapId]=abc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := fieldRequestWithTenant(http.MethodGet, fmt.Sprintf("%s/fields%s", srv.URL, tc.query), tenantId)
			resp, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestGetFieldsPaginationDeterminism(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(0)
	channelId := channel.Id(1)

	cp := character.NewProcessor(logrus.New(), ctx)
	shuffledMapIds := []uint32{400000000, 100000000, 600000000, 200000000, 500000000, 300000000}
	for i, mapId := range shuffledMapIds {
		f := cfield.NewBuilder(worldId, channelId, _map.Id(mapId)).SetInstance(uuid.Nil).Build()
		cp.Enter(uuid.New(), f, uint32(i+1))
	}

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	expectedPage1 := []string{
		"0:1:100000000:00000000-0000-0000-0000-000000000000",
		"0:1:200000000:00000000-0000-0000-0000-000000000000",
		"0:1:300000000:00000000-0000-0000-0000-000000000000",
	}
	expectedPage2 := []string{
		"0:1:400000000:00000000-0000-0000-0000-000000000000",
		"0:1:500000000:00000000-0000-0000-0000-000000000000",
		"0:1:600000000:00000000-0000-0000-0000-000000000000",
	}

	for i := 0; i < 5; i++ {
		resp1, doc1 := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?page[number]=1&page[size]=3", srv.URL))
		require.Equal(t, http.StatusOK, resp1.StatusCode)
		require.Len(t, doc1.Data.DataArray, 3)
		var got1 []string
		for _, item := range doc1.Data.DataArray {
			got1 = append(got1, item.ID)
		}
		assert.Equal(t, expectedPage1, got1)

		resp2, doc2 := doGetFields(t, tenantId, fmt.Sprintf("%s/fields?page[number]=2&page[size]=3", srv.URL))
		require.Equal(t, http.StatusOK, resp2.StatusCode)
		require.Len(t, doc2.Data.DataArray, 3)
		var got2 []string
		for _, item := range doc2.Data.DataArray {
			got2 = append(got2, item.ID)
		}
		assert.Equal(t, expectedPage2, got2)

		union := make(map[string]bool)
		for _, id := range got1 {
			union[id] = true
		}
		for _, id := range got2 {
			require.False(t, union[id], "duplicate id %s across pages", id)
			union[id] = true
		}
		assert.Len(t, union, 6)
	}
}

func TestGetFieldsAttributes(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(0)
	channelId := channel.Id(1)
	mapId := _map.Id(910340000)
	f := cfield.NewBuilder(worldId, channelId, mapId).SetInstance(uuid.Nil).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	cp.Enter(uuid.New(), f, 1)
	cp.Enter(uuid.New(), f, 2)
	cp.Enter(uuid.New(), f, 3)

	srv := httptest.NewServer(setupFieldRouter())
	defer srv.Close()

	resp, doc := doGetFields(t, tenantId, fmt.Sprintf("%s/fields", srv.URL))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, doc.Data.DataArray, 1)

	item := doc.Data.DataArray[0]
	assert.Equal(t, "fields", item.Type)

	var attrs map[string]interface{}
	require.NoError(t, json.Unmarshal(item.Attributes, &attrs))
	assert.EqualValues(t, 0, attrs["worldId"])
	assert.EqualValues(t, 1, attrs["channelId"])
	assert.EqualValues(t, 910340000, attrs["mapId"])
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", attrs["instanceId"])
	assert.EqualValues(t, 3, attrs["characterCount"])
}
