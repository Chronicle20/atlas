package data

import (
	"atlas-data/canonical"
	"context"
	"encoding/json"
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

type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null"`
	Type       string          `gorm:"not null"`
	DocumentId uint32          `gorm:"not null"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e testDocumentEntity) TableName() string { return "documents" }

type testServerInfo struct{}

func (testServerInfo) GetBaseURL() string { return "" }
func (testServerInfo) GetPrefix() string  { return "/api/" }

func setupStatusDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(logrus.StandardLogger(), logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		}),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testDocumentEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func setupStatusRouter(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	routeInit := InitResource(db)(testServerInfo{})
	routeInit(router, l)
	return router
}

func tenantRequest(tenantId uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/data/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func TestGetDataStatus_Empty(t *testing.T) {
	db := setupStatusDB(t)
	t.Cleanup(func() {
		db.Exec("DELETE FROM documents")
	})
	router := setupStatusRouter(db)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, tenantRequest(uuid.New()))

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				DocumentCount int64   `json:"documentCount"`
				UpdatedAt     *string `json:"updatedAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "dataStatus", resp.Data.Type)
	assert.Equal(t, int64(0), resp.Data.Attributes.DocumentCount)
	assert.Nil(t, resp.Data.Attributes.UpdatedAt)
}

func TestGetDataStatus_SharedScope(t *testing.T) {
	db := setupStatusDB(t)
	t.Cleanup(func() {
		db.Exec("DELETE FROM documents")
	})

	// Canonical baseline rows are anchored to canonical.TenantUUID, exactly
	// as a scope=shared ingest writes them.
	canonicalId := uuid.MustParse(canonical.TenantUUID)
	canonicalTn, err := tenant.Create(canonicalId, "GMS", 83, 1)
	require.NoError(t, err)
	canonicalCtx := tenant.WithContext(context.Background(), canonicalTn)
	for i := 0; i < 2; i++ {
		e := testDocumentEntity{
			Id:         uuid.New(),
			TenantId:   canonicalId,
			Type:       "ITEM",
			DocumentId: uint32(200 + i),
			Content:    json.RawMessage(`{}`),
		}
		require.NoError(t, db.WithContext(canonicalCtx).Create(&e).Error)
	}

	router := setupStatusRouter(db)
	caller := uuid.New() // an ordinary tenant, NOT the canonical sentinel

	// scope=shared with operator credentials sees the canonical rows even
	// though the calling tenant owns none of them.
	req := tenantRequest(caller)
	req.URL.RawQuery = "scope=shared"
	req.Header.Set("X-Atlas-Operator", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Id         string `json:"id"`
			Attributes struct {
				DocumentCount int64 `json:"documentCount"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Data.Attributes.DocumentCount)
	assert.Equal(t, canonical.TenantUUID, resp.Data.Id)

	// Default (tenant) scope for the same caller sees none of them.
	w = httptest.NewRecorder()
	router.ServeHTTP(w, tenantRequest(caller))
	require.Equal(t, http.StatusOK, w.Code)
	var tenantResp struct {
		Data struct {
			Attributes struct {
				DocumentCount int64 `json:"documentCount"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tenantResp))
	assert.Equal(t, int64(0), tenantResp.Data.Attributes.DocumentCount)
}

func TestGetDataStatus_SharedScope_RequiresOperator(t *testing.T) {
	db := setupStatusDB(t)
	router := setupStatusRouter(db)

	req := tenantRequest(uuid.New())
	req.URL.RawQuery = "scope=shared" // no X-Atlas-Operator header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetDataStatus_InvalidScope(t *testing.T) {
	db := setupStatusDB(t)
	router := setupStatusRouter(db)

	req := tenantRequest(uuid.New())
	req.URL.RawQuery = "scope=bogus"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDataStatus_Populated(t *testing.T) {
	db := setupStatusDB(t)
	t.Cleanup(func() {
		db.Exec("DELETE FROM documents")
	})

	tenantId := uuid.New()
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	for i := 0; i < 3; i++ {
		e := testDocumentEntity{
			Id:         uuid.New(),
			TenantId:   tenantId,
			Type:       "ITEM",
			DocumentId: uint32(100 + i),
			Content:    json.RawMessage(`{}`),
		}
		require.NoError(t, db.WithContext(ctx).Create(&e).Error)
	}

	// different tenant should not affect count
	other := uuid.New()
	otherTn, err := tenant.Create(other, "GMS", 83, 1)
	require.NoError(t, err)
	otherCtx := tenant.WithContext(context.Background(), otherTn)
	oth := testDocumentEntity{
		Id:         uuid.New(),
		TenantId:   other,
		Type:       "ITEM",
		DocumentId: 999,
		Content:    json.RawMessage(`{}`),
	}
	require.NoError(t, db.WithContext(otherCtx).Create(&oth).Error)

	router := setupStatusRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/data/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				DocumentCount int64   `json:"documentCount"`
				UpdatedAt     *string `json:"updatedAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "dataStatus", resp.Data.Type)
	assert.Equal(t, tenantId.String(), resp.Data.Id)
	assert.Equal(t, int64(3), resp.Data.Attributes.DocumentCount)
	require.NotNil(t, resp.Data.Attributes.UpdatedAt)
	_, err = time.Parse(time.RFC3339, *resp.Data.Attributes.UpdatedAt)
	assert.NoError(t, err)
}
