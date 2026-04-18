package _map

import (
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
