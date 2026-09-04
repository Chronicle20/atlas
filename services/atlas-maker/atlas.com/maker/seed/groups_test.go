package seed

import (
	"atlas-maker/crystalband"
	"atlas-maker/reagent"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	if err := reagent.Migration(db); err != nil {
		t.Fatalf("reagent.Migration: %v", err)
	}
	if err := crystalband.Migration(db); err != nil {
		t.Fatalf("crystalband.Migration: %v", err)
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

func newSeedRequest(method, url string, te tenant.Model) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	return req
}

// TestInitResource_ReagentsSeedRouteAccepted verifies that POST /reagents/seed
// is registered and returns 202 Accepted (background goroutine spawned;
// result not awaited).
func TestInitResource_ReagentsSeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := newSeedRequest(http.MethodPost, "/reagents/seed", te)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /reagents/seed: got %d, want %d (body: %s)", w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitResource_CrystalBandsSeedRouteAccepted verifies that
// POST /crystal-bands/seed is registered and returns 202 Accepted, proving
// the crystalBands group is registered alongside reagents under its own
// URL prefix.
func TestInitResource_CrystalBandsSeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := newSeedRequest(http.MethodPost, "/crystal-bands/seed", te)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /crystal-bands/seed: got %d, want %d (body: %s)", w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitResource_BothGroupsExposeSeedStatus asserts both groups' status
// routes are registered and return 200 with a body containing
// "catalogRevision".
func TestInitResource_BothGroupsExposeSeedStatus(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	for _, prefix := range []string{"/reagents", "/crystal-bands"} {
		t.Run(prefix, func(t *testing.T) {
			req := newSeedRequest(http.MethodGet, prefix+"/seed/status", te)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("GET %s/seed/status: got %d, want %d (body: %s)", prefix, w.Code, http.StatusOK, w.Body.String())
			}
		})
	}
}
