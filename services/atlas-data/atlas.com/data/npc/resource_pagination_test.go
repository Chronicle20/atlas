package npc

import (
	"atlas-data/quest"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestNpcsBareList_PaginationEnvelope exercises GET /data/npcs over the 4
// seeded NPCs (setupTestNpcData): page[size]=3 returns page 1 (3 items), a
// total of 4, and a next link.
func TestNpcsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs?page[number]=1&page[size]=3", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 3)
	assert.EqualValues(t, 4, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestNpcsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs?page[size]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestNpcsStorebankFilter_PaginationEnvelope covers the filter[storebank]
// arm (routed through handleSearchNpcs alongside ?search=): 2 of the 4
// seeded NPCs have Storebank=true.
func TestNpcsStorebankFilter_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs?filter[storebank]=true&page[number]=1&page[size]=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 1)
	assert.EqualValues(t, 2, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

// TestNpcsSearch_RejectsLegacyLimitParam confirms the search/storebank arm
// now rejects the pre-task-117 ad-hoc ?limit= param, matching the bare arm.
func TestNpcsSearch_RejectsLegacyLimitParam(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs?search=Bank&limit=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// testNpcMapSpawnIndexEntity is a sqlite-safe mirror of SpawnIndexEntity
// (the production tag `type:uuid` isn't understood by the sqlite driver).
type testNpcMapSpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	NpcId      uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testNpcMapSpawnIndexEntity) TableName() string { return "npc_spawn_index" }

func TestNpcMaps_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	require.NoError(t, db.AutoMigrate(&testNpcMapSpawnIndexEntity{}))
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	rows := []SpawnIndexEntity{
		{TenantId: tenantId, NpcId: 9010000, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 2, UpdatedAt: time.Now()},
		{TenantId: tenantId, NpcId: 9010000, MapId: 101000000, Name: "Ellinia", StreetName: "Victoria Road", SpawnCount: 1, UpdatedAt: time.Now()},
	}
	require.NoError(t, db.Create(&rows).Error)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs/9010000/maps?page[number]=1&page[size]=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 1)
	assert.EqualValues(t, 2, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestNpcMaps_RejectsBadPageNumber(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs/9010000/maps?page[number]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestNpcQuests_PaginationEnvelope covers the derived-filter by-parent route:
// 2 of 3 seeded quests reference npcId 9010000 somewhere in their
// requirements/actions.
func TestNpcQuests_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	qs := quest.NewStorage(l, db)
	quests := []quest.RestModel{
		{Id: 1000, Name: "Start Here", StartRequirements: quest.RequirementsRestModel{NpcId: 9010000}},
		{Id: 1001, Name: "End Here", EndRequirements: quest.RequirementsRestModel{NpcId: 9010000}},
		{Id: 1002, Name: "Unrelated", StartRequirements: quest.RequirementsRestModel{NpcId: 9010001}},
	}
	for _, q := range quests {
		_, err := qs.Add(ctx)(q)()
		require.NoError(t, err)
	}

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs/9010000/quests?page[number]=1&page[size]=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 1)
	assert.EqualValues(t, 2, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestNpcQuests_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestNpcData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/npcs/9010000/quests?page[size]=-1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
