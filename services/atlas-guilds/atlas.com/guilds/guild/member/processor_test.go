package member

import (
	"atlas-guilds/guild/character"
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

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate member: %v", err)
	}
	if err = character.Migration(db); err != nil {
		t.Fatalf("Failed to migrate character: %v", err)
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

func TestProcessor_AddMember_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.AddMember(1, 100, "TestPlayer", 111, 50, 5)

	require.NoError(t, err)
	assert.Equal(t, uint32(1), result.guildId)
	assert.Equal(t, uint32(100), result.characterId)
	assert.Equal(t, "TestPlayer", result.name)
	assert.Equal(t, uint16(111), result.jobId)
	assert.Equal(t, byte(50), result.level)
	assert.Equal(t, byte(5), result.title)

	// Verify character-guild association was created
	var charEntity character.Entity
	err = db.Where("tenant_id = ? AND character_id = ?", ten.Id(), 100).First(&charEntity).Error
	require.NoError(t, err)
	assert.Equal(t, uint32(1), charEntity.GuildId)
}

func TestProcessor_RemoveMember_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// First add a member
	p := NewProcessor(l, ctx, db)
	_, err := p.AddMember(1, 100, "TestPlayer", 111, 50, 5)
	require.NoError(t, err)

	// Now remove the member
	err = p.RemoveMember(1, 100)

	require.NoError(t, err)

	// Verify member was removed
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND guild_id = ? AND character_id = ?", ten.Id(), 1, 100).Count(&count)
	assert.Equal(t, int64(0), count)

	// Verify character-guild association was updated to 0
	var charEntity character.Entity
	err = db.Where("tenant_id = ? AND character_id = ?", ten.Id(), 100).First(&charEntity).Error
	require.NoError(t, err)
	assert.Equal(t, uint32(0), charEntity.GuildId)
}

func TestProcessor_UpdateStatus_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// First add a member
	p := NewProcessor(l, ctx, db)
	_, err := p.AddMember(1, 100, "TestPlayer", 111, 50, 5)
	require.NoError(t, err)

	// Update status to online
	err = p.UpdateStatus(100, true)

	require.NoError(t, err)

	// Verify status was updated
	var entity Entity
	err = db.Where("tenant_id = ? AND character_id = ?", ten.Id(), 100).First(&entity).Error
	require.NoError(t, err)
	assert.True(t, entity.Online)

	// Update status to offline
	err = p.UpdateStatus(100, false)
	require.NoError(t, err)

	err = db.Where("tenant_id = ? AND character_id = ?", ten.Id(), 100).First(&entity).Error
	require.NoError(t, err)
	assert.False(t, entity.Online)
}

func TestProcessor_UpdateTitle_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// First add a member with title 5
	p := NewProcessor(l, ctx, db)
	_, err := p.AddMember(1, 100, "TestPlayer", 111, 50, 5)
	require.NoError(t, err)

	// Update title to 2
	err = p.UpdateTitle(100, 2)

	require.NoError(t, err)

	// Verify title was updated
	var entity Entity
	err = db.Where("tenant_id = ? AND character_id = ?", ten.Id(), 100).First(&entity).Error
	require.NoError(t, err)
	assert.Equal(t, byte(2), entity.Title)
}

func TestProcessor_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx1 := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Add member for tenant 1
	p1 := NewProcessor(l, ctx1, db)
	_, err := p1.AddMember(1, 100, "Tenant1Player", 111, 50, 5)
	require.NoError(t, err)

	// Verify only tenant 1 member exists
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ?", ten1.Id()).Count(&count)
	assert.Equal(t, int64(1), count)

	db.Model(&Entity{}).Where("tenant_id = ?", ten2.Id()).Count(&count)
	assert.Equal(t, int64(0), count)
}
