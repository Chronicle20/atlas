package templates

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

// TestCharacterTemplatesResourceIntegration tests the REST API endpoints for character templates
// Note: Character templates only has a collection endpoint, no single-item endpoint
func TestCharacterTemplatesResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestTemplateData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetTemplatesEndpoint", func(t *testing.T) {
		testGetTemplatesEndpoint(t, testServer, tenantId)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, testServer)
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

	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)

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

func setupTestTemplateData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	templates := []RestModel{
		{
			Id:            0,
			CharacterType: "adventurer",
			Faces:         []uint32{20000, 20001, 20002},
			HairStyles:    []uint32{30000, 30010, 30020},
			HairColors:    []uint32{0, 1, 2, 3},
			SkinColors:    []uint32{0, 1, 2},
			Tops:          []uint32{1040002, 1040006},
			Bottoms:       []uint32{1060002, 1060006},
			Shoes:         []uint32{1072001, 1072005},
			Weapons:       []uint32{1302000, 1322005},
		},
		{
			Id:            1,
			CharacterType: "cygnus",
			Faces:         []uint32{20100, 20101},
			HairStyles:    []uint32{30100, 30110},
			HairColors:    []uint32{0, 1, 2},
			SkinColors:    []uint32{0, 1},
			Tops:          []uint32{1041002},
			Bottoms:       []uint32{1061002},
			Shoes:         []uint32{1072101},
			Weapons:       []uint32{1302100},
		},
		{
			Id:            2,
			CharacterType: "aran",
			Faces:         []uint32{20200, 20201},
			HairStyles:    []uint32{30200, 30210},
			HairColors:    []uint32{0, 1},
			SkinColors:    []uint32{0, 1},
			Tops:          []uint32{1042001},
			Bottoms:       []uint32{1062001},
			Shoes:         []uint32{1072201},
			Weapons:       []uint32{1442000},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "CHARACTER_TEMPLATE")
	for _, tmpl := range templates {
		_, err := storage.Add(ctx)(tmpl)()
		require.NoError(t, err)
	}
}

func createRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func testGetTemplatesEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAllTemplates", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/characters/templates", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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

		// Verify first template
		for _, item := range data {
			template := item.(map[string]interface{})
			assert.Contains(t, template, "type")
			assert.Contains(t, template, "id")
			assert.Contains(t, template, "attributes")

			attributes := template["attributes"].(map[string]interface{})
			assert.Contains(t, attributes, "characterType")
			assert.Contains(t, attributes, "faces")
			assert.Contains(t, attributes, "hairStyles")
		}
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server) {
	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/characters/templates", testServer.URL)
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

		url := fmt.Sprintf("%s/data/characters/templates", testServer.URL)
		req := createRequestWithTenant("GET", url, differentTenantId)

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
		url := fmt.Sprintf("%s/data/characters/templates", testServer.URL)
		req := createRequestWithTenant("GET", url, originalTenantId)

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
	t.Run("CollectionResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/characters/templates", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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
			assert.Equal(t, "characterTemplates", resource["type"])
		}
	})
}
