package cashpackage

import (
	"atlas-data/document"
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

// TestCashPackageResourceIntegration exercises the REST surface end to end,
// in particular the /{packageId} detail route: it asserts a request with
// packageId set actually reaches the handler (200 on a hit, 404 on a miss),
// which is exactly the path a mis-wired ParseItemId (hardcoded to the
// "itemId" mux var) breaks by returning 400 unconditionally.
func TestCashPackageResourceIntegration(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCashPackageData(t, db, tenantId)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("GetCashPackagesEndpoint", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/cashPackages", testServer.URL)
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
		assert.Len(t, data, 2)
	})

	t.Run("GetCashPackageById_Hit", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/cashPackages/9100000", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "cashPackages", data["type"])
		assert.Equal(t, "9100000", data["id"])

		attributes := data["attributes"].(map[string]interface{})
		serialNumbers := attributes["serialNumbers"].([]interface{})
		assert.Len(t, serialNumbers, 2)
	})

	t.Run("GetCashPackageById_Miss", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/cashPackages/9999999", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetCashPackageById_BadId", func(t *testing.T) {
		url := fmt.Sprintf("%s/data/cashPackages/bogus", testServer.URL)
		req := createRequestWithTenant("GET", url, tenantId)

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

func setupTestCashPackageData(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	packages := []RestModel{
		{Id: 9100000, SerialNumbers: []uint32{10000, 10001}},
		{Id: 9100001, SerialNumbers: []uint32{20000}},
	}

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "CASH_PACKAGE")
	for _, p := range packages {
		_, err := storage.Add(ctx)(p)()
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
