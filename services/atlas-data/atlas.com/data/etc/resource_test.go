package etc

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

// TestEtcResourceIntegration tests the REST API endpoints for etc item functionality
func TestEtcResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestEtcData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetEtcsEndpoint", func(t *testing.T) {
		testGetEtcsEndpoint(t, testServer, tenantId)
	})

	t.Run("GetEtcEndpoint", func(t *testing.T) {
		testGetEtcEndpoint(t, testServer, tenantId)
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

func setupTestEtcData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	etcItems := []RestModel{
		{
			Id:      4000000,
			Price:   10,
			SlotMax: 100,
		},
		{
			Id:      4000001,
			Price:   20,
			SlotMax: 100,
		},
		{
			Id:      4000002,
			Price:   50,
			SlotMax: 200,
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "ETC")
	for _, e := range etcItems {
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

func testGetEtcsEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAllEtcs", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs", testServer.URL)
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
		assert.Len(t, data, 3)
	})
}

func testGetEtcEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetEtcById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs/4000000", testServer.URL)
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

		assert.Equal(t, "etcs", data["type"])
		assert.Equal(t, "4000000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, float64(10), attributes["price"])
		assert.Equal(t, float64(100), attributes["slotMax"])
	})

	t.Run("GetEtcNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidItemId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs", testServer.URL)
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

		url := fmt.Sprintf("%s/data/etcs", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, differentTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		assert.Len(t, data, 0)
	})

	t.Run("OriginalTenantHasData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs", testServer.URL)
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
		assert.Len(t, data, 3)
	})
}

func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs/4000000", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "type")
		assert.Contains(t, data, "id")
		assert.Contains(t, data, "attributes")
		assert.Equal(t, "etcs", data["type"])
	})

	t.Run("CollectionResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/etcs", testServer.URL)
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
			assert.Equal(t, "etcs", resource["type"])
		}
	})
}
