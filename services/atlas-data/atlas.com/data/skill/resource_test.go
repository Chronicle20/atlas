package skill

import (
	"atlas-data/document"
	"atlas-data/skill/effect"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e testDocumentEntity) TableName() string {
	return "documents"
}

// TestSkillResourceIntegration tests the REST API endpoints for skill functionality
// Note: Skill only has single-item endpoint (/{skillId}), no collection endpoint
func TestSkillResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestSkillData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetSkillEndpoint", func(t *testing.T) {
		testGetSkillEndpoint(t, testServer, tenantId)
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

func setupTestSkillData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	skills := []RestModel{
		{
			Id:            1001004, // Warrior skill
			Action:        true,
			Element:       "physical",
			AnimationTime: 600,
			Effects: []effect.RestModel{
				{MPConsume: 5, Duration: 0, Damage: 100},
			},
		},
		{
			Id:            2001002, // Magician skill
			Action:        true,
			Element:       "fire",
			AnimationTime: 800,
			Effects: []effect.RestModel{
				{MPConsume: 10, Duration: 120000, MagicAttack: 20},
			},
		},
		{
			Id:            3001004, // Archer skill
			Action:        false,
			Element:       "neutral",
			AnimationTime: 0,
			Effects: []effect.RestModel{
				{MPConsume: 8, Duration: 180000, WeaponAttack: 15},
			},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "SKILL")
	for _, s := range skills {
		_, err := storage.Add(ctx)(s)()
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

func testGetSkillEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetSkillById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/1001004", testServer.URL)
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
		data := response["data"].(map[string]interface{})

		assert.Equal(t, "skills", data["type"])
		assert.Equal(t, "1001004", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, true, attributes["action"])
		assert.Equal(t, "physical", attributes["element"])
	})

	t.Run("GetSkillNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidSkillId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/1001004", testServer.URL)
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

		url := fmt.Sprintf("%s/data/skills/1001004", testServer.URL)
		req := createRequestWithTenant("GET", url, differentTenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Skill exists but not for this tenant - returns 404
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("OriginalTenantHasData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/1001004", testServer.URL)
		req := createRequestWithTenant("GET", url, originalTenantId)

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
		assert.Equal(t, "1001004", data["id"])
	})
}

func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills/1001004", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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
		assert.Equal(t, "skills", data["type"])
	})
}

func TestSkillIdsFilter(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestSkillData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	client := &http.Client{}

	t.Run("IdsCSV_ReturnsBoth", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills?ids=1001004,3001004", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Contains(t, response, "data")
		items := response["data"].([]interface{})
		assert.Len(t, items, 2)

		ids := make(map[string]bool)
		for _, item := range items {
			elem := item.(map[string]interface{})
			ids[elem["id"].(string)] = true
		}
		assert.True(t, ids["1001004"])
		assert.True(t, ids["3001004"])
	})

	t.Run("IdsNoMatch_EmptyData", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills?ids=9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Contains(t, response, "data")
		items := response["data"].([]interface{})
		assert.Len(t, items, 0)
	})

	t.Run("IdsRepeated_ReturnsBoth", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills?ids=1001004&ids=2001002", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Contains(t, response, "data")
		items := response["data"].([]interface{})
		assert.Len(t, items, 2)

		ids := make(map[string]bool)
		for _, item := range items {
			elem := item.(map[string]interface{})
			ids[elem["id"].(string)] = true
		}
		assert.True(t, ids["1001004"])
		assert.True(t, ids["2001002"])
	})

	t.Run("IdsMalformed_Returns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills?ids=abc", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("IdsWinsOverName_ReturnsOnlyById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills?ids=1001004&name=ignored", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Contains(t, response, "data")
		items := response["data"].([]interface{})
		assert.Len(t, items, 1)
		elem := items[0].(map[string]interface{})
		assert.Equal(t, "1001004", elem["id"].(string))
	})

	t.Run("NoParams_Returns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/skills", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("NameSearch_StillWorks", func(t *testing.T) {
		// The name= path is preserved with 10-result cap
		url := fmt.Sprintf("%s/data/skills?name=a", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Just verifying the endpoint still returns 200 for a valid name= query
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
	})
}
