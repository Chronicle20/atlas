package itemmake

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

const testXMLForResource = `<imgdir name="ItemMake.img">
  <imgdir name="0">
    <imgdir name="04260000">
      <int name="reqLevel" value="0"/>
      <int name="reqSkillLevel" value="0"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="0"/>
      <int name="meso" value="0"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000000"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="randomReward">
        <imgdir name="0">
          <int name="item" value="4260000"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="70"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4260001"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="25"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4260002"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="5"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="01082002">
      <int name="reqLevel" value="30"/>
      <int name="reqSkillLevel" value="2"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="7"/>
      <int name="meso" value="1200"/>
      <int name="catalyst" value="4130000"/>
      <int name="reqItem" value="4000021"/>
      <int name="reqEquip" value="1002419"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4011001"/>
          <int name="count" value="5"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4011002"/>
          <int name="count" value="3"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4021007"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="reqQuest">
        <int name="21614" value="3"/>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="2">
    <imgdir name="02020000">
      <int name="reqLevel" value="10"/>
      <int name="itemNum" value="3"/>
      <int name="meso" value="500"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000001"/>
          <int name="count" value="2"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="4">
    <imgdir name="04030000">
      <int name="reqLevel" value="15"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="800"/>
    </imgdir>
  </imgdir>
  <imgdir name="8">
    <imgdir name="08000000">
      <int name="reqLevel" value="20"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="900"/>
    </imgdir>
  </imgdir>
  <imgdir name="16">
    <imgdir name="16000000">
      <int name="reqLevel" value="25"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="1000"/>
    </imgdir>
  </imgdir>
</imgdir>`

func setupResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

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

func newTestContext(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

func seedFromFixture(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	t.Helper()

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := newTestContext(t, tenantId)

	ms, err := Read(l)(xml.FromByteArrayProvider([]byte(testXMLForResource)))()
	require.NoError(t, err)

	storage := document.NewStorage(l, db, GetModelRegistry(), "ITEM_MAKE")
	for _, m := range ms {
		_, err := storage.Add(ctx)(m)()
		require.NoError(t, err)
	}
}

func TestGetItemMakeById(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedFromFixture(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/item-makes/1082002", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "itemMakes", data["type"])
	assert.Equal(t, "1082002", data["id"])

	attributes := data["attributes"].(map[string]interface{})
	assert.Equal(t, float64(1), attributes["group"])
	assert.Equal(t, float64(30), attributes["reqLevel"])
	assert.Equal(t, float64(2), attributes["reqSkillLevel"])
	assert.Equal(t, float64(1), attributes["itemNum"])
	assert.Equal(t, float64(7), attributes["tuc"])
	assert.Equal(t, float64(1200), attributes["meso"])
	assert.Equal(t, float64(4130000), attributes["catalyst"])
	assert.Equal(t, float64(4000021), attributes["reqItem"])
	assert.Equal(t, float64(1002419), attributes["reqEquip"])

	recipe := attributes["recipe"].([]interface{})
	expectedRecipe := []map[string]float64{
		{"itemId": 4011001, "count": 5},
		{"itemId": 4011002, "count": 3},
		{"itemId": 4021007, "count": 1},
	}
	require.Len(t, recipe, len(expectedRecipe))
	for i, e := range expectedRecipe {
		entry := recipe[i].(map[string]interface{})
		assert.Equal(t, e["itemId"], entry["itemId"])
		assert.Equal(t, e["count"], entry["count"])
	}

	reqQuest := attributes["reqQuest"].([]interface{})
	require.Len(t, reqQuest, 1)
	q := reqQuest[0].(map[string]interface{})
	assert.Equal(t, float64(21614), q["questId"])
	assert.Equal(t, float64(3), q["state"])
}

func TestGetItemMakeByIdFromEachGroup(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedFromFixture(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	expected := map[uint32]uint32{
		4260000:  0,
		1082002:  1,
		2020000:  2,
		4030000:  4,
		8000000:  8,
		16000000: 16,
	}

	for id, group := range expected {
		url := fmt.Sprintf("%s/data/item-makes/%d", ts.URL, id)
		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		resp.Body.Close()

		data := response["data"].(map[string]interface{})
		attributes := data["attributes"].(map[string]interface{})
		assert.Equal(t, float64(group), attributes["group"], "id=%d", id)
	}
}

func TestGetItemMakeByIdNotFound(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/item-makes/9999999", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetItemMakeRandomRewardOrderSurvivesREST(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedFromFixture(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/item-makes/4260000", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	data := response["data"].(map[string]interface{})
	attributes := data["attributes"].(map[string]interface{})
	randomReward := attributes["randomReward"].([]interface{})

	expected := []map[string]float64{
		{"itemId": 4260000, "itemNum": 1, "prob": 70},
		{"itemId": 4260001, "itemNum": 1, "prob": 25},
		{"itemId": 4260002, "itemNum": 1, "prob": 5},
	}
	require.Len(t, randomReward, len(expected))
	for i, e := range expected {
		entry := randomReward[i].(map[string]interface{})
		assert.Equal(t, e["itemId"], entry["itemId"])
		assert.Equal(t, e["itemNum"], entry["itemNum"])
		assert.Equal(t, e["prob"], entry["prob"])
	}
}

func TestListItemMakesPaginates(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := newTestContext(t, tenantId)
	storage := document.NewStorage(l, db, GetModelRegistry(), "ITEM_MAKE")
	for i := uint32(1); i <= 25; i++ {
		_, err := storage.Add(ctx)(RestModel{
			Id:           4000000 + i,
			Group:        0,
			ItemNum:      1,
			Recipe:       []MaterialRestModel{},
			RandomReward: []RewardRestModel{},
			ReqQuest:     []QuestReqRestModel{},
		})()
		require.NoError(t, err)
	}

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/item-makes?page[number]=2&page[size]=10", ts.URL)
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

	assert.Len(t, doc.Data, 10)
	assert.EqualValues(t, 25, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestRegisterIsIdempotent(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := newTestContext(t, tenantId)

	processor := &ProcessorImpl{l: l, ctx: ctx, db: db}
	provider := Read(l)(xml.FromByteArrayProvider([]byte(testXMLForResource)))
	storage := NewStorage(l, db)

	require.NoError(t, processor.Register(storage, provider))

	var countAfterFirst int64
	require.NoError(t, db.Table("documents").Where("type = ?", "ITEM_MAKE").Count(&countAfterFirst).Error)

	require.NoError(t, processor.Register(storage, provider))

	var countAfterSecond int64
	require.NoError(t, db.Table("documents").Where("type = ?", "ITEM_MAKE").Count(&countAfterSecond).Error)

	assert.Equal(t, countAfterFirst, countAfterSecond)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/item-makes/1082002", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	data := response["data"].(map[string]interface{})
	attributes := data["attributes"].(map[string]interface{})
	assert.Equal(t, float64(30), attributes["reqLevel"])
	assert.Equal(t, float64(1200), attributes["meso"])
}
