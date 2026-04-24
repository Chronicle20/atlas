package script

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type statusSrvInfo struct{}

func (s statusSrvInfo) GetBaseURL() string { return "" }
func (s statusSrvInfo) GetPrefix() string  { return "/api/" }

type seedStatusEnvelopeJSON struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			ScriptCount int64   `json:"scriptCount"`
			UpdatedAt   *string `json:"updatedAt"`
		} `json:"attributes"`
	} `json:"data"`
}

func newSeedStatusTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	if err := MigrateTable(db); err != nil {
		t.Fatalf("Failed to migrate reactor_scripts: %v", err)
	}

	return db
}

func setupSeedStatusRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(statusSrvInfo{})
	routeInit := initFn(db)
	routeInit(router, l)
	return router
}

func seedStatusReq(tenantId uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/reactors/actions/seed/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func insertSeedStatusScript(t *testing.T, db *gorm.DB, tenantId uuid.UUID, reactorId string) {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), te)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)
	m := NewReactorScriptBuilder().
		SetReactorId(reactorId).
		SetDescription("status test script").
		Build()
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create reactor script %s: %v", reactorId, err)
	}
}

func TestSeedStatusHandler_Empty(t *testing.T) {
	db := newSeedStatusTestDB(t)
	router := setupSeedStatusRouter(t, db)

	tenantId := uuid.New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, seedStatusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env seedStatusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, w.Body.String())
	}
	if env.Data.Type != "reactorScriptsSeedStatus" {
		t.Errorf("type = %q, want reactorScriptsSeedStatus", env.Data.Type)
	}
	if env.Data.Id != tenantId.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenantId.String())
	}
	if env.Data.Attributes.ScriptCount != 0 {
		t.Errorf("scriptCount = %d, want 0", env.Data.Attributes.ScriptCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestSeedStatusHandler_Populated(t *testing.T) {
	db := newSeedStatusTestDB(t)
	router := setupSeedStatusRouter(t, db)

	tenantId := uuid.New()
	insertSeedStatusScript(t, db, tenantId, "reactor_a")
	insertSeedStatusScript(t, db, tenantId, "reactor_b")
	insertSeedStatusScript(t, db, tenantId, "reactor_c")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, seedStatusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env seedStatusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.ScriptCount != 3 {
		t.Errorf("scriptCount = %d, want 3", env.Data.Attributes.ScriptCount)
	}
	if env.Data.Attributes.UpdatedAt == nil {
		t.Errorf("updatedAt should be non-nil after inserts")
	}
}

func TestSeedStatusHandler_TenantIsolation(t *testing.T) {
	db := newSeedStatusTestDB(t)
	router := setupSeedStatusRouter(t, db)

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	insertSeedStatusScript(t, db, tenant1, "t1_reactor_a")
	insertSeedStatusScript(t, db, tenant2, "t2_reactor_a")
	insertSeedStatusScript(t, db, tenant2, "t2_reactor_b")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, seedStatusReq(tenant1))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env seedStatusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.ScriptCount != 1 {
		t.Errorf("scriptCount for tenant1 = %d, want 1", env.Data.Attributes.ScriptCount)
	}
	if env.Data.Id != tenant1.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenant1.String())
	}
}
