package _map

import (
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type pagedDoc struct {
	Data  []interface{}          `json:"data"`
	Meta  map[string]interface{} `json:"meta"`
	Links map[string]interface{} `json:"links"`
}

func decodePagedDoc(t *testing.T, resp *http.Response) pagedDoc {
	t.Helper()
	var doc pagedDoc
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	return doc
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// TestMapsBareList_PaginationEnvelope exercises GET /data/maps: page[size]=2
// over 3 seeded maps returns page 1 (2 items), total=3, and a next link.
func TestMapsBareList_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	for i, id := range []uint32{100000000, 101000000, 102000000} {
		_, err := s.Add(ctx)(RestModel{Id: _map.Id(id), Name: fmt.Sprintf("Map%d", i), StreetName: "Street"})()
		require.NoError(t, err)
	}

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestMapsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps?page[size]=0", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestMapsSearch_PaginationEnvelope proves the search arm presents the same
// envelope shape as the bare arm, now offset/limit-paged at the DB level.
func TestMapsSearch_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	for i, id := range []uint32{100000000, 101000000, 102000000} {
		_, err := s.Add(ctx)(RestModel{Id: _map.Id(id), Name: fmt.Sprintf("Victoria Map %d", i), StreetName: "Victoria Road"})()
		require.NoError(t, err)
	}

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps?search=Victoria&page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

// TestMapPortals_PaginationEnvelope covers the by-parent sub-list route
// (GetPortals) sliced from a single map document.
func TestMapPortals_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(100000000),
		Name:       "Henesys",
		StreetName: "Victoria Road",
		Portals: []portal.RestModel{
			{Id: "0", Name: "sp", TargetMapId: _map.Id(999999999)},
			{Id: "1", Name: "east00", TargetMapId: _map.Id(101000000)},
			{Id: "2", Name: "west00", TargetMapId: _map.Id(102000000)},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/portals?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestMapPortals_RejectsBadPageNumber(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	_, err := s.Add(ctx)(RestModel{Id: _map.Id(100000000), Name: "Henesys", StreetName: "Victoria Road"})()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/portals?page[number]=0", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestMapReactors_PaginationEnvelope covers the by-parent sub-list route
// (GetReactors) sliced from a single map document.
func TestMapReactors_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(100000000),
		Name:       "Henesys",
		StreetName: "Victoria Road",
		Reactors: []reactor.RestModel{
			{Id: 1, Name: "r1"},
			{Id: 2, Name: "r2"},
			{Id: 3, Name: "r3"},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/reactors?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

// TestMapNpcs_PaginationEnvelope covers the by-parent sub-list route
// (GetNpcs) sliced from a single map document.
func TestMapNpcs_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(100000000),
		Name:       "Henesys",
		StreetName: "Victoria Road",
		NPCs: []npc.RestModel{
			{Id: 1, Template: 9200000},
			{Id: 2, Template: 9200001},
			{Id: 3, Template: 9200002},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/npcs?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

// TestMapMonsters_PaginationEnvelope covers the by-parent sub-list route
// (GetMonsters) with zero spawns, proving the envelope is well-formed even
// when the underlying foothold-snap logic has nothing to snap.
func TestMapMonsters_PaginationEnvelope(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	_, err := s.Add(ctx)(RestModel{Id: _map.Id(100000000), Name: "Henesys", StreetName: "Victoria Road"})()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/monsters?page[number]=1&page[size]=10", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	doc := decodePagedDoc(t, resp)

	assert.Len(t, doc.Data, 0)
	assert.EqualValues(t, 0, doc.Meta["total"])
	assert.Nil(t, doc.Links["next"])
}
