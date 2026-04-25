package item

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
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
	TenantId    uuid.UUID `gorm:"type:text;primaryKey"`
	ItemId      uint32    `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	Compartment uint8     `gorm:"not null;default:0"`
	Subcategory string    `gorm:"not null;default:''"`
	JobMask     *uint8
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
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
	seedIdxFull(t, db, ctx, tenantId, id, name, 0, "", nil)
}

func seedIdxFull(t *testing.T, db *gorm.DB, ctx context.Context, tenantId uuid.UUID, id uint32, name string, compartment uint8, subcategory string, jobMask *uint8) {
	t.Helper()
	row := testSearchIndexEntity{
		TenantId:    tenantId,
		ItemId:      id,
		Name:        name,
		Compartment: compartment,
		Subcategory: subcategory,
		JobMask:     jobMask,
		UpdatedAt:   time.Now(),
	}
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

func TestItemStringStorage_Add_WritesCompartmentAndSubcategory(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	s := NewStringStorage(l, db)
	_, err := s.Add(ctx)(StringRestModel{Id: "2049000", Name: "Clean Slate Scroll 1%"})()
	require.NoError(t, err)

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 2049000).Error)
	assert.Equal(t, uint8(CompartmentUse), row.Compartment)
	assert.Equal(t, "scroll", row.Subcategory)
	assert.Nil(t, row.JobMask)
}

func TestItemStringStorage_Add_RefreshesCompartmentOnReingest(t *testing.T) {
	db := setupSearchTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	s := NewStringStorage(l, db)
	_, err := s.Add(ctx)(StringRestModel{Id: "2049000", Name: "Clean Slate Scroll 1%"})()
	require.NoError(t, err)
	_, err = s.Add(ctx)(StringRestModel{Id: "2049000", Name: "Clean Slate Scroll 1% (renamed)"})()
	require.NoError(t, err)

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 2049000).Error)
	assert.Equal(t, "Clean Slate Scroll 1% (renamed)", row.Name)
	assert.Equal(t, uint8(CompartmentUse), row.Compartment)
	assert.Equal(t, "scroll", row.Subcategory)
}

func TestParseFilters_Compartment(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=equipment")
	spec, errCode := parseFilters(q)
	require.Equal(t, 0, errCode)
	require.NotNil(t, spec.Compartment)
	assert.Equal(t, CompartmentEquipment, *spec.Compartment)
}

func TestParseFilters_RejectsUnknownCompartment(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=foo")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func TestParseFilters_SubcategoryWithoutCompartment(t *testing.T) {
	q := mustParseQuery(t, "filter[subcategory]=bow")
	spec, errCode := parseFilters(q)
	require.Equal(t, 0, errCode)
	assert.Equal(t, "bow", spec.Subcategory)
}

func TestParseFilters_RejectsUnknownSubcategory(t *testing.T) {
	q := mustParseQuery(t, "filter[subcategory]=zzz-not-real")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func TestParseFilters_RejectsSubcategoryFromOtherCompartment(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=use&filter[subcategory]=bow")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func TestParseFilters_ClassAny(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=equipment&filter[class]=any")
	spec, errCode := parseFilters(q)
	require.Equal(t, 0, errCode)
	assert.True(t, spec.ClassIsAny)
	assert.Equal(t, "any", spec.Class)
	assert.Equal(t, uint8(0), spec.JobMaskBits)
}

func TestParseFilters_ClassIntersection(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=equipment&filter[class]=warrior,bowman")
	spec, errCode := parseFilters(q)
	require.Equal(t, 0, errCode)
	assert.False(t, spec.ClassIsAny)
	assert.Equal(t, uint8(1|4), spec.JobMaskBits)
}

func TestParseFilters_RejectsClassWithoutEquipment(t *testing.T) {
	q := mustParseQuery(t, "filter[class]=warrior")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func TestParseFilters_RejectsClassWithWrongCompartment(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=cash&filter[class]=warrior")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func TestParseFilters_RejectsUnknownClass(t *testing.T) {
	q := mustParseQuery(t, "filter[compartment]=equipment&filter[class]=warrior,fighter")
	_, errCode := parseFilters(q)
	assert.Equal(t, 400, errCode)
}

func mustParseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	q, err := url.ParseQuery(raw)
	require.NoError(t, err)
	return q
}

func TestSearchIndex_DefaultBrowse_ExcludesStaleAndOrders(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1, "Stale Item", 0, "", nil)
	seedIdxFull(t, db, ctx, tn.Id(), 2, "Banana", uint8(CompartmentUse), "potion", nil)
	seedIdxFull(t, db, ctx, tn.Id(), 3, "Apple", uint8(CompartmentUse), "potion", nil)

	spec := searchindex.QuerySpec[StringSearchIndexEntity]{
		EntityIdColumn: "item_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, item_id ASC",
		IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
		ExtraPredicate: "compartment != 0",
	}
	rows, err := searchindex.SearchWithFilter(db, ctx, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Apple", rows[0].Name)
	assert.Equal(t, "Banana", rows[1].Name)
}

func TestSearchIndex_FilterCompartmentOnly(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1, "Hat", uint8(CompartmentEquipment), "hat", nil)
	seedIdxFull(t, db, ctx, tn.Id(), 2, "Potion", uint8(CompartmentUse), "potion", nil)

	spec := searchindex.QuerySpec[StringSearchIndexEntity]{
		EntityIdColumn: "item_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, item_id ASC",
		IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
		ExtraPredicate: "compartment = ?",
		ExtraArgs:      []interface{}{int(CompartmentEquipment)},
	}
	rows, err := searchindex.SearchWithFilter(db, ctx, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Hat", rows[0].Name)
}

func TestSearchIndex_FilterClassIntersection(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	mask := func(v uint8) *uint8 { return &v }
	seedIdxFull(t, db, ctx, tn.Id(), 1, "Bow", uint8(CompartmentEquipment), "bow", mask(4))
	seedIdxFull(t, db, ctx, tn.Id(), 2, "Sword", uint8(CompartmentEquipment), "one-handed-sword", mask(1))
	seedIdxFull(t, db, ctx, tn.Id(), 3, "Hat", uint8(CompartmentEquipment), "hat", mask(0))

	spec := searchindex.QuerySpec[StringSearchIndexEntity]{
		EntityIdColumn: "item_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, item_id ASC",
		IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
		ExtraPredicate: "(compartment = ?) AND (job_mask IS NOT NULL AND (job_mask = 0 OR (job_mask & ?) = ?))",
		ExtraArgs:      []interface{}{int(CompartmentEquipment), uint8(1), uint8(1)},
	}
	rows, err := searchindex.SearchWithFilter(db, ctx, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestSearchIndex_FilterClassAny(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	mask := func(v uint8) *uint8 { return &v }
	seedIdxFull(t, db, ctx, tn.Id(), 1, "Bow", uint8(CompartmentEquipment), "bow", mask(4))
	seedIdxFull(t, db, ctx, tn.Id(), 2, "Hat", uint8(CompartmentEquipment), "hat", mask(0))

	spec := searchindex.QuerySpec[StringSearchIndexEntity]{
		EntityIdColumn: "item_id",
		NameColumns:    []string{"name"},
		Order:          "name ASC, item_id ASC",
		IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
		ExtraPredicate: "(compartment = ?) AND (job_mask IS NOT NULL AND job_mask = 0)",
		ExtraArgs:      []interface{}{int(CompartmentEquipment)},
	}
	rows, err := searchindex.SearchWithFilter(db, ctx, 50, spec)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Hat", rows[0].Name)
}
