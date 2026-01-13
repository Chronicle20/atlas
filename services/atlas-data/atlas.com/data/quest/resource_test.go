package quest

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

// TestQuestResourceIntegration tests the REST API endpoints for quest functionality
func TestQuestResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestQuestData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetQuestsEndpoint", func(t *testing.T) {
		testGetQuestsEndpoint(t, testServer, tenantId)
	})

	t.Run("GetQuestEndpoint", func(t *testing.T) {
		testGetQuestEndpoint(t, testServer, tenantId)
	})

	t.Run("GetAutoStartQuestsEndpoint", func(t *testing.T) {
		testGetAutoStartQuestsEndpoint(t, testServer, tenantId)
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

func setupTestQuestData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	quests := []RestModel{
		{
			Id:        1000,
			Name:      "Beginner Quest",
			Area:      0,
			AutoStart: true,
			StartRequirements: RequirementsRestModel{
				LevelMin: 1,
				LevelMax: 10,
			},
			EndRequirements: RequirementsRestModel{
				NpcId: 9010000,
			},
			EndActions: ActionsRestModel{
				Exp: 100,
			},
		},
		{
			Id:        1001,
			Name:      "Second Quest",
			Area:      0,
			AutoStart: false,
			StartRequirements: RequirementsRestModel{
				LevelMin: 10,
				Quests: []QuestRequirement{
					{Id: 1000, State: 2},
				},
			},
			EndActions: ActionsRestModel{
				Exp:   500,
				Money: 1000,
			},
		},
		{
			Id:        1002,
			Name:      "Auto Start Quest 2",
			Area:      100,
			AutoStart: true,
			StartRequirements: RequirementsRestModel{
				LevelMin: 20,
			},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "QUEST")
	for _, q := range quests {
		_, err := storage.Add(ctx)(q)()
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

func testGetQuestsEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAllQuests", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests", testServer.URL)
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

func testGetQuestEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetQuestById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests/1000", testServer.URL)
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

		assert.Equal(t, "quests", data["type"])
		assert.Equal(t, "1000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, "Beginner Quest", attributes["name"])
		assert.Equal(t, true, attributes["autoStart"])
	})

	t.Run("GetQuestNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func testGetAutoStartQuestsEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAutoStartQuests", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests/auto-start", testServer.URL)
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
		assert.Len(t, data, 2) // Only auto-start quests

		for _, item := range data {
			quest := item.(map[string]interface{})
			attributes := quest["attributes"].(map[string]interface{})
			assert.Equal(t, true, attributes["autoStart"])
		}
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidQuestId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, nil, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests", testServer.URL)
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

		url := fmt.Sprintf("%s/data/quests", testServer.URL)
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
		url := fmt.Sprintf("%s/data/quests", testServer.URL)
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
		url := fmt.Sprintf("%s/data/quests/1000", testServer.URL)
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
		assert.Equal(t, "quests", data["type"])
	})

	t.Run("CollectionResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/quests", testServer.URL)
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
			assert.Equal(t, "quests", resource["type"])
		}
	})
}
