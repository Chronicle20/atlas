package compartment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type paginateTestServerInformation struct{}

func (t *paginateTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *paginateTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &paginateTestServerInformation{}

// compartmentMigrationSqlite creates the cash_compartments table directly
// (Migration's AutoMigrate emits a `DEFAULT uuid_generate_v4()` column
// default, which is PostgreSQL-specific and fails sqlite's DDL parser --
// "near '(': syntax error"). Tests always supply an explicit Id, so the
// default is never actually needed.
func compartmentMigrationSqlite(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS cash_compartments (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		account_id INTEGER NOT NULL,
		type INTEGER NOT NULL,
		capacity INTEGER NOT NULL DEFAULT 55
	)`).Error
}

func setupCompartmentRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&paginateTestServerInformation{})(db)
	ri(r, l)
	return r
}

func requestCompartmentsWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedCompartment(t *testing.T, db *gorm.DB, tenantId uuid.UUID, accountId uint32, type_ byte) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{
		Id:        uuid.New(),
		TenantId:  tenantId,
		AccountId: accountId,
		Type:      type_,
		Capacity:  55,
	}).Error)
}

// TestGetCompartmentsPaginates drives GET
// /accounts/{accountId}/cash-shop/inventory/compartments?type= (empty type
// value -- the "all compartments" branch) through the real resource router,
// verifying the JSON:API paginated envelope, 400 on invalid paging params,
// and empty-page handling past the last page. Also confirms another
// account's compartments are excluded from the total.
func TestGetCompartmentsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, compartmentMigrationSqlite)
	// sqlite's ":memory:" DSN (no cache=shared) hands out a brand-new,
	// empty in-memory database per connection; the default pool can open
	// more than one connection under concurrent httptest requests, which
	// intermittently loses the seeded table ("no such table:
	// cash_compartments"). Pin the pool to one connection so every request
	// in this test hits the same in-memory database.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	tenantId := uuid.New()

	seedCompartment(t, db, tenantId, 1, 1)
	seedCompartment(t, db, tenantId, 1, 2)
	seedCompartment(t, db, tenantId, 1, 3)
	seedCompartment(t, db, tenantId, 2, 1)

	srv := httptest.NewServer(setupCompartmentRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/1/cash-shop/inventory/compartments?type=&page[number]=1&page[size]=2", srv.URL)
		req := requestCompartmentsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude account 2's compartment")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/1/cash-shop/inventory/compartments?type=&page[size]=0", srv.URL)
		req := requestCompartmentsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/1/cash-shop/inventory/compartments?type=&limit=5", srv.URL)
		req := requestCompartmentsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/1/cash-shop/inventory/compartments?type=&page[number]=99&page[size]=2", srv.URL)
		req := requestCompartmentsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}
