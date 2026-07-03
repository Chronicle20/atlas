package history

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupHistoryRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedHistoryEntry(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint64, accountId uint32, ip string, hwid string, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{
		ID:          id,
		TenantId:    tenantId,
		AccountId:   accountId,
		AccountName: fmt.Sprintf("account-%d", accountId),
		IPAddress:   ip,
		HWID:        hwid,
		Success:     true,
		CreatedAt:   createdAt,
	}).Error)
}

// TestGetHistoryPaginates drives GET /history/ through the real resource
// router (InitResource) against an in-memory tenant-scoped DB, verifying the
// JSON:API paginated envelope, 400 on invalid paging params, and that
// created_at-desc ordering survives the pagination conversion.
func TestGetHistoryPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	now := time.Now()
	seedHistoryEntry(t, db, tenantId, 1, 1, "10.0.0.1", "HWID1", now.Add(-2*time.Hour))
	seedHistoryEntry(t, db, tenantId, 2, 2, "10.0.0.2", "HWID2", now.Add(-1*time.Hour))
	seedHistoryEntry(t, db, tenantId, 3, 3, "10.0.0.3", "HWID3", now)

	srv := httptest.NewServer(setupHistoryRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoNewestFirst", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		// CRITICAL — caller-order preservation: entitiesByTenant's
		// Order("created_at desc") must still be the effective ordering on
		// page 1 (accountId 3 is newest, seeded last).
		var first, second RestModel
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, 3, first.AccountId, "page 1 item 0 should be the newest row (created_at desc)")
		assert.EqualValues(t, 2, second.AccountId, "page 1 item 1 should be the second-newest row")

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

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

	// IPFilterStillUnpaginated is a regression guard for the ?ip=/?hwid=
	// branches, which are out of this task's scope and must keep their
	// pre-existing unpaginated (bare-array, no meta/links) response shape.
	t.Run("IPFilterStillUnpaginated", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/?ip=10.0.0.1", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 1)
		assert.Nil(t, doc.Meta, "ip-filtered branch must remain unpaginated (no meta envelope)")
	})
}

// TestGetHistoryByAccountIdPaginates drives GET /history/accounts/{accountId}
// through the real resource router, verifying the same paginated envelope
// plus created_at-desc ordering preserved on page 1.
func TestGetHistoryByAccountIdPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	now := time.Now()
	seedHistoryEntry(t, db, tenantId, 1, 42, "10.0.0.1", "HWID1", now.Add(-2*time.Hour))
	seedHistoryEntry(t, db, tenantId, 2, 42, "10.0.0.2", "HWID2", now.Add(-1*time.Hour))
	seedHistoryEntry(t, db, tenantId, 3, 42, "10.0.0.3", "HWID3", now)
	// A different account's row must not appear in account 42's page.
	seedHistoryEntry(t, db, tenantId, 4, 99, "10.0.0.9", "HWID9", now)

	srv := httptest.NewServer(setupHistoryRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoNewestFirst", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/accounts/42?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		// CRITICAL — caller-order preservation for the per-account query.
		var first, second RestModel
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, "10.0.0.3", first.IPAddress, "page 1 item 0 should be the newest row (created_at desc)")
		assert.EqualValues(t, "10.0.0.2", second.IPAddress, "page 1 item 1 should be the second-newest row")

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude account 99's row")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/accounts/42?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/accounts/42?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/history/accounts/42?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

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
