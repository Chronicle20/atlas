package monster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonstersBareList_PaginationEnvelope exercises GET /data/monsters over
// the 3 seeded monsters (setupTestMonsterData): page[size]=2 returns page 1
// (2 items), a total of 3, and a next link.
func TestMonstersBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters?page[number]=1&page[size]=2", ts.URL)
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

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestMonstersBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters?page[size]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// testMonsterSpawnIndexEntity is a sqlite-safe mirror of SpawnIndexEntity
// (the production tag `type:uuid` isn't understood by the sqlite driver).
type testMonsterSpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:text;primaryKey"`
	MonsterId  uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (testMonsterSpawnIndexEntity) TableName() string { return "monster_spawn_index" }

func TestMonsterMaps_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	require.NoError(t, db.AutoMigrate(&testMonsterSpawnIndexEntity{}))
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	rows := []SpawnIndexEntity{
		{TenantId: tenantId, MonsterId: 100100, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 5, UpdatedAt: time.Now()},
		{TenantId: tenantId, MonsterId: 100100, MapId: 101000000, Name: "Ellinia", StreetName: "Victoria Road", SpawnCount: 3, UpdatedAt: time.Now()},
		{TenantId: tenantId, MonsterId: 100100, MapId: 102000000, Name: "Perion", StreetName: "Victoria Road", SpawnCount: 1, UpdatedAt: time.Now()},
	}
	require.NoError(t, db.Create(&rows).Error)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters/100100/maps?page[number]=1&page[size]=2", ts.URL)
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

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestMonsterMaps_RejectsBadPageNumber(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters/100100/maps?page[number]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestMonsterLoseItems_PaginationEnvelope covers the by-parent sub-list
// route (loseItems) sliced from a single monster document.
func TestMonsterLoseItems_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := NewStorage(l, db)
	_, err = storage.Add(ctx)(RestModel{
		Id:   100100,
		Name: "Blue Snail",
		LoseItems: []loseItem{
			{Id: 4000019, Chance: 10, X: 1},
			{Id: 4000020, Chance: 20, X: 1},
			{Id: 4000021, Chance: 30, X: 1},
		},
	})()
	require.NoError(t, err)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters/100100/loseItems?page[number]=1&page[size]=2", ts.URL)
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

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestMonsterLoseItems_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters/100100/loseItems?page[size]=-1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestMonstersSearch_RejectsLegacyLimitParam confirms the search arm now
// rejects the pre-task-117 ad-hoc ?limit= param, matching the bare arm's
// paginate.ParseParams contract.
func TestMonstersSearch_RejectsLegacyLimitParam(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestMonsterData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/monsters?search=Snail&limit=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
