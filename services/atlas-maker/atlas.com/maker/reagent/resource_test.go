package reagent_test

import (
	"atlas-maker/reagent"
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
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupReagentRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	reagent.InitResource(&testServerInformation{})(db)(r, l)
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

// TestResourceIsReadOnly enforces FR-2.3 for reference data: a writable reagent
// table would let a client retune its own crafted-equip stat bonuses.
func TestResourceIsReadOnly(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	seedReagent(t, db, tenantId, item.Id(4250000), "incPAD", 1)

	srv := httptest.NewServer(setupReagentRouter(db))
	defer srv.Close()

	paths := []string{"/reagents", "/reagents/4250000"}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, path := range paths {
		for _, method := range methods {
			t.Run(method+path, func(t *testing.T) {
				req := requestWithTenant(method, srv.URL+path, tenantId)
				resp, err := (&http.Client{}).Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	}
}

// TestResourceStaysReadOnlyBesideTheSeedRoutes composes the reagent routes the
// way main.go does — with the seed catalog mounted under the same /reagents
// prefix afterwards. gorilla/mux only reports the method mismatch of the last
// route it tried, so a resource that leans on implicit 405 handling silently
// degrades to 404 here; and the seed route must still be reachable.
func TestResourceStaysReadOnlyBesideTheSeedRoutes(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()

	router := setupReagentRouter(db)
	seedRoutes := router.PathPrefix("/reagents").Subrouter()
	seedRoutes.HandleFunc("/seed", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}).Methods(http.MethodPost)

	srv := httptest.NewServer(router)
	defer srv.Close()

	for _, path := range []string{"/reagents", "/reagents/4250000"} {
		req := requestWithTenant(http.MethodPost, srv.URL+path, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "POST %s", path)
	}

	req := requestWithTenant(http.MethodPost, srv.URL+"/reagents/seed", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "the seed route must stay reachable")
}

func TestGetReagentReturnsTheSeededRow(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	seedReagent(t, db, tenantId, item.Id(4251202), "incReqLevel", -3)

	srv := httptest.NewServer(setupReagentRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+"/reagents/4251202", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)
	assert.Equal(t, "4251202", doc.Data.DataObject.ID)
	assert.Equal(t, "reagents", doc.Data.DataObject.Type)

	var attrs struct {
		Stat  string `json:"stat"`
		Value int16  `json:"value"`
	}
	require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
	assert.Equal(t, "incReqLevel", attrs.Stat)
	assert.Equal(t, int16(-3), attrs.Value)
}

func TestGetReagentNotFound(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupReagentRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+"/reagents/2000000", tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetAllReagentsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	seedReagent(t, db, tenantId, item.Id(4250000), "incPAD", 1)
	seedReagent(t, db, tenantId, item.Id(4250001), "incPAD", 2)
	seedReagent(t, db, tenantId, item.Id(4250002), "incPAD", 3)

	srv := httptest.NewServer(setupReagentRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/reagents?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)
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
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/reagents?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
