package searchindex

import (
	"context"
	"errors"
	"fmt"
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
	// Bypass the tenant create-callback's auto-injection: this fixture deliberately
	// writes the supplied tenantId (which may be uuid.Nil to seed a "global" row).
	require.NoError(t, db.WithContext(database.WithoutTenantFilter(ctx)).Create(&row).Error)
}

func widgetSpec() QuerySpec[testEntity] {
	return QuerySpec[testEntity]{
		EntityIdColumn: "widget_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, widget_id ASC",
	}
}

func TestSearch_SubstringMatch(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 1, "Henesys", "red", false)
	seed(t, db, ctx, tn.Id(), 2, "Ellinia", "green", false)

	res, err := Search(db, ctx, tn.Id(), "nesys", 0, 50, widgetSpec())
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

	res, err := Search(db, ctx, tn.Id(), "101", 0, 50, widgetSpec())
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
	res, err := Search(db, ctx, tn.Id(), "box", 0, 50, widgetSpec())
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestSearch_SinglePartition_TenantOwnsDataset(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seed(t, db, ctx, tn.Id(), 7, "TenantWidget", "", false)
	seed(t, db, ctx, uuid.Nil, 7, "GlobalWidget", "", false)
	seed(t, db, ctx, uuid.Nil, 8, "ExtraWidget", "", false)

	// Caller resolves to the tenant id since the tenant has rows; only
	// tenant rows should be visible.
	res, err := Search(db, ctx, tn.Id(), "widget", 0, 50, widgetSpec())
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "TenantWidget", res[0].Name)
}

func TestSearch_SinglePartition_ZeroRowTenantFallsBack(t *testing.T) {
	db := setupTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	// Tenant has no rows; only global rows present.
	seed(t, db, ctx, uuid.Nil, 7, "GlobalWidget", "", false)
	seed(t, db, ctx, uuid.Nil, 8, "ExtraWidget", "", false)

	// Caller resolves to uuid.Nil; only global rows should be visible.
	res, err := Search(db, ctx, uuid.Nil, "widget", 0, 50, widgetSpec())
	require.NoError(t, err)
	require.Len(t, res, 2)
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

	res, err := Search(db, ctx, tn.Id(), "shop", 0, 50, spec)
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

	res, err := SearchWithFilter(db, ctx, tn.Id(), 0, 50, spec)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, uint32(1), res[0].WidgetId)
	assert.Equal(t, uint32(3), res[1].WidgetId)
}

func TestSearch_Pagination_MiddlePage(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)
	for i := 0; i < 100; i++ {
		seed(t, db, ctx, tn.Id(), uint32(1+i), fmt.Sprintf("Apple %03d", i+1), "", false)
	}
	spec := QuerySpec[testEntity]{
		EntityIdColumn: "widget_id",
		NameColumns:    []string{"name"},
		Order:          "widget_id ASC",
	}
	rows, err := Search(db, ctx, tn.Id(), "apple", 40, 20, spec)
	require.NoError(t, err)
	require.Len(t, rows, 20)
	assert.Equal(t, uint32(41), rows[0].WidgetId)
	assert.Equal(t, uint32(60), rows[19].WidgetId)
}

func TestSearch_Pagination_PastEnd(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)
	for i := 0; i < 10; i++ {
		seed(t, db, ctx, tn.Id(), uint32(1+i), "Apple", "", false)
	}
	spec := QuerySpec[testEntity]{EntityIdColumn: "widget_id", NameColumns: []string{"name"}, Order: "widget_id ASC"}
	rows, err := Search(db, ctx, tn.Id(), "apple", 9999, 20, spec)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestSearch_ExactIdHoistOnlyPage1(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)
	// Seed ids 1400..1599 so id=1500 exists for the exact-id hoist.
	for i := 0; i < 200; i++ {
		seed(t, db, ctx, tn.Id(), uint32(1400+i), "Bow", "", false)
	}
	spec := QuerySpec[testEntity]{EntityIdColumn: "widget_id", NameColumns: []string{"name"}, Order: "widget_id ASC"}

	page1, err := Search(db, ctx, tn.Id(), "1500", 0, 50, spec)
	require.NoError(t, err)
	require.NotEmpty(t, page1)
	assert.Equal(t, uint32(1500), page1[0].WidgetId, "exact-id row should be hoisted on page 1")

	page2, err := Search(db, ctx, tn.Id(), "1500", 50, 50, spec)
	require.NoError(t, err)
	// In this seed, no row's name contains "1500", so substring matches zero.
	// Exact-id hoist on page 1 returns row 1500 only. Page 2 returns empty.
	assert.Empty(t, page2)
}

func TestSearchWithFilter_SinglePartition_TenantHasRows(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	tn := tenant.MustFromContext(ctx)
	seed(t, db, ctx, tn.Id(), 1, "TenantA", "", false)
	seed(t, db, ctx, uuid.Nil, 2, "Global", "", false)
	spec := QuerySpec[testEntity]{EntityIdColumn: "widget_id", Order: "widget_id ASC"}
	rows, err := SearchWithFilter(db, ctx, tn.Id(), 0, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "TenantA", rows[0].Name)
}

func TestSearchWithFilter_SinglePartition_ZeroRowTenantFallsBack(t *testing.T) {
	db := setupTestDB(t)
	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	seed(t, db, ctx, uuid.Nil, 1, "Global", "", false)
	spec := QuerySpec[testEntity]{EntityIdColumn: "widget_id", Order: "widget_id ASC"}
	rows, err := SearchWithFilter(db, ctx, uuid.Nil, 0, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Global", rows[0].Name)
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
