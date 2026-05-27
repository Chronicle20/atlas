package definition

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
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testGroupsSrvInfo struct{}

func (testGroupsSrvInfo) GetBaseURL() string { return "" }
func (testGroupsSrvInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = testGroupsSrvInfo{}

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

// TestInitSeedResource_SeedRouteAccepted verifies that POST /party-quests/definitions/seed
// is registered and returns 202 Accepted.
func TestInitSeedResource_SeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/party-quests/definitions/seed", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /party-quests/definitions/seed: got %d, want %d (body: %s)",
			w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitSeedResource_StatusRouteOK verifies that GET /party-quests/definitions/seed/status
// is registered and returns 200 with a body containing "catalogRevision".
func TestInitSeedResource_StatusRouteOK(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/party-quests/definitions/seed/status", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /party-quests/definitions/seed/status: got %d, want %d (body: %s)",
			w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if body == "" {
		t.Error("GET /party-quests/definitions/seed/status: empty body")
	}
	if !containsStr(body, "catalogRevision") {
		t.Errorf("GET /party-quests/definitions/seed/status: body missing 'catalogRevision': %s", body)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
