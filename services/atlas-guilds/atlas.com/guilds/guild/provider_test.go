package guild

import (
	"atlas-guilds/guild/member"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func newGuildsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	// title.Migration uses PostgreSQL-specific uuid_generate_v4() and cannot run on
	// sqlite; create the titles table directly so the guild preloads have a target.
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	db := databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, titlesMigration)
	tidA, tidB := uuid.New(), uuid.New()
	// Both tenants get a guild with the same name in the same world to prove
	// isolation by tenant, not by primary key. Ids differ because the guild
	// table's primary key is autoincrement-scoped globally, not per-tenant.
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidA, WorldId: 0, Name: "Phoenix", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 2, TenantId: tidB, WorldId: 0, Name: "Phoenix", LeaderId: 200}).Error)
	return db, tidA, tidB
}

func TestGuildProvider_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newGuildsDB(t)

	// Tenant A queries its own id 1 and sees its row.
	gotA, err := getById(1)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, uint32(100), gotA.LeaderId, "tenant A's row")

	// Tenant B queries its own id 2 and sees its row.
	gotB, err := getById(2)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, uint32(200), gotB.LeaderId, "tenant B's row")

	// Critically, tenant B asking for id 1 (which exists, but belongs to tenant A)
	// must not return tenant A's row.
	_, err = getById(1)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.Error(t, err, "tenant B must not see tenant A's row by id")
}

func TestGuildProvider_GetForName_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newGuildsDB(t)
	results, err := getForName(world.Id(0), "Phoenix")(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, results, 1, "even though both tenants have a 'Phoenix' in world 0, only tenant A's row is returned")
	assert.Equal(t, uint32(100), results[0].LeaderId)
}

func TestGuildProvider_GetAll_FiltersByTenant(t *testing.T) {
	db, _, tidB := newGuildsDB(t)
	page := model.Page{Number: 1, Size: 50}
	all, err := getAll(page)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, all.Items, 1, "GetAll must not leak across tenants")
	assert.Equal(t, uint32(200), all.Items[0].LeaderId)
	assert.Equal(t, 1, all.Total)
}

func TestGuildProvider_GetByNameLike_MatchesSubstringCaseInsensitive(t *testing.T) {
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	db := databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, titlesMigration)
	tid := uuid.New()
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tid, WorldId: 0, Name: "Alpha", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 2, TenantId: tid, WorldId: 0, Name: "alphabet", LeaderId: 101}).Error)
	require.NoError(t, db.Create(&Entity{Id: 3, TenantId: tid, WorldId: 0, Name: "Beta", LeaderId: 102}).Error)

	page := model.Page{Number: 1, Size: 50}
	paged, err := getByNameLike("alpha", page)(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	assert.Equal(t, 2, paged.Total)
	require.Len(t, paged.Items, 2)
}

func TestGuildProvider_GetByNameLike_EscapesPercentAndUnderscore(t *testing.T) {
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	db := databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, titlesMigration)
	tid := uuid.New()
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tid, WorldId: 0, Name: "100%_raw", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 2, TenantId: tid, WorldId: 0, Name: "100xraw", LeaderId: 101}).Error)

	page := model.Page{Number: 1, Size: 50}
	paged, err := getByNameLike("0%_r", page)(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 1, "literal %/_ must be treated as literal characters, not wildcards")
	assert.Equal(t, uint32(100), paged.Items[0].LeaderId)
}

func TestGuildProvider_GetByNameLike_PagingComposition(t *testing.T) {
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	db := databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, titlesMigration)
	tid := uuid.New()
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Create(&Entity{Id: uint32(i + 1), TenantId: tid, WorldId: 0, Name: "MatchGuild", LeaderId: uint32(100 + i)}).Error)
	}

	page := model.Page{Number: 2, Size: 2}
	paged, err := getByNameLike("match", page)(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	assert.Equal(t, 5, paged.Total)
	require.Len(t, paged.Items, 2)
}

func TestGuildProvider_GetById_PreloadsAreTenantScoped(t *testing.T) {
	// title.Migration uses PostgreSQL-specific uuid_generate_v4() and cannot run on
	// sqlite; create the titles table directly so the guild preloads have a target.
	titlesMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS titles (
			tenant_id TEXT NOT NULL,
			id TEXT,
			guild_id INTEGER,
			name TEXT,
			"index" INTEGER
		)`).Error
	}
	db := databasetest.NewInMemoryTenantDB(t, Migration, member.Migration, titlesMigration)
	tidA, tidB := uuid.New(), uuid.New()

	// Same guild name across tenants. Ids differ because the guild table's PK is
	// autoincrement-scoped globally on sqlite, not per-tenant.
	require.NoError(t, db.Create(&Entity{Id: 1, TenantId: tidA, WorldId: 0, Name: "Phoenix", LeaderId: 100}).Error)
	require.NoError(t, db.Create(&Entity{Id: 2, TenantId: tidB, WorldId: 0, Name: "Phoenix", LeaderId: 200}).Error)

	// member.Entity's primaryKey is CharacterId alone (autoincrement disabled),
	// so duplicates across tenants would violate the PK. Use different
	// CharacterIds, but seed both with GuildId=1 so an un-tenanted preload of
	// tenant A's guild (Id=1) would erroneously pull in tenant B's member.
	require.NoError(t, db.Create(&member.Entity{CharacterId: 11, TenantId: tidA, GuildId: 1, Name: "alice", Level: 10}).Error)
	require.NoError(t, db.Create(&member.Entity{CharacterId: 12, TenantId: tidB, GuildId: 1, Name: "bob", Level: 10}).Error)

	got, err := getById(1)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, got.Members, 1, "preload must not leak tenant B's members")
	assert.Equal(t, "alice", got.Members[0].Name)
}
