package npc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atlas-data/searchindex"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSearchTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{SlowThreshold: time.Second, LogLevel: logger.Silent, Colorful: false},
		),
	})
	require.NoError(t, err)
	db.Migrator().DropTable(&testDocumentEntity{}, &testSearchIndexEntity{})
	require.NoError(t, db.AutoMigrate(&testDocumentEntity{}, &testSearchIndexEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func newSearchTenant(t *testing.T) tenant.Model {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

func seedIdx(t *testing.T, db *gorm.DB, ctx context.Context, tenantId uuid.UUID, id uint32, name string, storebank bool) {
	t.Helper()
	row := testSearchIndexEntity{
		TenantId: tenantId, NpcId: id, Name: name, Storebank: storebank, UpdatedAt: time.Now(),
	}
	require.NoError(t, db.WithContext(ctx).Create(&row).Error)
}

func searchSpec(storebankOnly bool) searchindex.QuerySpec[SearchIndexEntity] {
	spec := searchindex.QuerySpec[SearchIndexEntity]{
		EntityIdColumn: "npc_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, npc_id ASC",
		IdOf:           func(e SearchIndexEntity) uint64 { return uint64(e.NpcId) },
	}
	if storebankOnly {
		spec.ExtraPredicate = "storebank = ?"
		spec.ExtraArgs = []interface{}{true}
	}
	return spec
}

func TestNpcSearch_ExactIdFirst(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	tn := tenant.MustFromContext(ctx)
	seedIdx(t, db, ctx, tn.Id(), 9010000, "Alpha", false)
	seedIdx(t, db, ctx, tn.Id(), 9010001, "Alpha Other", false)

	res, err := searchindex.Search(db, ctx, "9010001", 50, searchSpec(false))
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, uint32(9010001), res[0].NpcId)
}

func TestNpcSearch_Substring(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1, "Storage Keeper", true)
	seedIdx(t, db, ctx, tn.Id(), 2, "Gate Guard", false)

	res, err := searchindex.Search(db, ctx, "keeper", 50, searchSpec(false))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint32(1), res[0].NpcId)
}

func TestNpcSearch_LimitEnforced(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	for i := 0; i < 60; i++ {
		seedIdx(t, db, ctx, tn.Id(), uint32(1000+i), "Box", false)
	}
	res, err := searchindex.Search(db, ctx, "box", 50, searchSpec(false))
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestNpcSearch_TenantFallback(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 5, "TenantNpc", false)
	seedIdx(t, db, ctx, uuid.Nil, 5, "GlobalOverridden", false)
	seedIdx(t, db, ctx, uuid.Nil, 6, "GlobalExtra", false)

	res, err := searchindex.Search(db, ctx, "npc", 50, searchSpec(false))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "TenantNpc", res[0].Name)

	res, err = searchindex.Search(db, ctx, "globalextra", 50, searchSpec(false))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint32(6), res[0].NpcId)
}

func TestNpcSearch_StorebankFilterAlone(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1, "Not Storebank", false)
	seedIdx(t, db, ctx, tn.Id(), 2, "Is Storebank", true)
	seedIdx(t, db, ctx, tn.Id(), 3, "Also Storebank", true)

	res, err := searchindex.SearchWithFilter(db, ctx, 50, searchSpec(true))
	require.NoError(t, err)
	require.Len(t, res, 2)
	for _, r := range res {
		assert.True(t, r.Storebank)
	}
}

func TestNpcSearch_StorebankFilterComposedWithSearch(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1, "Henesys Keeper", true)
	seedIdx(t, db, ctx, tn.Id(), 2, "Henesys Guard", false)
	seedIdx(t, db, ctx, tn.Id(), 3, "Ellinia Keeper", true)

	res, err := searchindex.Search(db, ctx, "henesys", 50, searchSpec(true))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint32(1), res[0].NpcId)
}

func TestHandleGetNpcsRequest_EmptySearchRejected(t *testing.T) {
	db := setupSearchTestDB(t)
	tn := newSearchTenant(t)
	req := httptest.NewRequest(http.MethodGet, "/data/npcs?search=", nil)
	req.Header.Set("TENANT_ID", tn.Id().String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	router := setupTestRouter(db)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStorage_Add_RollbackOnIndexFailure(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	tn := newSearchTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	err := db.Callback().Create().After("gorm:create").Register("test:fail_npc_index", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "npc_search_index" {
			tx.AddError(errors.New("forced index failure"))
		}
	})
	require.NoError(t, err)
	defer db.Callback().Create().Remove("test:fail_npc_index")

	s := NewStorage(l, db)
	_, addErr := s.Add(ctx)(RestModel{Id: 1, Name: "Guard", Storebank: false})()
	require.Error(t, addErr)

	var docCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "NPC").Count(&docCount).Error)
	assert.Equal(t, int64(0), docCount, "documents insert must roll back on index failure")

	var idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), idxCount)
}

func TestStorage_Clear_CascadesToSearchIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	tn := newSearchTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(l, db)
	_, err := s.Add(ctx)(RestModel{Id: 1, Name: "A", Storebank: true})()
	require.NoError(t, err)

	require.NoError(t, s.Clear(ctx))

	var docCount, idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "NPC").Count(&docCount).Error)
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), docCount)
	assert.Equal(t, int64(0), idxCount)
}

// Compile-time check that json encoding is stable.
var _ = json.Marshal
