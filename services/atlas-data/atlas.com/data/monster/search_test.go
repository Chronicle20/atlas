package monster

import (
	"context"
	"errors"
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

func seedIdx(t *testing.T, db *gorm.DB, ctx context.Context, tenantId uuid.UUID, id uint32, name string) {
	t.Helper()
	row := testSearchIndexEntity{TenantId: tenantId, MonsterId: id, Name: name, UpdatedAt: time.Now()}
	require.NoError(t, db.WithContext(ctx).Create(&row).Error)
}

func searchSpec() searchindex.QuerySpec[SearchIndexEntity] {
	return searchindex.QuerySpec[SearchIndexEntity]{
		EntityIdColumn: "monster_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, monster_id ASC",
		IdOf:           func(e SearchIndexEntity) uint64 { return uint64(e.MonsterId) },
	}
}

func TestMonsterSearch_ExactIdFirst(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 100100, "Blue Snail")
	seedIdx(t, db, ctx, tn.Id(), 100101, "Red Snail")

	res, err := searchindex.Search(db, ctx, "100101", 50, searchSpec())
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, uint32(100101), res[0].MonsterId)
}

func TestMonsterSearch_Substring(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1, "Papulatus")
	seedIdx(t, db, ctx, tn.Id(), 2, "Snail")

	res, err := searchindex.Search(db, ctx, "papu", 50, searchSpec())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Papulatus", res[0].Name)
}

func TestMonsterSearch_LimitEnforced(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	for i := 0; i < 60; i++ {
		seedIdx(t, db, ctx, tn.Id(), uint32(1000+i), "Slime")
	}
	res, err := searchindex.Search(db, ctx, "slime", 50, searchSpec())
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestMonsterSearch_TenantFallback(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 5, "TenantMob")
	seedIdx(t, db, ctx, uuid.Nil, 5, "GlobalOverridden")
	seedIdx(t, db, ctx, uuid.Nil, 6, "Mob")

	res, err := searchindex.Search(db, ctx, "mob", 50, searchSpec())
	require.NoError(t, err)
	require.Len(t, res, 2)

	byId := map[uint32]SearchIndexEntity{res[0].MonsterId: res[0], res[1].MonsterId: res[1]}
	assert.Equal(t, "TenantMob", byId[5].Name, "tenant row must win over global")
}

func TestMonsterStorage_Add_RollbackOnIndexFailure(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	err := db.Callback().Create().After("gorm:create").Register("test:fail_monster_index", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "monster_search_index" {
			tx.AddError(errors.New("forced index failure"))
		}
	})
	require.NoError(t, err)
	defer db.Callback().Create().Remove("test:fail_monster_index")

	s := NewStorage(l, db)
	_, addErr := s.Add(ctx)(RestModel{Id: 1, Name: "Snail"})()
	require.Error(t, addErr)

	var docCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "MONSTER").Count(&docCount).Error)
	assert.Equal(t, int64(0), docCount)
}

func TestMonsterStorage_Clear_CascadesToSearchIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	s := NewStorage(l, db)
	_, err := s.Add(ctx)(RestModel{Id: 1, Name: "A"})()
	require.NoError(t, err)

	require.NoError(t, s.Clear(ctx))

	var docCount, idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "MONSTER").Count(&docCount).Error)
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), docCount)
	assert.Equal(t, int64(0), idxCount)
}
