package npc

import (
	"atlas-npc-conversations/conversation/recipe"
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

type reindexSrvInfo struct{}

func (s reindexSrvInfo) GetBaseURL() string { return "" }
func (s reindexSrvInfo) GetPrefix() string  { return "/api/" }

type reindexEnvelope struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			DeletedCount         int64 `json:"deletedCount"`
			InsertedCount        int   `json:"insertedCount"`
			SkippedCount         int   `json:"skippedCount"`
			ConversationsScanned int   `json:"conversationsScanned"`
		} `json:"attributes"`
	} `json:"data"`
}

func setupReindexDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := logtest.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := MigrateTable(db); err != nil {
		t.Fatalf("migrate npc: %v", err)
	}
	if err := recipe.MigrateTable(db); err != nil {
		t.Fatalf("migrate recipe: %v", err)
	}
	return db
}

func TestReindexHandler_HappyPath(t *testing.T) {
	db := setupReindexDB(t)

	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	l, _ := logtest.NewNullLogger()
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	if _, err := p.Create(craftConversationModel(t, 1000, craftStateForNpc(t, "c0", "10"))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.WithContext(ctx).Exec("DELETE FROM recipes").Error; err != nil {
		t.Fatalf("wipe: %v", err)
	}

	router := mux.NewRouter()
	InitResource(reindexSrvInfo{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/npcs/conversations/reindex-recipes", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	var env reindexEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Type != "recipeReindexResults" {
		t.Errorf("type = %q, want recipeReindexResults", env.Data.Type)
	}
	if env.Data.Id != tenantId.String() {
		t.Errorf("id = %q, want %q", env.Data.Id, tenantId.String())
	}
	if env.Data.Attributes.InsertedCount != 1 {
		t.Errorf("insertedCount = %d, want 1", env.Data.Attributes.InsertedCount)
	}
	if env.Data.Attributes.ConversationsScanned != 1 {
		t.Errorf("conversationsScanned = %d, want 1", env.Data.Attributes.ConversationsScanned)
	}
}
