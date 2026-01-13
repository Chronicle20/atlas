package guild

import (
	"atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
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
		t.Fatalf("Failed to migrate guild: %v", err)
	}
	if err = member.Migration(db); err != nil {
		t.Fatalf("Failed to migrate member: %v", err)
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

func TestNewProcessor_ExtractsTenant(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)
	impl := p.(*ProcessorImpl)

	assert.Equal(t, ten, impl.t)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	ctx := context.Background() // No tenant in context
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx, db)
	})
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	_, err := p.GetById(9999)

	assert.Error(t, err)
}

func TestProcessor_GetById_ReturnsGuild(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entity directly in database
	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "TestGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetById(e.Id)

	require.NoError(t, err)
	assert.Equal(t, e.Id, result.Id())
	assert.Equal(t, "TestGuild", result.name)
	assert.Equal(t, uint32(100), result.LeaderId())
}

func TestProcessor_GetByName_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByName(0, "NonExistentGuild")

	// GetByName uses FirstProvider which returns "empty slice" error when not found
	assert.Error(t, err)
	assert.Equal(t, uint32(0), result.Id())
}

func TestProcessor_GetByName_ReturnsGuild(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entity directly in database
	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "UniqueGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByName(0, "UniqueGuild")

	require.NoError(t, err)
	assert.Equal(t, e.Id, result.Id())
	assert.Equal(t, "UniqueGuild", result.name)
}

func TestProcessor_GetByName_CaseInsensitive(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entity directly in database
	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "MixedCaseGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByName(0, "mixedcaseguild")

	require.NoError(t, err)
	assert.Equal(t, e.Id, result.Id())
}

func TestProcessor_GetSlice_Empty(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetSlice()

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessor_GetSlice_ReturnsGuilds(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entities directly in database
	e1 := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "Guild1",
		LeaderId: 100,
		Capacity: 30,
	}
	e2 := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "Guild2",
		LeaderId: 200,
		Capacity: 30,
	}
	db.Create(&e1)
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetSlice()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestProcessor_GetSlice_WithMemberFilter(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test entities directly in database
	e1 := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "GuildWithMember",
		LeaderId: 100,
		Capacity: 30,
	}
	e2 := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "GuildWithoutMember",
		LeaderId: 200,
		Capacity: 30,
	}
	db.Create(&e1)
	db.Create(&e2)

	// Add member to first guild
	m := member.Entity{
		TenantId:    ten.Id(),
		GuildId:     e1.Id,
		CharacterId: 500,
		Name:        "TestMember",
		Level:       50,
	}
	db.Create(&m)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetSlice(MemberFilter(500))

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, e1.Id, result[0].Id())
}

func TestProcessor_TenantIsolation(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create entity for tenant 1
	e1 := Entity{
		TenantId: ten1.Id(),
		WorldId:  0,
		Name:     "Tenant1Guild",
		LeaderId: 100,
		Capacity: 30,
	}
	db.Create(&e1)

	// Create entity for tenant 2 (should not be visible)
	e2 := Entity{
		TenantId: ten2.Id(),
		WorldId:  0,
		Name:     "Tenant2Guild",
		LeaderId: 200,
		Capacity: 30,
	}
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetSlice()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Tenant1Guild", result[0].name)
}

func TestProcessor_GetByMemberId_NotFound(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	_, err := p.GetByMemberId(9999)

	assert.Error(t, err)
}

func TestProcessor_GetByMemberId_ReturnsGuild(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	// Create test guild
	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "TestGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	db.Create(&e)

	// Create character-guild association
	c := character.Entity{
		TenantId:    ten.Id(),
		CharacterId: 500,
		GuildId:     e.Id,
	}
	db.Create(&c)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByMemberId(500)

	require.NoError(t, err)
	assert.Equal(t, e.Id, result.Id())
	assert.Equal(t, "TestGuild", result.name)
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
