package title

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func setupTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	// Use unique database per test to avoid conflicts
	dbName := uuid.New().String()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Use raw SQL for title table to avoid PostgreSQL-specific uuid_generate_v4()
	if err = db.Exec(`CREATE TABLE IF NOT EXISTS titles (
		tenant_id TEXT NOT NULL,
		id TEXT,
		guild_id INTEGER,
		name TEXT,
		"index" INTEGER
	)`).Error; err != nil {
		t.Fatalf("Failed to create title table: %v", err)
	}

	return db
}

func TestNewProcessor(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	assert.NotNil(t, p)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	ctx := context.Background() // No tenant in context
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx, db)
	})
}

func TestProcessor_CreateDefaults_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.CreateDefaults(1)

	require.NoError(t, err)
	assert.Len(t, result, 5) // Default titles: Master, Jr. Master, Member, and 2 more

	// Verify they exist in database
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND guild_id = ?", ten.Id(), 1).Count(&count)
	assert.Equal(t, int64(5), count)
}

func TestProcessor_Replace_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	// First create defaults
	_, err := p.CreateDefaults(1)
	require.NoError(t, err)

	// Now replace with custom titles
	customTitles := []string{"Leader", "Officer", "Member", "Recruit", "Newbie"}
	err = p.Replace(1, customTitles)

	require.NoError(t, err)

	// Verify titles were replaced
	var entities []Entity
	db.Where("tenant_id = ? AND guild_id = ?", ten.Id(), 1).Find(&entities)
	assert.Len(t, entities, 5)

	// Check title names
	names := make(map[string]bool)
	for _, e := range entities {
		names[e.Name] = true
	}
	for _, title := range customTitles {
		assert.True(t, names[title], "Expected title %s not found", title)
	}
}

func TestProcessor_Clear_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	// First create defaults
	_, err := p.CreateDefaults(1)
	require.NoError(t, err)

	// Now clear them
	err = p.Clear(1)

	require.NoError(t, err)

	// Verify all titles are gone
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND guild_id = ?", ten.Id(), 1).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProcessor_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx1 := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create defaults for tenant 1
	p1 := NewProcessor(l, ctx1, db)
	_, err := p1.CreateDefaults(1)
	require.NoError(t, err)

	// Verify only tenant 1 titles exist
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ?", ten1.Id()).Count(&count)
	assert.Equal(t, int64(5), count)

	db.Model(&Entity{}).Where("tenant_id = ?", ten2.Id()).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProcessor_Clear_DoesNotAffectOtherGuilds(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	// Create defaults for two guilds
	_, err := p.CreateDefaults(1)
	require.NoError(t, err)
	_, err = p.CreateDefaults(2)
	require.NoError(t, err)

	// Clear guild 1 only
	err = p.Clear(1)
	require.NoError(t, err)

	// Verify guild 1 is cleared
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND guild_id = ?", ten.Id(), 1).Count(&count)
	assert.Equal(t, int64(0), count)

	// Verify guild 2 is still intact
	db.Model(&Entity{}).Where("tenant_id = ? AND guild_id = ?", ten.Id(), 2).Count(&count)
	assert.Equal(t, int64(5), count)
}
