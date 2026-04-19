package npc

import (
	"atlas-data/document"
	"atlas-data/quest"
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

type testSearchIndexEntity struct {
	TenantId  uuid.UUID `gorm:"type:text;primaryKey"`
	NpcId     uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Storebank bool      `gorm:"not null;default:false"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (testSearchIndexEntity) TableName() string { return "npc_search_index" }

type testSpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	NpcId      uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testSpawnIndexEntity) TableName() string { return "npc_spawn_index" }

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

// TestNpcResourceIntegration tests the REST API endpoints for NPC functionality
func TestNpcResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetNpcsEndpoint", func(t *testing.T) {
		testGetNpcsEndpoint(t, testServer, tenantId)
	})

	t.Run("GetNpcEndpoint", func(t *testing.T) {
		testGetNpcEndpoint(t, testServer, tenantId)
	})

	t.Run("FilterEndpoints", func(t *testing.T) {
		testFilterEndpoints(t, testServer, tenantId)
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

	t.Run("GetNpcMaps", func(t *testing.T) {
		testGetNpcMaps(t, testServer, db, tenantId)
	})

	t.Run("SingularMapRetired", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9010000/map", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetNpcQuests", func(t *testing.T) {
		testGetNpcQuests(t, testServer, db, tenantId)
	})
}

func testGetNpcMaps(t *testing.T, testServer *httptest.Server, db *gorm.DB, tenantId uuid.UUID) {
	// seed spawn index rows: NPC 9010000 on two maps; NPC 9010001 on one map.
	rows := []testSpawnIndexEntity{
		{TenantId: tenantId, NpcId: 9010000, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 3},
		{TenantId: tenantId, NpcId: 9010000, MapId: 200000000, Name: "Ellinia", StreetName: "Victoria Road", SpawnCount: 1},
		{TenantId: tenantId, NpcId: 9010001, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 2},
	}
	require.NoError(t, db.Create(&rows).Error)

	t.Run("ReturnsAllRowsSorted", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9010000/maps", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		require.Len(t, data, 2)

		first := data[0].(map[string]interface{})
		assert.Equal(t, "npc-maps", first["type"])
		assert.Equal(t, "100000000", first["id"])
		firstAttrs := first["attributes"].(map[string]interface{})
		assert.Equal(t, float64(100000000), firstAttrs["mapId"])
		assert.Equal(t, "Henesys", firstAttrs["name"])
		assert.Equal(t, float64(3), firstAttrs["spawnCount"])

		second := data[1].(map[string]interface{})
		assert.Equal(t, "200000000", second["id"])
		secondAttrs := second["attributes"].(map[string]interface{})
		assert.Equal(t, float64(1), secondAttrs["spawnCount"])
	})

	t.Run("EmptyResultReturnsEmptyList", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9999999/maps", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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

	t.Run("BadIdReturns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/notanumber/maps", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantReturns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9010000/maps", testServer.URL)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("TenantIsolation", func(t *testing.T) {
		otherTenantId := uuid.New()
		url := fmt.Sprintf("%s/data/npcs/9010000/maps", testServer.URL)
		req := createRequestWithTenant("GET", url, otherTenantId)

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
}

func testGetNpcQuests(t *testing.T, testServer *httptest.Server, db *gorm.DB, tenantId uuid.UUID) {
	// Seed quest documents for the tenant.
	quests := []quest.RestModel{
		{
			Id:   2000,
			Name: "Initiator Only Quest",
			Area: 1,
			StartRequirements: quest.RequirementsRestModel{NpcId: 1012100, LevelMin: 50},
			EndActions:        quest.ActionsRestModel{Exp: 5000},
		},
		{
			Id:   2001,
			Name: "Completer Only Quest",
			Area: 1,
			StartRequirements: quest.RequirementsRestModel{LevelMin: 10},
			EndActions:        quest.ActionsRestModel{NpcId: 1012100, Exp: 7500},
		},
		{
			Id:   2002,
			Name: "Both Quest",
			Area: 1,
			StartRequirements: quest.RequirementsRestModel{NpcId: 1012100},
			EndActions:        quest.ActionsRestModel{NpcId: 1012100},
		},
		{
			Id:   2003,
			Name: "Unrelated Quest",
			Area: 1,
			StartRequirements: quest.RequirementsRestModel{NpcId: 9999999},
			EndActions:        quest.ActionsRestModel{NpcId: 9999999},
		},
		{
			Id:   2004,
			Name: "EndRequirements Only",
			Area: 1,
			EndRequirements: quest.RequirementsRestModel{NpcId: 1012100},
		},
		{
			Id:   2005,
			Name: "StartActions Only",
			Area: 1,
			StartActions: quest.ActionsRestModel{NpcId: 1012100},
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, quest.GetModelRegistry(), "QUEST")
	for _, q := range quests {
		_, err := storage.Add(ctx)(q)()
		require.NoError(t, err)
	}

	t.Run("ReturnsMatchingQuestsSortedById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/1012100/quests", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		require.Len(t, data, 5)

		ids := make([]string, 0, len(data))
		for _, item := range data {
			resource := item.(map[string]interface{})
			assert.Equal(t, "quests", resource["type"])
			ids = append(ids, resource["id"].(string))
		}
		assert.Equal(t, []string{"2000", "2001", "2002", "2004", "2005"}, ids)

		// Confirm that the "Both Quest" appears exactly once.
		count := 0
		for _, id := range ids {
			if id == "2002" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("UnrelatedNpcReturnsEmpty", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/1234567/quests", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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

	t.Run("TenantWithNoQuestsReturnsEmpty", func(t *testing.T) {
		otherTenant := uuid.New()
		url := fmt.Sprintf("%s/data/npcs/1012100/quests", testServer.URL)
		req := createRequestWithTenant("GET", url, otherTenant)

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

	t.Run("BadIdReturns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/notanumber/quests", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantReturns400", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/1012100/quests", testServer.URL)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
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

	err = db.AutoMigrate(&testDocumentEntity{}, &testSearchIndexEntity{}, &testSpawnIndexEntity{})
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

func setupTestNpcData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	npcs := []RestModel{
		{
			Id:        9010000,
			Name:      "Maple Administrator",
			TrunkPut:  0,
			TrunkGet:  0,
			Storebank: false,
			HideName:  false,
		},
		{
			Id:        9010001,
			Name:      "Storage Keeper",
			TrunkPut:  500,
			TrunkGet:  500,
			Storebank: true,
			HideName:  false,
		},
		{
			Id:        9010002,
			Name:      "Bank Teller",
			TrunkPut:  100,
			TrunkGet:  100,
			Storebank: true,
			HideName:  false,
		},
		{
			Id:        2040000,
			Name:      "Nella",
			TrunkPut:  0,
			TrunkGet:  0,
			Storebank: false,
			HideName:  true,
		},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := NewStorage(l, db)
	for _, n := range npcs {
		_, err := storage.Add(ctx)(n)()
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

func testGetNpcsEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetAllNpcs", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs", testServer.URL)
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
		assert.Len(t, data, 4)
	})
}

func testGetNpcEndpoint(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("GetNpcById", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9010000", testServer.URL)
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

		assert.Equal(t, "npcs", data["type"])
		assert.Equal(t, "9010000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, "Maple Administrator", attributes["name"])
		assert.Equal(t, false, attributes["storebank"])
	})

	t.Run("GetNpcNotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func testFilterEndpoints(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("FilterStorebankTrue", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs?filter[storebank]=true", testServer.URL)
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
		assert.Len(t, data, 2) // Only storebank NPCs

		for _, item := range data {
			npc := item.(map[string]interface{})
			attributes := npc["attributes"].(map[string]interface{})
			assert.Equal(t, true, attributes["storebank"])
		}
	})
}

func testErrorHandling(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("InvalidNpcId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/invalid", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MissingTenantHeader", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs", testServer.URL)
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

		url := fmt.Sprintf("%s/data/npcs", testServer.URL)
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
		url := fmt.Sprintf("%s/data/npcs", testServer.URL)
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
		assert.Len(t, data, 4)
	})
}

func testJSONAPICompliance(t *testing.T, testServer *httptest.Server, tenantId uuid.UUID) {
	t.Run("SingleResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs/9010000", testServer.URL)
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
		assert.Equal(t, "npcs", data["type"])
	})

	t.Run("CollectionResourceStructure", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/npcs", testServer.URL)
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
			assert.Equal(t, "npcs", resource["type"])
		}
	})
}
