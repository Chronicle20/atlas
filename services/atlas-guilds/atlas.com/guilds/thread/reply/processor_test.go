package reply

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

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate reply: %v", err)
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

func TestProcessor_Add_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.Add(1, 100, "Test reply message")

	require.NoError(t, err)
	assert.NotZero(t, result.id)
	assert.Equal(t, uint32(100), result.posterId)
	assert.Equal(t, "Test reply message", result.message)
	assert.NotZero(t, result.createdAt)
}

func TestProcessor_Add_MultipleReplies(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	r1, err := p.Add(1, 100, "First reply")
	require.NoError(t, err)

	r2, err := p.Add(1, 200, "Second reply")
	require.NoError(t, err)

	assert.NotEqual(t, r1.id, r2.id)

	// Verify both exist in database
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND thread_id = ?", ten.Id(), 1).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestProcessor_Delete_Success(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	// First add a reply
	reply, err := p.Add(1, 100, "Reply to delete")
	require.NoError(t, err)

	// Now delete it
	err = p.Delete(1, reply.id)

	require.NoError(t, err)

	// Verify it's gone
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ? AND thread_id = ? AND id = ?", ten.Id(), 1, reply.id).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProcessor_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx1 := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Add reply for tenant 1
	p1 := NewProcessor(l, ctx1, db)
	_, err := p1.Add(1, 100, "Tenant 1 reply")
	require.NoError(t, err)

	// Verify only tenant 1 reply exists
	var count int64
	db.Model(&Entity{}).Where("tenant_id = ?", ten1.Id()).Count(&count)
	assert.Equal(t, int64(1), count)

	db.Model(&Entity{}).Where("tenant_id = ?", ten2.Id()).Count(&count)
	assert.Equal(t, int64(0), count)
}
