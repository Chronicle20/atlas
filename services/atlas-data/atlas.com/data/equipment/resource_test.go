package equipment

import (
	"atlas-data/document"
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

// TestEquipmentResourceIntegration tests the REST API endpoints for equipment functionality
// Note: Equipment only has single-item endpoints (/{equipmentId} and /{equipmentId}/slots), no collection endpoint
func TestEquipmentResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestEquipmentData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetSingleEquipmentEndpoint", func(t *testing.T) {
		testGetSingleEquipmentEndpoint(t, testServer, tenantId)
	})

	t.Run("GetEquipmentSlotsEndpoint", func(t *testing.T) {
		testGetEquipmentSlotsEndpoint(t, testServer, tenantId)
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

func setupTestEquipmentData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	equipment := []RestModel{
		{
			Id:           1302000, // Sword
			Strength:     0,
			WeaponAttack: 17,
			Slots:        7,
			Price:        1000,
			Cash:         false,
			EquipSlots: []SlotRestModel{
				{Id: "Wp", Name: "Weapon", WZ: "Wp", Slot: -11},
			},
		},
		{
			Id:           1302001, // Another sword
			Strength:     1,
			WeaponAttack: 20,
			Slots:        7,
			Price:        2000,
			Cash:         false,
			EquipSlots: []SlotRestModel{
				{Id: "Wp", Name: "Weapon", WZ: "Wp", Slot: -11},
			},
		},
		{
			Id:            1002000, // Hat
			WeaponDefense: 5,
			Slots:         5,
			Price:         500,
			Cash:          false,
			EquipSlots: []SlotRestModel{
				{Id: "Cp", Name: "Cap", WZ: "Cp", Slot: -1},
			},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "EQUIPMENT")
	for _, e := range equipment {
		_, err := storage.Add(ctx)(e)()
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

func testGetSingleEquipmentEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetEquipmentById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000", testServer.URL)
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

		assert.Equal(t, "statistics", data["type"])
		assert.Equal(t, "1302000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, float64(17), attributes["weaponAttack"])
		assert.Equal(t, float64(7), attributes["slots"])
	})

	t.Run("GetEquipmentNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Equipment handler returns 500 for all errors including not found
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func testGetEquipmentSlotsEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetEquipmentSlots", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000/slots", testServer.URL)
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
		data := response["data"].([]interface{})
		assert.Len(t, data, 1)

		slot := data[0].(map[string]interface{})
		assert.Equal(t, "slots", slot["type"])
	})

	t.Run("GetEquipmentSlotsNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/9999999/slots", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Equipment handler returns 500 for all errors including not found
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidItemId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000", testServer.URL)
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

		// For single-item endpoint, different tenant should get 500 (not found)
		url := fmt.Sprintf("%s/data/equipment/1302000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, differentTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Equipment exists but not for this tenant - returns 500
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("OriginalTenantHasData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000", testServer.URL)
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
		assert.Equal(t, "1302000", data["id"])
	})
}

func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000", testServer.URL)
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
		assert.Equal(t, "statistics", data["type"])
	})

	t.Run("SlotsCollectionStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/equipment/1302000/slots", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
		data := response["data"].([]interface{})

		for _, item := range data {
			resource := item.(map[string]interface{})
			assert.Contains(t, resource, "type")
			assert.Contains(t, resource, "id")
			assert.Contains(t, resource, "attributes")
			assert.Equal(t, "slots", resource["type"])
		}
	})
}
