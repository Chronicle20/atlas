package crystalband_test

import (
	"atlas-maker/crystalband"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupCrystalBandRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	crystalband.InitResource(&testServerInformation{})(db)(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestResourceIsReadOnly enforces the same read-only-reference-data
// requirement reagent's resource carries: a writable crystal-band table
// would let a client retune its own disassembly yield.
func TestResourceIsReadOnly(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	seedDerivedBands(t, db, tenantId)

	srv := httptest.NewServer(setupCrystalBandRouter(db))
	defer srv.Close()

	paths := []string{"/crystal-bands", "/crystal-bands/31"}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, path := range paths {
		for _, method := range methods {
			t.Run(method+path, func(t *testing.T) {
				req := requestWithTenant(method, srv.URL+path, tenantId)
				resp, err := (&http.Client{}).Do(req)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	}
}

func TestGetCrystalBandReturnsTheSeededRow(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	seedDerivedBands(t, db, tenantId)

	srv := httptest.NewServer(setupCrystalBandRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+"/crystal-bands/31", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)
	assert.Equal(t, "31", doc.Data.DataObject.ID)
	assert.Equal(t, "crystalBands", doc.Data.DataObject.Type)

	var attrs struct {
		MaxLevel      uint32 `json:"maxLevel"`
		CrystalItemId uint32 `json:"crystalItemId"`
		Count         uint32 `json:"count"`
	}
	require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
	assert.EqualValues(t, 50, attrs.MaxLevel)
	assert.EqualValues(t, 4260000, attrs.CrystalItemId)
	assert.EqualValues(t, 1, attrs.Count)
}

func TestGetCrystalBandNotFound(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupCrystalBandRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+"/crystal-bands/31", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetAllCrystalBandsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	seedDerivedBands(t, db, tenantId)

	srv := httptest.NewServer(setupCrystalBandRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+"/crystal-bands?page[number]=1&page[size]=5", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 5)
	require.NotNil(t, doc.Meta)
	assert.EqualValues(t, 9, doc.Meta["total"])
}
