package fame

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func setupProcessorTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func TestNewProcessor(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	assert.NotNil(t, p)
}

func TestNewProcessor_ExtractsTenant(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)
	impl := p.(*ProcessorImpl)

	assert.Equal(t, ten, impl.t)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	ctx := context.Background() // No tenant in context
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx, db)
	})
}

func TestProcessor_GetByCharacterIdLastMonth_Empty(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessor_GetByCharacterIdLastMonth_ReturnsResults(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	// Create test entity directly in database
	now := time.Now()
	e := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(1000), result[0].CharacterId())
	assert.Equal(t, uint32(2000), result[0].TargetId())
	assert.Equal(t, int8(1), result[0].Amount())
}

func TestProcessor_GetByCharacterIdLastMonth_FiltersByTenant(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()

	// Create entity for tenant 1
	e1 := Entity{
		TenantId:    ten1.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e1)

	// Create entity for tenant 2 (same character ID)
	e2 := Entity{
		TenantId:    ten2.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2001,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ten1.Id(), result[0].TenantId())
}

func TestProcessor_GetByCharacterIdLastMonth_ExcludesOldRecords(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()

	// Create recent entity
	e1 := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e1)

	// Create old entity (older than 1 month)
	e2 := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2001,
		Amount:      1,
		CreatedAt:   now.AddDate(0, -2, 0),
	}
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(2000), result[0].TargetId())
}

func TestProcessor_ByCharacterIdLastMonthProvider(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()
	e := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	provider := p.ByCharacterIdLastMonthProvider(1000)
	result, err := provider()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}
