package definition

import (
	"context"
	"testing"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	database.RegisterTenantCallbacks(l, db)
	require.NoError(t, MigrateTable(db))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

func setupTestContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	return ctx, ten
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testDefinition(questId, name string) Model {
	m, _ := NewBuilder().
		SetQuestId(questId).
		SetName(name).
		SetDuration(1800).
		SetExit(100000000).
		Build()
	return m
}

func TestProcessor_Create(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	m := testDefinition("kerning_pq", "Kerning PQ")
	created, err := p.Create(m)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.Id())
	assert.Equal(t, "kerning_pq", created.QuestId())
	assert.Equal(t, "Kerning PQ", created.Name())
	assert.Equal(t, uint64(1800), created.Duration())
	assert.Equal(t, uint32(100000000), created.Exit())
}

func TestProcessor_ByIdProvider(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	m := testDefinition("ludi_pq", "Ludi PQ")
	created, err := p.Create(m)
	require.NoError(t, err)

	got, err := p.ByIdProvider(created.Id())()
	require.NoError(t, err)
	assert.Equal(t, created.Id(), got.Id())
	assert.Equal(t, "ludi_pq", got.QuestId())
}

func TestProcessor_ByIdProvider_NotFound(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	_, err := p.ByIdProvider(uuid.New())()
	assert.Error(t, err)
}

func TestProcessor_ByQuestIdProvider(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	m := testDefinition("orbis_pq", "Orbis PQ")
	_, err := p.Create(m)
	require.NoError(t, err)

	got, err := p.ByQuestIdProvider("orbis_pq")()
	require.NoError(t, err)
	assert.Equal(t, "orbis_pq", got.QuestId())
	assert.Equal(t, "Orbis PQ", got.Name())
}

func TestProcessor_AllProvider(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	_, err := p.Create(testDefinition("pq_1", "PQ 1"))
	require.NoError(t, err)
	_, err = p.Create(testDefinition("pq_2", "PQ 2"))
	require.NoError(t, err)

	all, err := p.AllProvider()()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestProcessor_Delete(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	created, err := p.Create(testDefinition("del_pq", "Delete PQ"))
	require.NoError(t, err)

	err = p.Delete(created.Id())
	require.NoError(t, err)

	_, err = p.ByIdProvider(created.Id())()
	assert.Error(t, err)
}

func TestProcessor_DeleteAllForTenant(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)

	p := NewProcessor(testLogger(), ctx, db)

	_, _ = p.Create(testDefinition("pq_a", "PQ A"))
	_, _ = p.Create(testDefinition("pq_b", "PQ B"))

	count, err := p.DeleteAllForTenant()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	all, err := p.AllProvider()()
	require.NoError(t, err)
	assert.Len(t, all, 0)
}

func TestProcessor_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := tenant.WithContext(context.Background(), ten1)
	ctx2 := tenant.WithContext(context.Background(), ten2)

	l := testLogger()
	p1 := NewProcessor(l, ctx1, db)
	p2 := NewProcessor(l, ctx2, db)

	_, err := p1.Create(testDefinition("iso_pq", "Isolation PQ"))
	require.NoError(t, err)

	all1, err := p1.AllProvider()()
	require.NoError(t, err)
	assert.Len(t, all1, 1)

	all2, err := p2.AllProvider()()
	require.NoError(t, err)
	assert.Len(t, all2, 0)
}
