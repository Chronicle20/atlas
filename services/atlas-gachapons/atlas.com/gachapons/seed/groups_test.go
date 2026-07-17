package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if err := gachapon.Migration(db); err != nil {
		t.Fatalf("gachapon.Migration: %v", err)
	}
	if err := item.Migration(db); err != nil {
		t.Fatalf("item.Migration: %v", err)
	}
	if err := global.Migration(db); err != nil {
		t.Fatalf("global.Migration: %v", err)
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

// TestInitResource_SeedRouteAccepted verifies that POST /gachapons/seed is registered
// and returns 202 Accepted (background goroutine spawned; result not awaited).
func TestInitResource_SeedRouteAccepted(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/gachapons/seed", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /gachapons/seed: got %d, want %d (body: %s)",
			w.Code, http.StatusAccepted, w.Body.String())
	}
}

// TestInitResource_StatusRouteOK verifies that GET /gachapons/seed/status is registered
// and returns 200 with a body containing "catalogRevision".
func TestInitResource_StatusRouteOK(t *testing.T) {
	db := newGroupsTestDB(t)
	router := newGroupsTestRouter(t, db)

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/gachapons/seed/status", nil)
	req.Header.Set(tenant.ID, te.Id().String())
	req.Header.Set(tenant.Region, te.Region())
	req.Header.Set(tenant.MajorVersion, fmt.Sprintf("%d", te.MajorVersion()))
	req.Header.Set(tenant.MinorVersion, fmt.Sprintf("%d", te.MinorVersion()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /gachapons/seed/status: got %d, want %d (body: %s)",
			w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if body == "" {
		t.Error("GET /gachapons/seed/status: empty body")
	}
	if !strings.Contains(body, "catalogRevision") {
		t.Errorf("GET /gachapons/seed/status: body missing 'catalogRevision': %s", body)
	}
}

// TestSeed_IncubatorKindAndWeightRoundTrip verifies that seeding a catalog
// entry with attributes.kind = "incubator" and weighted items produces a
// gachapon whose Kind() is "incubator" and items whose Weight() round-trips
// from the catalog file through Decode/Build/BulkCreate into the DB.
func TestSeed_IncubatorKindAndWeightRoundTrip(t *testing.T) {
	db := newGroupsTestDB(t)
	// The gachapon and item subdomains BulkCreate concurrently inside
	// seeder.Seed (one errgroup goroutine per subdomain, each in its own
	// transaction). A shared-cache sqlite :memory: DB serializes writers
	// via table locks rather than queueing connections, so two concurrent
	// writer transactions can surface as a real "database table is locked"
	// error instead of blocking. Capping the pool to one connection
	// serializes the transactions the way a real Postgres connection pool
	// would, without changing what's under test.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	l, _ := test.NewNullLogger()

	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	root := t.TempDir()
	gachaponsDir := filepath.Join(root, "gms", "83_1", "gachapons")
	if err := os.MkdirAll(gachaponsDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	catalogJSON := `{
  "data": {
    "type": "gachapon",
    "id": "4170099",
    "attributes": {
      "name": "Pigmy Egg Test",
      "kind": "incubator",
      "npcIds": [],
      "commonWeight": 0,
      "uncommonWeight": 0,
      "rareWeight": 0,
      "items": [
        {
          "itemId": 1002000,
          "quantity": 1,
          "tier": "common",
          "weight": 4
        },
        {
          "itemId": 2040000,
          "quantity": 1,
          "tier": "common",
          "weight": 10
        }
      ]
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(gachaponsDir, "gachapon-4170099.json"), []byte(catalogJSON), 0o644); err != nil {
		t.Fatalf("write fixture catalog file: %v", err)
	}

	src := seeder.NewFilesystemCatalogSource("X_NO_ENV_SET", root)
	g := seeder.Group{
		Name:      "gachapons",
		URLPrefix: "/gachapons",
		Subdomains: []seeder.SubdomainAny{
			seeder.AdaptSubdomain[gachapon.GachaponAttributes, gachapon.Model](gachapon.Subdomain{}),
			seeder.AdaptSubdomain[gachapon.GachaponAttributes, item.Model](item.Subdomain{}),
		},
	}

	ctx := tenant.WithContext(context.Background(), te)
	if _, err := seeder.Seed(ctx, db, src, g); err != nil {
		t.Fatalf("seeder.Seed: %v", err)
	}

	gm, err := gachapon.NewProcessor(l, ctx, db).GetById("4170099")
	if err != nil {
		t.Fatalf("gachapon GetById(4170099): %v", err)
	}
	if gm.Kind() != "incubator" {
		t.Errorf("Kind() = %q, want %q", gm.Kind(), "incubator")
	}

	items, err := item.NewProcessor(l, ctx, db).GetByGachaponId("4170099")()
	if err != nil {
		t.Fatalf("item GetByGachaponId(4170099): %v", err)
	}
	weights := make(map[uint32]uint32, len(items))
	for _, it := range items {
		weights[it.ItemId()] = it.Weight()
	}
	if w, ok := weights[1002000]; !ok || w != 4 {
		t.Errorf("item 1002000 Weight() = %d (present=%v), want 4", w, ok)
	}
	if w, ok := weights[2040000]; !ok || w != 10 {
		t.Errorf("item 2040000 Weight() = %d (present=%v), want 10", w, ok)
	}
}
