package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testSrvInfo struct{}

func (testSrvInfo) GetBaseURL() string { return "" }
func (testSrvInfo) GetPrefix() string  { return "/api/" }

func newGroupsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := monsterdrop.Migration(db); err != nil {
		t.Fatalf("monsterdrop.Migration: %v", err)
	}
	if err := continentdrop.Migration(db); err != nil {
		t.Fatalf("continentdrop.Migration: %v", err)
	}
	if err := reactordrop.Migration(db); err != nil {
		t.Fatalf("reactordrop.Migration: %v", err)
	}
	if err := db.AutoMigrate(&seeder.SeedState{}); err != nil {
		t.Fatalf("seeder.SeedState migration: %v", err)
	}
	return db
}

func newGroupsTestRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	l, _ := test.NewNullLogger()
	router := mux.NewRouter()
	routeInit := InitResource(testSrvInfo{})(db)
	if routeInit == nil {
		t.Fatal("InitResource(db) returned nil RouteInitializer")
	}
	routeInit(router, l)
	return router
}

// TestInitResource_SeedRouteAccepted verifies that POST /drops/seed is registered
// and returns 202 Accepted (background goroutine spawned; result not awaited).
func TestInitResource_SeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drops/seed", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /drops/seed: got %d, want %d (body: %s)",
			w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitResource_StatusRouteOK verifies that GET /drops/seed/status is registered
// and returns 200 with a body containing "catalogRevision".
func TestInitResource_StatusRouteOK(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drops/seed/status", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /drops/seed/status: got %d, want %d (body: %s)",
			w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if body == "" {
		t.Error("GET /drops/seed/status: empty body")
	}
	if !strings.Contains(body, "catalogRevision") {
		t.Errorf("GET /drops/seed/status: body missing 'catalogRevision': %s", body)
	}
}
