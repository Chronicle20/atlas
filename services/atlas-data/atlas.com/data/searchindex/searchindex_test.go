package searchindex

import (
	"context"
	"errors"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testEntity struct {
	TenantId  uuid.UUID `gorm:"type:text;primaryKey"`
	WidgetId  uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Color     string    `gorm:"not null;default:''"`
	Flagged   bool      `gorm:"not null;default:false"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (testEntity) TableName() string { return "widget_search_index" }

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{SlowThreshold: time.Second, LogLevel: logger.Silent, Colorful: false},
		),
	})
	require.NoError(t, err)
	db.Migrator().DropTable(&testEntity{})
	require.NoError(t, db.AutoMigrate(&testEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func newTestTenant(t *testing.T) tenant.Model {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

func seed(t *testing.T, db *gorm.DB, ctx context.Context, tenantId uuid.UUID, id uint32, name, color string, flagged bool) {
	t.Helper()
	row := testEntity{
		TenantId: tenantId, WidgetId: id, Name: name, Color: color, Flagged: flagged, UpdatedAt: time.Now(),
	}
	require.NoError(t, db.WithContext(ctx).Create(&row).Error)
}

func widgetSpec() QuerySpec[testEntity] {
	return QuerySpec[testEntity]{
		EntityIdColumn: "widget_id",
		NameColumns:    []string{"name"},
		IdOf:           func(e testEntity) uint64 { return uint64(e.WidgetId) },
		Order:          "name ASC, widget_id ASC",
	}
}

func TestSearch_SubstringMatch(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "Henesys", "red", false)
	seed(t, db, ctx, tn.Id(), 2, "Ellinia", "green", false)

	res, err := Search(db, ctx, "nesys", 50, widgetSpec())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Henesys", res[0].Name)
}

func TestSearch_ExactIdFirst(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 100, "Alpha 100", "", false)
	seed(t, db, ctx, tn.Id(), 101, "Alpha 101", "", false)
	seed(t, db, ctx, tn.Id(), 102, "Alpha 102", "", false)

	res, err := Search(db, ctx, "101", 50, widgetSpec())
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, uint32(101), res[0].WidgetId)
}

func TestSearch_LimitEnforced(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	for i := 0; i < 60; i++ {
		seed(t, db, ctx, tn.Id(), uint32(1000+i), "Box", "", false)
	}
	res, err := Search(db, ctx, "box", 50, widgetSpec())
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestSearch_TenantFallback_TenantWins(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 7, "TenantWidget", "", false)
	seed(t, db, ctx, uuid.Nil, 7, "GlobalWidget", "", false)
	seed(t, db, ctx, uuid.Nil, 8, "ExtraWidget", "", false)

	res, err := Search(db, ctx, "widget", 50, widgetSpec())
	require.NoError(t, err)
	require.Len(t, res, 2)

	byId := map[uint32]testEntity{res[0].WidgetId: res[0], res[1].WidgetId: res[1]}
	assert.Equal(t, "TenantWidget", byId[7].Name, "tenant row must win over global")
	assert.Equal(t, "ExtraWidget", byId[8].Name)
}

func TestSearch_ExtraPredicate(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "Shop Apple", "", true)
	seed(t, db, ctx, tn.Id(), 2, "Shop Banana", "", false)

	spec := widgetSpec()
	spec.ExtraPredicate = "flagged = ?"
	spec.ExtraArgs = []interface{}{true}

	res, err := Search(db, ctx, "shop", 50, spec)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint32(1), res[0].WidgetId)
}

func TestSearchWithFilter(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "A", "", true)
	seed(t, db, ctx, tn.Id(), 2, "B", "", false)
	seed(t, db, ctx, tn.Id(), 3, "C", "", true)

	spec := widgetSpec()
	spec.ExtraPredicate = "flagged = ?"
	spec.ExtraArgs = []interface{}{true}

	res, err := SearchWithFilter(db, ctx, 50, spec)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, uint32(1), res[0].WidgetId)
	assert.Equal(t, uint32(3), res[1].WidgetId)
}

func TestUpsert_InsertThenUpdate(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	row := testEntity{TenantId: tn.Id(), WidgetId: 1, Name: "Original", Color: "red", UpdatedAt: time.Now()}
	require.NoError(t, Upsert(db.WithContext(ctx), &row, []string{"tenant_id", "widget_id"}, []string{"name", "color", "updated_at"}))

	row2 := testEntity{TenantId: tn.Id(), WidgetId: 1, Name: "Updated", Color: "blue", UpdatedAt: time.Now()}
	require.NoError(t, Upsert(db.WithContext(ctx), &row2, []string{"tenant_id", "widget_id"}, []string{"name", "color", "updated_at"}))

	var out testEntity
	require.NoError(t, db.WithContext(ctx).Where("widget_id = ?", 1).Take(&out).Error)
	assert.Equal(t, "Updated", out.Name)
	assert.Equal(t, "blue", out.Color)
}

func TestUpsert_RollsBackOuterTransaction(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := testEntity{TenantId: tn.Id(), WidgetId: 1, Name: "A", UpdatedAt: time.Now()}
		if err := Upsert(tx, &row, []string{"tenant_id", "widget_id"}, []string{"name", "updated_at"}); err != nil {
			return err
		}
		return errors.New("forced failure after upsert")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.WithContext(ctx).Model(&testEntity{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "outer transaction rollback must undo the upsert")
}

func TestDeleteAllForTenant(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "X", "", false)
	seed(t, db, ctx, tn.Id(), 2, "Y", "", false)
	seed(t, db, ctx, uuid.Nil, 3, "Z", "", false)

	require.NoError(t, DeleteAllForTenant(db.WithContext(ctx), tn.Id(), &testEntity{}))

	var count int64
	require.NoError(t, db.WithContext(database.WithoutTenantFilter(ctx)).Model(&testEntity{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "only tenant rows should be deleted")
}

func TestResolveTenantId_TenantHasRows(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "Active", "", false)
	seed(t, db, ctx, uuid.Nil, 2, "Global", "", false)

	got, err := ResolveTenantId[testEntity](db, ctx, QuerySpec[testEntity]{})
	require.NoError(t, err)
	assert.Equal(t, tn.Id(), got)
}

func TestResolveTenantId_TenantHasZeroRows(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, uuid.Nil, 1, "Global", "", false)

	got, err := ResolveTenantId[testEntity](db, ctx, QuerySpec[testEntity]{})
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestResolveTenantId_EmptyTable(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	got, err := ResolveTenantId[testEntity](db, ctx, QuerySpec[testEntity]{})
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, got)
}

func TestCount_FilterOnly(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)

	for i := 0; i < 30; i++ {
		seed(t, db, ctx, tn.Id(), uint32(1+i), "Apple", "", false)
	}
	got, err := CountWithFilter[testEntity](db, ctx, tn.Id(), QuerySpec[testEntity]{})
	require.NoError(t, err)
	assert.Equal(t, 30, got)
}

func TestCount_SubstringMatch(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)

	seed(t, db, ctx, tn.Id(), 1, "Red Cap", "", false)
	seed(t, db, ctx, tn.Id(), 2, "Blue Helm", "", false)
	seed(t, db, ctx, tn.Id(), 3, "Green Cap", "", false)

	spec := QuerySpec[testEntity]{
		EntityIdColumn: "widget_id",
		NameColumns:    []string{"name"},
	}
	got, err := Count[testEntity](db, ctx, tn.Id(), "cap", spec)
	require.NoError(t, err)
	assert.Equal(t, 2, got)
}

func TestCount_NumericQueryUnionedOnce(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)

	// id 1452000 — name contains "1452000" so substring + exact-id both match
	seed(t, db, ctx, tn.Id(), 1452000, "Bow 1452000", "", false)
	// id 1452001 — substring-only match
	seed(t, db, ctx, tn.Id(), 1452001, "Bow Reference 1452000", "", false)
	// id 1452002 — exact-id only (name does not contain digits)
	seed(t, db, ctx, tn.Id(), 1452002, "Wooden Bow", "", false)

	spec := QuerySpec[testEntity]{
		EntityIdColumn: "widget_id",
		NameColumns:    []string{"name"},
	}
	// q="1452000": substring matches rows 1+2; exact-id matches row 1.
	// Single OR -> row 1 counted once. Total = 2.
	got, err := Count[testEntity](db, ctx, tn.Id(), "1452000", spec)
	require.NoError(t, err)
	assert.Equal(t, 2, got)

	// q="1452002": substring matches no rows; exact-id matches row 3.
	// Total = 1.
	got, err = Count[testEntity](db, ctx, tn.Id(), "1452002", spec)
	require.NoError(t, err)
	assert.Equal(t, 1, got)
}
