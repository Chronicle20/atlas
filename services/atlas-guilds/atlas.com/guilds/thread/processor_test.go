package thread

import (
	"atlas-guilds/thread/reply"
	"context"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas-database"
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

	l := setupTestLogger(t)
	database.RegisterTenantCallbacks(l, db)

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate thread: %v", err)
	}
	if err = reply.Migration(db); err != nil {
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

func TestProcessor_GetAll_Empty(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetAll(1)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessor_GetAll_ReturnsThreads(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entities directly in database
	now := time.Now()
	e1 := Entity{
		TenantId:  ten.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Thread 1",
		Message:   "Message 1",
		CreatedAt: now,
	}
	e2 := Entity{
		TenantId:  ten.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Thread 2",
		Message:   "Message 2",
		CreatedAt: now,
	}
	db.Create(&e1)
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetAll(1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestProcessor_GetAll_FiltersByGuild(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	now := time.Now()
	// Thread for guild 1
	e1 := Entity{
		TenantId:  ten.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Guild 1 Thread",
		Message:   "Message",
		CreatedAt: now,
	}
	// Thread for guild 2
	e2 := Entity{
		TenantId:  ten.Id(),
		GuildId:   2,
		PosterId:  100,
		Title:     "Guild 2 Thread",
		Message:   "Message",
		CreatedAt: now,
	}
	db.Create(&e1)
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetAll(1)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Guild 1 Thread", result[0].title)
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	_, err := p.GetById(1, 9999)

	assert.Error(t, err)
}

func TestProcessor_GetById_ReturnsThread(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	now := time.Now()
	e := Entity{
		TenantId:   ten.Id(),
		GuildId:    1,
		PosterId:   100,
		Title:      "Test Thread",
		Message:    "Test Message",
		EmoticonId: 5,
		Notice:     false,
		CreatedAt:  now,
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetById(1, e.Id)

	require.NoError(t, err)
	assert.Equal(t, e.Id, result.id)
	assert.Equal(t, "Test Thread", result.title)
	assert.Equal(t, "Test Message", result.message)
	assert.Equal(t, uint32(100), result.posterId)
}

func TestProcessor_GetById_IncludesReplies(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	now := time.Now()
	e := Entity{
		TenantId:  ten.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Thread With Replies",
		Message:   "Message",
		CreatedAt: now,
	}
	db.Create(&e)

	// Add replies
	r1 := reply.Entity{
		TenantId:  ten.Id(),
		ThreadId:  e.Id,
		PosterId:  200,
		Message:   "Reply 1",
		CreatedAt: now,
	}
	r2 := reply.Entity{
		TenantId:  ten.Id(),
		ThreadId:  e.Id,
		PosterId:  300,
		Message:   "Reply 2",
		CreatedAt: now,
	}
	db.Create(&r1)
	db.Create(&r2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetById(1, e.Id)

	require.NoError(t, err)
	assert.Len(t, result.replies, 2)
}

func TestProcessor_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	now := time.Now()
	// Thread for tenant 1
	e1 := Entity{
		TenantId:  ten1.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Tenant 1 Thread",
		Message:   "Message",
		CreatedAt: now,
	}
	// Thread for tenant 2 (should not be visible)
	e2 := Entity{
		TenantId:  ten2.Id(),
		GuildId:   1,
		PosterId:  100,
		Title:     "Tenant 2 Thread",
		Message:   "Message",
		CreatedAt: now,
	}
	db.Create(&e1)
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetAll(1)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Tenant 1 Thread", result[0].title)
}

func TestProcessor_WithTransaction(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	// Start a transaction
	tx := db.Begin()
	defer tx.Rollback()

	txProcessor := p.WithTransaction(tx)

	assert.NotNil(t, txProcessor)
	assert.NotEqual(t, p, txProcessor)
}
