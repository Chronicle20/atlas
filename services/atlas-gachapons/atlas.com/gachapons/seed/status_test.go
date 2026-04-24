package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
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

type statusEnvelopeJSON struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			GachaponCount   int64   `json:"gachaponCount"`
			ItemCount       int64   `json:"itemCount"`
			GlobalItemCount int64   `json:"globalItemCount"`
			UpdatedAt       *string `json:"updatedAt"`
		} `json:"attributes"`
	} `json:"data"`
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	if err := gachapon.Migration(db); err != nil {
		t.Fatalf("Failed to migrate gachapons: %v", err)
	}
	if err := item.Migration(db); err != nil {
		t.Fatalf("Failed to migrate items: %v", err)
	}
	if err := global.Migration(db); err != nil {
		t.Fatalf("Failed to migrate global items: %v", err)
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
	req := httptest.NewRequest(http.MethodGet, "/gachapons/seed/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedGachapon(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id string) {
	t.Helper()
	l, _ := test.NewNullLogger()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), te)
	m, err := gachapon.NewBuilder(tenantId, id).
		SetName("seed-" + id).
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("build gachapon: %v", err)
	}
	if err := gachapon.NewProcessor(l, ctx, db).Create(m); err != nil {
		t.Fatalf("create gachapon: %v", err)
	}
}

func seedItem(t *testing.T, db *gorm.DB, tenantId uuid.UUID, gachaponId string, itemId uint32) {
	t.Helper()
	l, _ := test.NewNullLogger()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), te)
	m, err := item.NewBuilder(tenantId, 0).
		SetGachaponId(gachaponId).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("build item: %v", err)
	}
	if err := item.NewProcessor(l, ctx, db).Create(m); err != nil {
		t.Fatalf("create item: %v", err)
	}
}

func seedGlobalItem(t *testing.T, db *gorm.DB, tenantId uuid.UUID, itemId uint32) {
	t.Helper()
	l, _ := test.NewNullLogger()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), te)
	m, err := global.NewBuilder(tenantId, 0).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("build global item: %v", err)
	}
	if err := global.NewProcessor(l, ctx, db).Create(m); err != nil {
		t.Fatalf("create global item: %v", err)
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
	if env.Data.Type != "gachaponsSeedStatus" {
		t.Errorf("type = %q, want gachaponsSeedStatus", env.Data.Type)
	}
	if env.Data.Id != tenantId.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenantId.String())
	}
	if env.Data.Attributes.GachaponCount != 0 {
		t.Errorf("gachaponCount = %d, want 0", env.Data.Attributes.GachaponCount)
	}
	if env.Data.Attributes.ItemCount != 0 {
		t.Errorf("itemCount = %d, want 0", env.Data.Attributes.ItemCount)
	}
	if env.Data.Attributes.GlobalItemCount != 0 {
		t.Errorf("globalItemCount = %d, want 0", env.Data.Attributes.GlobalItemCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestStatusHandler_Populated(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenantId := uuid.New()
	seedGachapon(t, db, tenantId, "pop-1")
	seedGachapon(t, db, tenantId, "pop-2")
	seedItem(t, db, tenantId, "pop-1", 2000000)
	seedItem(t, db, tenantId, "pop-1", 2000001)
	seedItem(t, db, tenantId, "pop-1", 2000002)
	seedItem(t, db, tenantId, "pop-2", 2000003)
	seedItem(t, db, tenantId, "pop-2", 2000004)
	seedGlobalItem(t, db, tenantId, 3000000)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.GachaponCount != 2 {
		t.Errorf("gachaponCount = %d, want 2", env.Data.Attributes.GachaponCount)
	}
	if env.Data.Attributes.ItemCount != 5 {
		t.Errorf("itemCount = %d, want 5", env.Data.Attributes.ItemCount)
	}
	if env.Data.Attributes.GlobalItemCount != 1 {
		t.Errorf("globalItemCount = %d, want 1", env.Data.Attributes.GlobalItemCount)
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

	// Tenant 1: 1 gachapon, 2 items, 1 global
	seedGachapon(t, db, tenant1, "iso-1a")
	seedItem(t, db, tenant1, "iso-1a", 2100000)
	seedItem(t, db, tenant1, "iso-1a", 2100001)
	seedGlobalItem(t, db, tenant1, 3100000)

	// Tenant 2: 3 gachapons, 1 item, 2 globals
	seedGachapon(t, db, tenant2, "iso-2a")
	seedGachapon(t, db, tenant2, "iso-2b")
	seedGachapon(t, db, tenant2, "iso-2c")
	seedItem(t, db, tenant2, "iso-2a", 2200000)
	seedGlobalItem(t, db, tenant2, 3200000)
	seedGlobalItem(t, db, tenant2, 3200001)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenant1))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Id != tenant1.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenant1.String())
	}
	if env.Data.Attributes.GachaponCount != 1 {
		t.Errorf("gachaponCount for tenant1 = %d, want 1", env.Data.Attributes.GachaponCount)
	}
	if env.Data.Attributes.ItemCount != 2 {
		t.Errorf("itemCount for tenant1 = %d, want 2", env.Data.Attributes.ItemCount)
	}
	if env.Data.Attributes.GlobalItemCount != 1 {
		t.Errorf("globalItemCount for tenant1 = %d, want 1", env.Data.Attributes.GlobalItemCount)
	}
}
