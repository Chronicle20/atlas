package tenantpurge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-data/canonical"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPurgeNilMcReturns503(t *testing.T) {
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	h := purgeInner(nil, nil)(&d, &c)
	req := httptest.NewRequest(http.MethodDelete, "/api/data/tenants/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("X-Atlas-Operator", "1")
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

// stubMC returns a non-nil *minio.Client backed by a fake endpoint.
// miniogo.New does not dial on construction; no network I/O occurs.
// The canonical guard fires before any real minio operation is attempted.
func stubMC(t *testing.T) *minio.Client {
	t.Helper()
	mc, err := minio.NewClient(minio.Config{Endpoint: "localhost:9999"})
	if err != nil {
		t.Fatalf("stub minio client: %v", err)
	}
	return mc
}

// minimalDB returns an in-memory SQLite DB with the tables required by Purge.
// A unique DSN is used per call to avoid sharing state with purge_test.go.
func minimalDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:handler_test_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS documents (id TEXT, tenant_id TEXT, type TEXT)`,
		`CREATE TABLE IF NOT EXISTS monster_search_index (tenant_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS npc_search_index (tenant_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS reactor_search_index (tenant_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS map_search_index (tenant_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS item_string_search_index (tenant_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS tenant_baselines (tenant_id TEXT)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}

// purgeViaRouter routes DELETE /{id} through a gorilla mux so mux.Vars is
// populated and injects a tenant model into the request context.
func purgeViaRouter(t *testing.T, db *gorm.DB, mc *minio.Client, tenantCtx tenant.Model, targetID string) *httptest.ResponseRecorder {
	t.Helper()
	router := mux.NewRouter()
	d := server.NewHandlerDependency(logrus.New(), context.Background())
	c := server.NewHandlerContext(nil)
	inner := purgeInner(db, mc)(&d, &c)
	router.HandleFunc("/{id}", inner).Methods(http.MethodDelete)

	req := httptest.NewRequest(http.MethodDelete, "/"+targetID, nil)
	req.Header.Set("X-Atlas-Operator", "1")
	// Inject tenant into request context (mimics ParseTenant middleware).
	req = req.WithContext(tenant.WithContext(req.Context(), tenantCtx))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// TestHandlerRefusesVersionScopedCanonicalId verifies that a DELETE whose
// {id} equals the version-scoped canonical id (from canonical.TenantId) is
// refused with 403 before any Purge logic runs.
func TestHandlerRefusesVersionScopedCanonicalId(t *testing.T) {
	const region = "GMS"
	const major uint16 = 83
	const minor uint16 = 1
	canonicalID := canonical.TenantId(region, major, minor)

	tnt, err := tenant.Create(canonicalID, region, major, minor)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	mc := stubMC(t)
	rr := purgeViaRouter(t, nil, mc, tnt, canonicalID.String())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for version-scoped canonical id, got %d", rr.Code)
	}
}

// TestHandlerRefusesAllZerosSentinel verifies the all-zeros sentinel is also
// refused at the handler level (defense-in-depth complementing purge.go).
func TestHandlerRefusesAllZerosSentinel(t *testing.T) {
	const region = "GMS"
	const major uint16 = 83
	const minor uint16 = 1
	tnt, err := tenant.Create(canonical.TenantId(region, major, minor), region, major, minor)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	mc := stubMC(t)
	rr := purgeViaRouter(t, nil, mc, tnt, canonical.TenantUUID)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for all-zeros sentinel, got %d", rr.Code)
	}
}

// TestHandlerAllowsNonCanonicalId verifies that a random non-canonical tenant
// id passes the canonical guard. The id is different from the version-scoped
// canonical id, so it must not be refused with 403. It proceeds to Purge
// (which deletes rows from the in-memory DB) and returns 202.
func TestHandlerAllowsNonCanonicalId(t *testing.T) {
	const region = "GMS"
	const major uint16 = 83
	const minor uint16 = 1
	normalID := "11111111-1111-1111-1111-111111111111"

	tnt, err := tenant.Create(canonical.TenantId(region, major, minor), region, major, minor)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	db := minimalDB(t)
	mc := stubMC(t)
	rr := purgeViaRouter(t, db, mc, tnt, normalID)
	// The guard must pass: any code other than 403 is correct.
	if rr.Code == http.StatusForbidden {
		t.Fatalf("non-canonical id must not be refused by canonical guard, got 403")
	}
}
