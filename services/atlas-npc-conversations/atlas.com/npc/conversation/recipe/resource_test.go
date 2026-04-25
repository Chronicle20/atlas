package recipe

import (
	"atlas-npc-conversations/test"
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
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type srvInfo struct{}

func (s srvInfo) GetBaseURL() string { return "" }
func (s srvInfo) GetPrefix() string  { return "/api/" }

type listEnvelope struct {
	Data []struct {
		Id         string                 `json:"id"`
		Type       string                 `json:"type"`
		Attributes map[string]interface{} `json:"attributes"`
	} `json:"data"`
}

func newResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := logtest.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := MigrateTable(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func reqWithTenant(method, path string, tenantId uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func setupRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter()
	l, _ := logtest.NewNullLogger()
	InitResource(srvInfo{})(db)(router, l)
	return router
}


func TestGetByItem_Empty(t *testing.T) {
	db := newResourceTestDB(t)
	router := setupRouter(t, db)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqWithTenant(http.MethodGet, "/items/1082007/recipes", uuid.New()))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var env listEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(env.Data))
	}
}

func TestGetByItem_Populated_OrderedAndTenantScoped(t *testing.T) {
	db := newResourceTestDB(t)
	router := setupRouter(t, db)
	tenantId := uuid.New()
	otherTenant := uuid.New()
	teA, _ := tenant.Create(tenantId, "GMS", 83, 1)
	teB, _ := tenant.Create(otherTenant, "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), teA)
	ctxB := tenant.WithContext(context.Background(), teB)

	for _, m := range []Model{
		newRecipe(t, tenantId, uuid.New(), 2040020, "craftB", 1082007),
		newRecipe(t, tenantId, uuid.New(), 1010000, "craftA", 1082007),
	} {
		if _, err := createRecipe(db.WithContext(ctxA))(tenantId)(m); err != nil {
			t.Fatalf("seed A: %v", err)
		}
	}
	if _, err := createRecipe(db.WithContext(ctxB))(otherTenant)(newRecipe(t, otherTenant, uuid.New(), 9999, "craftZ", 1082007)); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqWithTenant(http.MethodGet, "/items/1082007/recipes", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	var env listEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(env.Data))
	}
	if int(env.Data[0].Attributes["npcId"].(float64)) != 1010000 {
		t.Errorf("ordering wrong: first npcId=%v", env.Data[0].Attributes["npcId"])
	}
	if env.Data[0].Type != "recipes" {
		t.Errorf("type = %q, want recipes", env.Data[0].Type)
	}
}

func TestGetByItem_BadItemIdReturns400(t *testing.T) {
	db := newResourceTestDB(t)
	router := setupRouter(t, db)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqWithTenant(http.MethodGet, "/items/notANumber/recipes", uuid.New()))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetByNpc_Populated(t *testing.T) {
	db := newResourceTestDB(t)
	router := setupRouter(t, db)
	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	for _, m := range []Model{
		newRecipe(t, tenantId, uuid.New(), 2040020, "craftB", 1082007),
		newRecipe(t, tenantId, uuid.New(), 2040020, "craftA", 1082008),
		newRecipe(t, tenantId, uuid.New(), 9999, "noise", 0),
	} {
		if _, err := createRecipe(db.WithContext(ctx))(tenantId)(m); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqWithTenant(http.MethodGet, "/npcs/2040020/recipes", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	var env listEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(env.Data))
	}
	if env.Data[0].Attributes["stateId"] != "craftA" {
		t.Errorf("ordering: first stateId=%v", env.Data[0].Attributes["stateId"])
	}
}

func TestGetByNpc_BadNpcIdReturns400(t *testing.T) {
	_ = test.SetupTestDB
	db := newResourceTestDB(t)
	router := setupRouter(t, db)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, reqWithTenant(http.MethodGet, "/npcs/notANumber/recipes", uuid.New()))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
