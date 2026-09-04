package _map

import (
	"atlas-data/map/object"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type mapsServerInfo struct{}

func (mapsServerInfo) GetVersion() string { return "1.0.0" }
func (mapsServerInfo) GetURI() string     { return "/api/data/" }
func (mapsServerInfo) GetPrefix() string  { return "/api/data/" }
func (mapsServerInfo) GetBaseURL() string { return "http://localhost:8080" }

func buildMapsRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	routeInitializer := InitResource(db)(mapsServerInfo{})
	routeInitializer(router, l)
	return router
}

func mapsRequest(url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
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

func TestMapsSearch_ValidationRejectsEmptyQuery(t *testing.T) {
	db := setupStorageTestDB(t)
	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	tid := uuid.New()
	req := mapsRequest(ts.URL+"/data/maps?search=", tid)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestMapsSearch_ValidationAcceptsMaxLength(t *testing.T) {
	db := setupStorageTestDB(t)
	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	tid := uuid.New()
	q128 := strings.Repeat("a", 128)
	req := mapsRequest(fmt.Sprintf("%s/data/maps?search=%s", ts.URL, q128), tid)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 128-char query is accepted; a valid response returns 200 with empty data.
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMapsSearch_ValidationRejectsOverLength(t *testing.T) {
	db := setupStorageTestDB(t)
	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	tid := uuid.New()
	q129 := strings.Repeat("a", 129)
	req := mapsRequest(fmt.Sprintf("%s/data/maps?search=%s", ts.URL, q129), tid)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestMapsSearch_ValidationRejectsZeroLimit(t *testing.T) {
	db := setupStorageTestDB(t)
	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	tid := uuid.New()
	req := mapsRequest(ts.URL+"/data/maps?search=foo&limit=0", tid)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetMapObjectsEmpty(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(100000000),
		Name:       "Henesys",
		StreetName: "Victoria Road",
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	req := mapsRequest(ts.URL+"/data/maps/100000000/objects", tn.Id())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	data, ok := doc["data"].([]interface{})
	require.True(t, ok, "expected data to be an array, got %v", doc["data"])
	assert.Len(t, data, 0)
}

func TestGetMapObjectsReturnsRows(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(100000000),
		Name:       "Henesys",
		StreetName: "Victoria Road",
		Objects: []object.RestModel{
			{
				Id:           "ENVIRONMENT:gate",
				Kind:         "ENVIRONMENT",
				Name:         "gate",
				ObjectSource: "effect",
				L0:           "quest",
				L1:           "gate",
				L2:           "1",
				X:            640,
				Y:            120,
				Z:            0,
				Layer:        3,
			},
			{
				Id:           "OBSTACLE:menhir0",
				Kind:         "OBSTACLE",
				Name:         "menhir0",
				ObjectSource: "trapGL",
				L0:           "ckPQ",
				L1:           "menhir",
				L2:           "0",
				X:            -30,
				Y:            45,
				Z:            7,
				Layer:        2,
			},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	req := mapsRequest(ts.URL+"/data/maps/100000000/objects", tn.Id())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	data, ok := doc["data"].([]interface{})
	require.True(t, ok, "expected data to be an array, got %v", doc["data"])
	require.Len(t, data, 2)

	foundIds := make(map[string]bool)
	var gateAttrs map[string]interface{}
	for _, raw := range data {
		item, ok := raw.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "map-objects", item["type"])
		id, _ := item["id"].(string)
		foundIds[id] = true
		if id == "ENVIRONMENT:gate" {
			gateAttrs, _ = item["attributes"].(map[string]interface{})
		}
	}
	assert.True(t, foundIds["ENVIRONMENT:gate"])
	assert.True(t, foundIds["OBSTACLE:menhir0"])

	require.NotNil(t, gateAttrs)
	assert.Equal(t, map[string]interface{}{
		"kind":         "ENVIRONMENT",
		"name":         "gate",
		"objectSource": "effect",
		"l0":           "quest",
		"l1":           "gate",
		"l2":           "1",
		"x":            float64(640),
		"y":            float64(120),
		"z":            float64(0),
		"layer":        float64(3),
	}, gateAttrs)
}

func TestGetMapObjectsUnknownMap(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	req := mapsRequest(ts.URL+"/data/maps/999999999/objects", tn.Id())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
