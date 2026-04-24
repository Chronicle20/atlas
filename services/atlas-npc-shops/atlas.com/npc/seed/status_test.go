package seed

import (
	"atlas-npc/commodities"
	"atlas-npc/shops"
	"encoding/json"
	"fmt"
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
			ShopCount      int64   `json:"shopCount"`
			CommodityCount int64   `json:"commodityCount"`
			UpdatedAt      *string `json:"updatedAt"`
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

	if err := shops.Migration(db); err != nil {
		t.Fatalf("Failed to migrate shops: %v", err)
	}
	if err := commodities.Migration(db); err != nil {
		t.Fatalf("Failed to migrate commodities: %v", err)
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
	req := httptest.NewRequest(http.MethodGet, "/shops/seed/status", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedShop(t *testing.T, db *gorm.DB, tenantId uuid.UUID, npcId uint32) {
	t.Helper()
	m, err := shops.NewBuilder(npcId).SetRecharger(false).Build()
	if err != nil {
		t.Fatalf("build shop: %v", err)
	}
	if err := shops.BulkCreateShops(db, tenantId, []shops.Model{m}); err != nil {
		t.Fatalf("bulk create shops: %v", err)
	}
}

func seedCommodity(t *testing.T, db *gorm.DB, tenantId uuid.UUID, npcId uint32, templateId uint32) {
	t.Helper()
	m, err := commodities.NewBuilder().
		SetId(uuid.New()).
		SetNpcId(npcId).
		SetTemplateId(templateId).
		SetMesoPrice(100).
		Build()
	if err != nil {
		t.Fatalf("build commodity: %v", err)
	}
	if err := commodities.BulkCreateCommodities(db, tenantId, []commodities.Model{m}); err != nil {
		t.Fatalf("bulk create commodities: %v", err)
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
	if env.Data.Type != "npcShopsSeedStatus" {
		t.Errorf("type = %q, want npcShopsSeedStatus", env.Data.Type)
	}
	if env.Data.Id != tenantId.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenantId.String())
	}
	if env.Data.Attributes.ShopCount != 0 {
		t.Errorf("shopCount = %d, want 0", env.Data.Attributes.ShopCount)
	}
	if env.Data.Attributes.CommodityCount != 0 {
		t.Errorf("commodityCount = %d, want 0", env.Data.Attributes.CommodityCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestStatusHandler_Populated(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenantId := uuid.New()
	seedShop(t, db, tenantId, 9000001)
	seedShop(t, db, tenantId, 9000002)
	seedCommodity(t, db, tenantId, 9000001, 2000000)
	seedCommodity(t, db, tenantId, 9000001, 2000001)
	seedCommodity(t, db, tenantId, 9000002, 2000002)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusReq(tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.ShopCount != 2 {
		t.Errorf("shopCount = %d, want 2", env.Data.Attributes.ShopCount)
	}
	if env.Data.Attributes.CommodityCount != 3 {
		t.Errorf("commodityCount = %d, want 3", env.Data.Attributes.CommodityCount)
	}
	if env.Data.Attributes.UpdatedAt == nil {
		t.Errorf("updatedAt should not be null (entities embed gorm.Model)")
	}
}

func TestStatusHandler_TenantIsolation(t *testing.T) {
	db := newTestDB(t)
	router := setupStatusRouter(t, db)

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Tenant 1: 1 shop, 1 commodity
	seedShop(t, db, tenant1, 9100001)
	seedCommodity(t, db, tenant1, 9100001, 2100000)

	// Tenant 2: 2 shops, 2 commodities
	seedShop(t, db, tenant2, 9200001)
	seedShop(t, db, tenant2, 9200002)
	seedCommodity(t, db, tenant2, 9200001, 2200000)
	seedCommodity(t, db, tenant2, 9200002, 2200001)

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
	if env.Data.Attributes.ShopCount != 1 {
		t.Errorf("shopCount for tenant1 = %d, want 1", env.Data.Attributes.ShopCount)
	}
	if env.Data.Attributes.CommodityCount != 1 {
		t.Errorf("commodityCount for tenant1 = %d, want 1", env.Data.Attributes.CommodityCount)
	}
}
