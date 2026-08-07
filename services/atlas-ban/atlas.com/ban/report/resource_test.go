package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupReportRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, body []byte, tenantId uuid.UUID) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
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

func seedReport(t *testing.T, db *gorm.DB, tenantId uuid.UUID, status Status) Entity {
	t.Helper()
	e := &Entity{
		Id:           uuid.New(),
		TenantId:     tenantId,
		Kind:         string(KindClaim),
		ReporterId:   1001,
		ReporterName: "Reporter",
		AccusedId:    2002,
		AccusedName:  "Accused",
		ReasonType:   3,
		Description:  "desc",
		Status:       string(status),
	}
	require.NoError(t, db.Create(e).Error)
	return *e
}

// TestUpdateReportStatusHandler drives PATCH /reports/{reportId} through the
// real resource router (InitResource) against an in-memory tenant-scoped DB,
// covering all four documented status codes: 200 echo, 400 id-mismatch, 400
// invalid status, and 404 unknown id.
func TestUpdateReportStatusHandler(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupReportRouter(db))
	defer srv.Close()

	t.Run("SuccessEchoesUpdatedResource", func(t *testing.T) {
		e := seedReport(t, db, tenantId, StatusOpen)

		body := []byte(fmt.Sprintf(`{"data":{"type":"reports","id":%q,"attributes":{"status":"reviewed"}}}`, e.Id.String()))
		url := fmt.Sprintf("%s/reports/%s", srv.URL, e.Id.String())
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.NotNil(t, doc.Data.DataObject)
		assert.Equal(t, e.Id.String(), doc.Data.DataObject.ID)

		var attrs map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
		assert.Equal(t, "reviewed", attrs["status"])

		// Persisted change must survive a fresh read.
		var reread Entity
		require.NoError(t, db.Where("id = ?", e.Id).First(&reread).Error)
		assert.Equal(t, "reviewed", reread.Status)
	})

	t.Run("IdMismatchIsBadRequest", func(t *testing.T) {
		e := seedReport(t, db, tenantId, StatusOpen)
		otherId := uuid.New()

		body := []byte(fmt.Sprintf(`{"data":{"type":"reports","id":%q,"attributes":{"status":"reviewed"}}}`, otherId.String()))
		url := fmt.Sprintf("%s/reports/%s", srv.URL, e.Id.String())
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var reread Entity
		require.NoError(t, db.Where("id = ?", e.Id).First(&reread).Error)
		assert.Equal(t, string(StatusOpen), reread.Status, "mismatched PATCH must not mutate the row")
	})

	t.Run("InvalidStatusIsBadRequest", func(t *testing.T) {
		e := seedReport(t, db, tenantId, StatusOpen)

		body := []byte(fmt.Sprintf(`{"data":{"type":"reports","id":%q,"attributes":{"status":"bogus"}}}`, e.Id.String()))
		url := fmt.Sprintf("%s/reports/%s", srv.URL, e.Id.String())
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var reread Entity
		require.NoError(t, db.Where("id = ?", e.Id).First(&reread).Error)
		assert.Equal(t, string(StatusOpen), reread.Status, "invalid status PATCH must not mutate the row")
	})

	t.Run("UnknownIdIsNotFound", func(t *testing.T) {
		unknownId := uuid.New()

		body := []byte(fmt.Sprintf(`{"data":{"type":"reports","id":%q,"attributes":{"status":"reviewed"}}}`, unknownId.String()))
		url := fmt.Sprintf("%s/reports/%s", srv.URL, unknownId.String())
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestGetReportsHandler covers the list endpoint and its ?status= filter,
// plus the 400 on an invalid status value.
func TestGetReportsHandler(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedReport(t, db, tenantId, StatusOpen)
	seedReport(t, db, tenantId, StatusOpen)
	seedReport(t, db, tenantId, StatusReviewed)

	srv := httptest.NewServer(setupReportRouter(db))
	defer srv.Close()

	t.Run("NoFilterReturnsAll", func(t *testing.T) {
		url := fmt.Sprintf("%s/reports/", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 3)
	})

	t.Run("StatusFilterNarrowsResults", func(t *testing.T) {
		url := fmt.Sprintf("%s/reports/?status=open", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)
	})

	t.Run("InvalidStatusFilterIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/reports/?status=bogus", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetReportByIdHandler covers GET /reports/{reportId} for both the
// found and not-found cases.
func TestGetReportByIdHandler(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	e := seedReport(t, db, tenantId, StatusOpen)

	srv := httptest.NewServer(setupReportRouter(db))
	defer srv.Close()

	t.Run("Found", func(t *testing.T) {
		url := fmt.Sprintf("%s/reports/%s", srv.URL, e.Id.String())
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.NotNil(t, doc.Data.DataObject)
		assert.Equal(t, e.Id.String(), doc.Data.DataObject.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/reports/%s", srv.URL, uuid.New().String())
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestGetReportByIdHandler_TransientErrorIsNotConflatedWithNotFound is the
// DOM-17/DOM-27 regression: a genuine not-found (unknown id, real
// gorm.ErrRecordNotFound) and a database/transport failure must map to
// different status codes. Prior to the fix, handleGetReportById collapsed
// every non-nil GetById error into 404 — a momentary DB outage would tell a
// GM "no such report" instead of "the backend is broken". This closes the
// backing *sql.DB out from under an already-migrated, seeded database (so
// the query fails with a driver/transport error, never
// gorm.ErrRecordNotFound) and asserts the response is NOT 404.
func TestGetReportByIdHandler_TransientErrorIsNotConflatedWithNotFound(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	e := seedReport(t, db, tenantId, StatusOpen)

	srv := httptest.NewServer(setupReportRouter(db))
	defer srv.Close()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close()) // simulate a DB outage for all subsequent queries

	url := fmt.Sprintf("%s/reports/%s", srv.URL, e.Id.String())
	req := requestWithTenant(http.MethodGet, url, nil, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "a DB outage must not be reported as 404 Not Found")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
