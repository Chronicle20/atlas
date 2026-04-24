package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type statusSrvInfo struct{}

func (s statusSrvInfo) GetBaseURL() string { return "" }
func (s statusSrvInfo) GetPrefix() string  { return "/api/" }

type statusEnvelopeJSON struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			MonsterDropCount   int64   `json:"monsterDropCount"`
			ContinentDropCount int64   `json:"continentDropCount"`
			ReactorDropCount   int64   `json:"reactorDropCount"`
			UpdatedAt          *string `json:"updatedAt"`
		} `json:"attributes"`
	} `json:"data"`
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	if err := monsterdrop.Migration(db); err != nil {
		t.Fatalf("Failed to migrate monster drops: %v", err)
	}
	if err := continentdrop.Migration(db); err != nil {
		t.Fatalf("Failed to migrate continent drops: %v", err)
	}
	if err := reactordrop.Migration(db); err != nil {
		t.Fatalf("Failed to migrate reactor drops: %v", err)
	}

	// Ensure clean tables between tests (shared cache means state can leak).
	if err := db.Exec("DELETE FROM monster_drops").Error; err != nil {
		t.Fatalf("Failed to reset monster_drops: %v", err)
	}
	if err := db.Exec("DELETE FROM continent_drops").Error; err != nil {
		t.Fatalf("Failed to reset continent_drops: %v", err)
	}
	if err := db.Exec("DELETE FROM reactor_drops").Error; err != nil {
		t.Fatalf("Failed to reset reactor_drops: %v", err)
	}

	return db
}

func setupStatusRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(statusSrvInfo{})
	routeInit := initFn(db)
	routeInit(router, l)
	return router
}

func statusReq(tenantId uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/drops/seed/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func insertMonster(t *testing.T, db *gorm.DB, tenantId uuid.UUID, monsterId, itemId uint32) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
		tenantId, monsterId, itemId, 1, 1, 0, 10000,
	).Error; err != nil {
		t.Fatalf("seed monster: %v", err)
	}
}

func insertContinent(t *testing.T, db *gorm.DB, tenantId uuid.UUID, continentId int32, itemId uint32) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
		tenantId, continentId, itemId, 1, 1, 0, 10000,
	).Error; err != nil {
		t.Fatalf("seed continent: %v", err)
	}
}

func insertReactor(t *testing.T, db *gorm.DB, tenantId uuid.UUID, reactorId, itemId uint32) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO reactor_drops (tenant_id, reactor_id, item_id, quest_id, chance) VALUES (?, ?, ?, ?, ?)",
		tenantId, reactorId, itemId, 0, 10000,
	).Error; err != nil {
		t.Fatalf("seed reactor: %v", err)
	}
}

func TestStatusHandler_Empty(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenantId := uuid.New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, w.Body.String())
	}
	if env.Data.Type != "dropsSeedStatus" {
		t.Errorf("type = %q, want dropsSeedStatus", env.Data.Type)
	}
	if env.Data.Id != tenantId.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenantId.String())
	}
	if env.Data.Attributes.MonsterDropCount != 0 {
		t.Errorf("monsterDropCount = %d, want 0", env.Data.Attributes.MonsterDropCount)
	}
	if env.Data.Attributes.ContinentDropCount != 0 {
		t.Errorf("continentDropCount = %d, want 0", env.Data.Attributes.ContinentDropCount)
	}
	if env.Data.Attributes.ReactorDropCount != 0 {
		t.Errorf("reactorDropCount = %d, want 0", env.Data.Attributes.ReactorDropCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestStatusHandler_Populated(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenantId := uuid.New()
	insertMonster(t, db, tenantId, 100100, 2000000)
	insertMonster(t, db, tenantId, 100101, 2000001)
	insertContinent(t, db, tenantId, 0, 2000002)
	insertReactor(t, db, tenantId, 1000, 2000003)
	insertReactor(t, db, tenantId, 1001, 2000004)
	insertReactor(t, db, tenantId, 1002, 2000005)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.MonsterDropCount != 2 {
		t.Errorf("monsterDropCount = %d, want 2", env.Data.Attributes.MonsterDropCount)
	}
	if env.Data.Attributes.ContinentDropCount != 1 {
		t.Errorf("continentDropCount = %d, want 1", env.Data.Attributes.ContinentDropCount)
	}
	if env.Data.Attributes.ReactorDropCount != 3 {
		t.Errorf("reactorDropCount = %d, want 3", env.Data.Attributes.ReactorDropCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null (no updated_at columns), got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestStatusHandler_TenantIsolation(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	insertMonster(t, db, tenant1, 100100, 2000000)
	insertMonster(t, db, tenant2, 100100, 2000000)
	insertMonster(t, db, tenant2, 100101, 2000001)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenant1))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.MonsterDropCount != 1 {
		t.Errorf("monsterDropCount for tenant1 = %d, want 1", env.Data.Attributes.MonsterDropCount)
	}
	if env.Data.Id != tenant1.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenant1.String())
	}
}

