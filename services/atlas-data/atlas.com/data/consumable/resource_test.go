package consumable

import (
	"atlas-data/document"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas-database"
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

// TestConsumableResourceIntegration tests the REST API endpoints for consumable functionality
func TestConsumableResourceIntegration(t *testing.T) {
	// Setup test database
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestConsumableData(t, db, tenantId)

	// Setup test server
	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetConsumablesEndpoint", func(t *testing.T) {
		testGetConsumablesEndpoint(t, testServer, tenantId)
	})

	t.Run("GetConsumableEndpoint", func(t *testing.T) {
		testGetConsumableEndpoint(t, testServer, tenantId)
	})

	t.Run("FilterConsumablesEndpoint", func(t *testing.T) {
		testFilterConsumablesEndpoint(t, testServer, tenantId)
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

// setupResourceTestDB creates an in-memory SQLite database for testing
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

	// Run migrations using test-compatible entity (avoids PostgreSQL-specific uuid_generate_v4)
	err = db.AutoMigrate(&testDocumentEntity{})
	require.NoError(t, err)

	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)

	return db
}

// setupTestRouter creates a test router with consumable routes
func setupTestRouter(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create server info
	serverInfo := testServerInfo{}

	// Initialize routes
	routeInitializer := InitResource(db)(serverInfo)
	routeInitializer(router, l)

	return router
}

// testServerInfo implements jsonapi.ServerInformation for testing
type testServerInfo struct{}

func (t testServerInfo) GetVersion() string { return "1.0.0" }
func (t testServerInfo) GetURI() string     { return "/api/data/" }
func (t testServerInfo) GetPrefix() string  { return "/api/data/" }
func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }

// setupTestConsumableData creates test consumable data in the database
func setupTestConsumableData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	// Create test consumables
	consumables := []RestModel{
		{
			Id:           2000000,
			Price:        50,
			SlotMax:      100,
			Rechargeable: false,
			Spec: map[SpecType]int32{
				SpecTypeHP: 50,
			},
		},
		{
			Id:           2000001,
			Price:        200,
			SlotMax:      100,
			Rechargeable: false,
			Spec: map[SpecType]int32{
				SpecTypeHP: 150,
			},
		},
		{
			Id:           2000002,
			Price:        500,
			SlotMax:      100,
			Rechargeable: false,
			Spec: map[SpecType]int32{
				SpecTypeHP: 300,
			},
		},
		{
			Id:           2070000,
			Price:        500,
			SlotMax:      1000,
			Rechargeable: true,
		},
	}

	// Create a context with the test tenant
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	// Use the document storage to add consumables (handles JSON:API serialization)
	storage := document.NewStorage(l, db, GetModelRegistry(), "CONSUMABLE")
	for _, c := range consumables {
		_, err := storage.Add(ctx)(c)()
		require.NoError(t, err)
	}
}

// createRequestWithTenant creates an HTTP request with tenant headers
func createRequestWithTenant(method, url string, body []byte, tenantId uuid.UUID) *http.Request {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		panic(err)
	}

	// Add tenant headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	return req
}

// testGetConsumablesEndpoint tests GET /data/consumables
func testGetConsumablesEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAllConsumables", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify JSON:API structure
		assert.Contains(t, response, "data")
		data := response["data"].([]interface{})

		// Should have all 4 test consumables
		assert.Len(t, data, 4)

		// Verify each consumable has proper structure
		for _, item := range data {
			consumable := item.(map[string]interface{})
			assert.Contains(t, consumable, "type")
			assert.Contains(t, consumable, "id")
			assert.Contains(t, consumable, "attributes")
		}
	})
}

// testGetConsumableEndpoint tests GET /data/consumables/{itemId}
func testGetConsumableEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetConsumableById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables/2000000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify JSON:API structure
		assert.Contains(t, response, "data")
		data := response["data"].(map[string]interface{})

		assert.Equal(t, "consumables", data["type"])
		assert.Equal(t, "2000000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, float64(50), attributes["price"])
		assert.Equal(t, float64(100), attributes["slotMax"])
	})

	t.Run("GetConsumableNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// testFilterConsumablesEndpoint tests GET /data/consumables with filter
func testFilterConsumablesEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("FilterRechargeableTrue", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables?filter[rechargeable]=true", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})

		// Should only have rechargeable items (1 throwing star)
		assert.Len(t, data, 1)

		consumable := data[0].(map[string]interface{})
		attributes := consumable["attributes"].(map[string]interface{})
		assert.Equal(t, true, attributes["rechargeable"])
	})

	t.Run("FilterRechargeableFalse", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables?filter[rechargeable]=false", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})

		// Should have 3 non-rechargeable potions
		assert.Len(t, data, 3)
	})
}

// testErrorHandling tests various error scenarios
func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidItemId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")
		// Missing TENANT_ID header

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error due to missing tenant context
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req := createRequestWithTenant("POST", url, []byte("{}"), tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Gorilla Mux returns 404 for unregistered method/path combinations
		// (not 405 which would require explicit method-specific handling)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// testTenantIsolation tests that tenant isolation works correctly
func testTenantIsolation(t *testing.T, testServer *httptest.Server, originalTenantId uuid.UUID) {
	t.Run("DifferentTenantNoData", func(t *testing.T) {
		// Use a different tenant ID
		differentTenantId := uuid.New()

		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, differentTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return OK but with empty data (or fallback to region data)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		// Different tenant should see empty array (no fallback region data in test)
		assert.Len(t, data, 0)
	})

	t.Run("OriginalTenantHasData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, originalTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		assert.Len(t, data, 4)
	})
}

// testJSONAPICompliance tests JSON:API specification compliance
func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables/2000000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify top-level structure
		assert.Contains(t, response, "data")
		assert.NotContains(t, response, "errors")

		// Verify resource object structure
		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "type")
		assert.Contains(t, data, "id")
		assert.Contains(t, data, "attributes")

		// Verify type is correct
		assert.Equal(t, "consumables", data["type"])
	})

	t.Run("CollectionResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/consumables", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify collection structure
		assert.Contains(t, response, "data")
		data := response["data"].([]interface{})

		// Verify each resource in collection
		for _, item := range data {
			resource := item.(map[string]interface{})
			assert.Contains(t, resource, "type")
			assert.Contains(t, resource, "id")
			assert.Contains(t, resource, "attributes")
			assert.Equal(t, "consumables", resource["type"])
		}
	})
}
