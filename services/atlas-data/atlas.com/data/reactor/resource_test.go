package reactor

import (
	"atlas-data/document"
	"atlas-data/point"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDocumentEntity is a test-compatible version of document.Entity without PostgreSQL-specific defaults
type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null"`
	Type       string          `gorm:"not null"`
	DocumentId uint32          `gorm:"not null"`
	Content    json.RawMessage `gorm:"type:text;not null"`
}

func (e testDocumentEntity) TableName() string {
	return "documents"
}

// TestReactorResourceIntegration tests the REST API endpoints for reactor functionality
// Note: Reactor only has single-item endpoint (/{reactorId}), no collection endpoint
func TestReactorResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestReactorData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetReactorEndpoint", func(t *testing.T) {
		testGetReactorEndpoint(t, testServer, tenantId)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, testServer, tenantId)
	})

	t.Run("TenantIsolation", func(t *testing.T) {
		testTenantIsolation(t, testServer, tenantId)
	})

	t.Run("JSONAPICompliance", func(t *testing.T) {
		testJSONAPICompliance(t, testServer, tenantId)
	})
}

func setupResourceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&testDocumentEntity{})
	require.NoError(t, err)

	return db
}

func setupTestRouter(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	serverInfo := testServerInfo{}
	routeInitializer := InitResource(db)(serverInfo)
	routeInitializer(router, l)

	return router
}

type testServerInfo struct{}

func (t testServerInfo) GetVersion() string { return "1.0.0" }
func (t testServerInfo) GetURI() string     { return "/api/data/" }
func (t testServerInfo) GetPrefix() string  { return "/api/data/" }
func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }

func setupTestReactorData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	reactors := []RestModel{
		{
			Id:   2001000,
			Name: "Test Reactor",
			TL:   point.RestModel{X: -10, Y: -50},
			BR:   point.RestModel{X: 10, Y: 0},
			StateInfo: map[int8][]ReactorStateRestModel{
				0: {
					{Type: 0, NextState: 1, ActiveSkills: []uint32{}},
				},
				1: {
					{Type: 5, NextState: 127, ActiveSkills: []uint32{}},
				},
			},
			TimeoutInfo: map[int8]int32{},
		},
		{
			Id:   2001001,
			Name: "Item Reactor",
			TL:   point.RestModel{X: -20, Y: -60},
			BR:   point.RestModel{X: 20, Y: 0},
			StateInfo: map[int8][]ReactorStateRestModel{
				0: {
					{
						Type:      100,
						NextState: 1,
						ReactorItem: &ReactorItemRestModel{
							ItemId:   4000019,
							Quantity: 10,
						},
						ActiveSkills: []uint32{},
					},
				},
			},
			TimeoutInfo: map[int8]int32{
				0: 5000,
			},
		},
		{
			Id:          2006000,
			Name:        "Empty Reactor",
			TL:          point.RestModel{X: 0, Y: -100},
			BR:          point.RestModel{X: 50, Y: 0},
			StateInfo:   map[int8][]ReactorStateRestModel{},
			TimeoutInfo: map[int8]int32{},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "REACTOR")
	for _, r := range reactors {
		_, err := storage.Add(ctx)(r)()
		require.NoError(t, err)
	}
}

func createRequestWithTenant(method, url string, body []byte, tenantId uuid.UUID) *http.Request {
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

func testGetReactorEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetReactorById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/2001000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
		data := response["data"].(map[string]interface{})

		assert.Equal(t, "reactors", data["type"])
		assert.Equal(t, "2001000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Contains(t, attributes, "name")
		assert.Equal(t, "Test Reactor", attributes["name"])
		assert.Contains(t, attributes, "tl")
		assert.Contains(t, attributes, "br")
		assert.Contains(t, attributes, "stateInfo")
	})

	t.Run("GetReactorWithItemRequirement", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/2001001", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "2001001", data["id"])
	})

	t.Run("GetReactorNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidReactorId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/2001000", testServer.URL)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func testTenantIsolation(t *testing.T, testServer *httptest.Server, originalTenantId uuid.UUID) {
	t.Run("DifferentTenantNoData", func(t *testing.T) {
		differentTenantId := uuid.New()

		url := fmt.Sprintf("%s/data/reactors/2001000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, differentTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Reactor exists but not for this tenant - returns 404
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("OriginalTenantHasData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/2001000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, originalTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "2001000", data["id"])
	})
}

func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/reactors/2001000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.NotContains(t, response, "errors")

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "type")
		assert.Contains(t, data, "id")
		assert.Contains(t, data, "attributes")
		assert.Equal(t, "reactors", data["type"])
	})
}
