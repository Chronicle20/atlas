package database

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// tenantEntity is a test entity with a TenantId field.
type tenantEntity struct {
	ID       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"not null"`
	Name     string    `gorm:"not null"`
}

func (tenantEntity) TableName() string {
	return "tenant_entities"
}

// globalEntity is a test entity without a TenantId field.
type globalEntity struct {
	ID   uint32 `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"not null"`
}

func (globalEntity) TableName() string {
	return "global_entities"
}

func setupTestDB(t *testing.T) (*gorm.DB, logrus.FieldLogger) {
	t.Helper()
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	registerTenantCallbacks(l, db)

	require.NoError(t, db.AutoMigrate(&tenantEntity{}, &globalEntity{}))
	return db, l
}

func tenantContext(id uuid.UUID) context.Context {
	t, _ := tenant.Create(id, "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func TestQueryWithTenantContext_FiltersByTenant(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	// Seed data for two tenants
	db.Create(&tenantEntity{TenantId: tid1, Name: "tenant1-item"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "tenant2-item"})

	// Query with tenant1 context
	var results []tenantEntity
	err := db.WithContext(tenantContext(tid1)).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tenant1-item", results[0].Name)

	// Query with tenant2 context
	results = nil
	err = db.WithContext(tenantContext(tid2)).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tenant2-item", results[0].Name)
}

func TestQueryWithoutTenantContext_ReturnsAll(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	db.Create(&tenantEntity{TenantId: tid1, Name: "item1"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "item2"})

	// Query without tenant context — no filter applied, returns all
	var results []tenantEntity
	err := db.Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestQueryWithSkipTenantFilter_ReturnsAll(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	db.Create(&tenantEntity{TenantId: tid1, Name: "item1"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "item2"})

	// Query with tenant context but skip filter
	ctx := WithoutTenantFilter(tenantContext(tid1))
	var results []tenantEntity
	err := db.WithContext(ctx).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestQueryGlobalEntity_NoFilterApplied(t *testing.T) {
	db, _ := setupTestDB(t)

	db.Create(&globalEntity{Name: "global1"})
	db.Create(&globalEntity{Name: "global2"})

	// Query global entity with tenant context — should NOT filter
	var results []globalEntity
	err := db.WithContext(tenantContext(uuid.New())).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestUpdateWithTenantContext_ScopedToTenant(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	db.Create(&tenantEntity{TenantId: tid1, Name: "before"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "before"})

	// Update only tenant1's records
	err := db.WithContext(tenantContext(tid1)).Model(&tenantEntity{}).Where("name = ?", "before").Update("name", "after").Error
	require.NoError(t, err)

	// Verify tenant1 updated
	var t1 []tenantEntity
	db.Where("tenant_id = ?", tid1).Find(&t1)
	assert.Equal(t, "after", t1[0].Name)

	// Verify tenant2 unchanged
	var t2 []tenantEntity
	db.Where("tenant_id = ?", tid2).Find(&t2)
	assert.Equal(t, "before", t2[0].Name)
}

func TestDeleteWithTenantContext_ScopedToTenant(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	db.Create(&tenantEntity{TenantId: tid1, Name: "delete-me"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "keep-me"})

	// Delete with tenant1 context
	err := db.WithContext(tenantContext(tid1)).Where("name = ?", "delete-me").Delete(&tenantEntity{}).Error
	require.NoError(t, err)

	// Verify tenant1 record deleted
	var all []tenantEntity
	db.Find(&all)
	assert.Len(t, all, 1)
	assert.Equal(t, "keep-me", all[0].Name)
}

func TestFirstWithTenantContext_FiltersByTenant(t *testing.T) {
	db, _ := setupTestDB(t)
	tid1 := uuid.New()
	tid2 := uuid.New()

	db.Create(&tenantEntity{TenantId: tid1, Name: "t1"})
	db.Create(&tenantEntity{TenantId: tid2, Name: "t2"})

	// First with tenant2 context
	var result tenantEntity
	err := db.WithContext(tenantContext(tid2)).First(&result).Error
	require.NoError(t, err)
	assert.Equal(t, "t2", result.Name)
}

func TestDoubleWhereIsHarmless(t *testing.T) {
	db, _ := setupTestDB(t)
	tid := uuid.New()

	db.Create(&tenantEntity{TenantId: tid, Name: "item"})

	// Manual WHERE + automatic callback = double filter, should still work
	var results []tenantEntity
	err := db.WithContext(tenantContext(tid)).Where("tenant_id = ?", tid).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCreateDoesNotInjectWhere(t *testing.T) {
	db, _ := setupTestDB(t)
	tid := uuid.New()

	// Create with tenant context should work normally
	e := tenantEntity{TenantId: tid, Name: "created"}
	err := db.WithContext(tenantContext(tid)).Create(&e).Error
	require.NoError(t, err)

	// Verify it was created
	var result tenantEntity
	db.Where("tenant_id = ?", tid).First(&result)
	assert.Equal(t, "created", result.Name)
}
