package item

import (
	"context"
	"encoding/json"
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

type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (testDocumentEntity) TableName() string { return "documents" }

type testSearchIndexEntity struct {
	TenantId  uuid.UUID `gorm:"type:text;primaryKey"`
	ItemId    uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (testSearchIndexEntity) TableName() string { return "item_string_search_index" }

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
	row := testSearchIndexEntity{TenantId: tenantId, ItemId: id, Name: name, UpdatedAt: time.Now()}
	require.NoError(t, db.WithContext(ctx).Create(&row).Error)
}

func searchSpec() searchindex.QuerySpec[StringSearchIndexEntity] {
	return searchindex.QuerySpec[StringSearchIndexEntity]{
		EntityIdColumn: "item_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, item_id ASC",
		IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
	}
}

func TestItemStringSearch_ExactIdFirst(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1002000, "Hat")
	seedIdx(t, db, ctx, tn.Id(), 1002001, "Cap")

	res, err := searchindex.Search(db, ctx, "1002001", 50, searchSpec())
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, uint32(1002001), res[0].ItemId)
}

func TestItemStringSearch_Substring(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 1, "Red Cap")
	seedIdx(t, db, ctx, tn.Id(), 2, "Blue Helm")

	res, err := searchindex.Search(db, ctx, "cap", 50, searchSpec())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Red Cap", res[0].Name)
}

func TestItemStringSearch_LimitEnforced(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	for i := 0; i < 60; i++ {
		seedIdx(t, db, ctx, tn.Id(), uint32(1000+i), "Apple")
	}
	res, err := searchindex.Search(db, ctx, "apple", 50, searchSpec())
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestItemStringSearch_TenantFallback(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdx(t, db, ctx, tn.Id(), 5, "TenantItem")
	seedIdx(t, db, ctx, uuid.Nil, 5, "GlobalOverridden")

	res, err := searchindex.Search(db, ctx, "item", 50, searchSpec())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "TenantItem", res[0].Name)
}

func TestItemStringStorage_Add_RollbackOnIndexFailure(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	err := db.Callback().Create().After("gorm:create").Register("test:fail_item_index", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "item_string_search_index" {
			tx.AddError(errors.New("forced index failure"))
		}
	})
	require.NoError(t, err)
	defer db.Callback().Create().Remove("test:fail_item_index")

	s := NewStringStorage(l, db)
	_, addErr := s.Add(ctx)(StringRestModel{Id: "1", Name: "Hat"})()
	require.Error(t, addErr)

	var docCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "ITEM_STRING").Count(&docCount).Error)
	assert.Equal(t, int64(0), docCount)
}

func TestItemStringStorage_Clear_CascadesToSearchIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))

	s := NewStringStorage(l, db)
	_, err := s.Add(ctx)(StringRestModel{Id: "1", Name: "A"})()
	require.NoError(t, err)

	require.NoError(t, s.Clear(ctx))

	var docCount, idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testDocumentEntity{}).Where("type = ?", "ITEM_STRING").Count(&docCount).Error)
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(0), docCount)
	assert.Equal(t, int64(0), idxCount)
}
