package script

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

type testGroupsSrvInfo struct{}

func (testGroupsSrvInfo) GetBaseURL() string { return "" }
func (testGroupsSrvInfo) GetPrefix() string  { return "/api/" }

func newGroupsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := MigrateTable(db); err != nil {
		t.Fatalf("MigrateTable: %v", err)
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
	routeInit := InitSeedResource(testGroupsSrvInfo{})(db)
	if routeInit == nil {
		t.Fatal("InitSeedResource(db) returned nil RouteInitializer")
	}
	routeInit(router, l)
	return router
}

// TestInitSeedResource_SeedRouteAccepted verifies that POST /portals/scripts/seed
// is registered and returns 202 Accepted.
func TestInitSeedResource_SeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/portals/scripts/seed", nil)
	req = req.WithContext(tenant.WithContext(req.Context(), te))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /portals/scripts/seed: got %d, want %d (body: %s)",
			w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitSeedResource_StatusRouteOK verifies that GET /portals/scripts/seed/status
// is registered and returns 200 with a body containing "catalogRevision".
func TestInitSeedResource_StatusRouteOK(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/portals/scripts/seed/status", nil)
	req = req.WithContext(tenant.WithContext(req.Context(), te))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /portals/scripts/seed/status: got %d, want %d (body: %s)",
			w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if body == "" {
		t.Error("GET /portals/scripts/seed/status: empty body")
	}
	if !containsPortalStr(body, "catalogRevision") {
		t.Errorf("GET /portals/scripts/seed/status: body missing 'catalogRevision': %s", body)
	}
}

func containsPortalStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
