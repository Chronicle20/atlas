package parcel_test

import (
	"atlas-parcel/parcel"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

// newParcelTestDB is databasetest.NewInMemoryTenantDB with GORM's default
// logger silenced — without this, the "get by id missing" subtest's
// deliberate not-found lookup prints a stray "record not found" line to
// stdout (matches the fix processor_test.go already applies for the same
// reason).
func newParcelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, parcel.Migration)
	db.Logger = logger.Default.LogMode(logger.Silent)
	return db
}

func newParcelServer(db *gorm.DB) *httptest.Server {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	parcel.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, tenantId uuid.UUID, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// withTenantBody is withTenant plus a JSON:API-marshaled body, for the PATCH
// discard route.
func withTenantBody(t *testing.T, tenantId uuid.UUID, method, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	req.Header.Set("Content-Type", "application/json")
	return req
}

// discardBody marshals a DiscardRestModel the way atlas-channel's PATCH
// request does (github.com/Chronicle20/atlas/libs/atlas-rest/requests'
// PatchRequest uses jsonapi.Marshal on the same interface).
func discardBody(t *testing.T, parcelId uuid.UUID, recipientId uint32) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(parcel.DiscardRestModel{Id: parcelId.String(), RecipientId: recipientId})
	require.NoError(t, err)
	return b
}

// notifyBody marshals a NotifyRestModel the way atlas-channel's PATCH
// /parcels/{id}/notify request does — see requests.go's notifyRestModel in
// atlas-channel, whose only content is the resource identifier.
func notifyBody(t *testing.T, parcelId uuid.UUID) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(parcel.NotifyRestModel{Id: parcelId.String()})
	require.NoError(t, err)
	return b
}

// seedParcel builds and persists one pending parcel under tenantId, returning
// the created Model.
func seedParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, senderId, recipientId uint32) parcel.Model {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	m, err := parcel.NewBuilder().
		SetId(uuid.New()).
		SetWorldId(0).
		SetSenderId(senderId).
		SetSenderAccountId(1).
		SetSenderName("Sender").
		SetRecipientId(recipientId).
		SetRecipientAccountId(2).
		SetStatus(parcel.StatusPending).
		SetCreatedAt(now).
		SetReceivableAt(now).
		SetExpiresAt(now.Add(parcel.ExpiryWindow)).
		Build()
	require.NoError(t, err)

	created, err := parcel.Create(db.WithContext(databasetest.TenantContext(tenantId)))(m)
	require.NoError(t, err)
	return created
}

// envelope is the minimal JSON:API shape the assertions below need.
type envelope struct {
	Data json.RawMessage `json:"data"`
}

type dataList struct {
	Data []json.RawMessage `json:"data"`
}

type resourceIdentifier struct {
	Id string `json:"id"`
}

type parcelStatusAttributes struct {
	Attributes struct {
		InFlight bool `json:"inFlight"`
	} `json:"attributes"`
}

// TestParcelResource exercises the four parcel routes end to end over HTTP,
// against an in-memory tenant-scoped DB. It is table-driven over run: every
// case hits a different route/method/body combination with a distinct
// assertion set, so the scenario itself is carried as a closure rather than
// shared data fields.
func TestParcelResource(t *testing.T) {
	client := &http.Client{}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "list by recipient", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			seedParcel(t, db, tid, 900, 100)
			seedParcel(t, db, tid, 901, 100)
			seedParcel(t, db, tid, 902, 200)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/parcels?filter[recipientId]=100&filter[worldId]=0&filter[status]=pending", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env dataList
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			require.Len(t, env.Data, 2)
		}},
		// list by recipient without filter[worldId] must be a clean 400,
		// never a silent default to world 0 — world 0 is an ordinary real
		// world, not a sentinel, and a tenant has many worlds.
		{name: "list by recipient missing worldId", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			seedParcel(t, db, tid, 900, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/parcels?filter[recipientId]=100&filter[status]=pending", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}},
		{name: "list by sender", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			seedParcel(t, db, tid, 300, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/parcels?filter[senderId]=300&filter[status]=pending", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env dataList
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			require.Len(t, env.Data, 1)
		}},
		{name: "get by id", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			created := seedParcel(t, db, tid, 300, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/parcels/%s", srv.URL, created.Id().String())))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env envelope
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			var ri resourceIdentifier
			require.NoError(t, json.Unmarshal(env.Data, &ri))
			require.Equal(t, created.Id().String(), ri.Id)
		}},
		{name: "get by id missing", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()

			srv := newParcelServer(db)
			defer srv.Close()

			missing := uuid.New().String()
			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/parcels/%s", srv.URL, missing)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		}},
		{name: "parcel-status true", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			seedParcel(t, db, tid, 100, 200)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/characters/100/parcel-status", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env envelope
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			var attrs parcelStatusAttributes
			require.NoError(t, json.Unmarshal(env.Data, &attrs))
			require.True(t, attrs.Attributes.InFlight)
		}},
		{name: "parcel-status false", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tid, http.MethodGet, fmt.Sprintf("%s/characters/100/parcel-status", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env envelope
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			var attrs parcelStatusAttributes
			require.NoError(t, json.Unmarshal(env.Data, &attrs))
			require.False(t, attrs.Attributes.InFlight)
		}},
		{name: "discard", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			created := seedParcel(t, db, tid, 300, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			body := discardBody(t, created.Id(), 100)
			resp, err := client.Do(withTenantBody(t, tid, http.MethodPatch, fmt.Sprintf("%s/parcels/%s", srv.URL, created.Id().String()), body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			m, err := parcel.NewProcessor(logrus.New(), databasetest.TenantContext(tid), db).GetById(created.Id())
			require.NoError(t, err)
			require.Equal(t, parcel.StatusDiscarded, m.Status())
		}},
		{name: "discard not the recipient", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			created := seedParcel(t, db, tid, 300, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			body := discardBody(t, created.Id(), 999)
			resp, err := client.Do(withTenantBody(t, tid, http.MethodPatch, fmt.Sprintf("%s/parcels/%s", srv.URL, created.Id().String()), body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusConflict, resp.StatusCode)

			m, err := parcel.NewProcessor(logrus.New(), databasetest.TenantContext(tid), db).GetById(created.Id())
			require.NoError(t, err)
			require.Equal(t, parcel.StatusPending, m.Status())
		}},
		{name: "discard missing", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()

			srv := newParcelServer(db)
			defer srv.Close()

			missing := uuid.New()
			body := discardBody(t, missing, 100)
			resp, err := client.Do(withTenantBody(t, tid, http.MethodPatch, fmt.Sprintf("%s/parcels/%s", srv.URL, missing.String()), body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		}},
		{name: "notify", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()
			created := seedParcel(t, db, tid, 300, 100)
			require.Nil(t, created.LastNotified())

			srv := newParcelServer(db)
			defer srv.Close()

			body := notifyBody(t, created.Id())
			resp, err := client.Do(withTenantBody(t, tid, http.MethodPatch, fmt.Sprintf("%s/parcels/%s/notify", srv.URL, created.Id().String()), body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			m, err := parcel.NewProcessor(logrus.New(), databasetest.TenantContext(tid), db).GetById(created.Id())
			require.NoError(t, err)
			require.NotNil(t, m.LastNotified())
		}},
		{name: "notify missing", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tid := uuid.New()

			srv := newParcelServer(db)
			defer srv.Close()

			missing := uuid.New()
			body := notifyBody(t, missing)
			resp, err := client.Do(withTenantBody(t, tid, http.MethodPatch, fmt.Sprintf("%s/parcels/%s/notify", srv.URL, missing.String()), body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		}},
		{name: "tenant isolation", run: func(t *testing.T) {
			db := newParcelTestDB(t)
			tidA, tidB := uuid.New(), uuid.New()
			seedParcel(t, db, tidA, 900, 100)

			srv := newParcelServer(db)
			defer srv.Close()

			resp, err := client.Do(withTenant(t, tidB, http.MethodGet, fmt.Sprintf("%s/parcels?filter[recipientId]=100&filter[worldId]=0", srv.URL)))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var env dataList
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
			require.Len(t, env.Data, 0)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
