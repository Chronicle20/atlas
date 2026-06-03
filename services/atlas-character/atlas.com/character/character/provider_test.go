package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newCharsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	// Same accountId and name across tenants — prove isolation by tenant only.
	// Ids differ because the characters table's primary key is autoincrement-scoped
	// globally on sqlite, not per-tenant.
	require.NoError(t, db.Create(&entity{ID: 1, TenantId: tidA, AccountId: 7, World: 0, Name: "Hero", Level: 1, JobId: 0}).Error)
	require.NoError(t, db.Create(&entity{ID: 2, TenantId: tidB, AccountId: 7, World: 0, Name: "Hero", Level: 200, JobId: 0}).Error)
	return db, tidA, tidB
}

func TestCharacterProvider_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newCharsDB(t)

	// Tenant A queries its own id 1 and sees its row.
	gotA, err := getById(1)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, byte(1), gotA.Level, "tenant A's row")

	// Tenant B queries its own id 2 and sees its row.
	gotB, err := getById(2)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, byte(200), gotB.Level, "tenant B's row")

	// Critically, tenant B asking for id 1 (which exists, but belongs to tenant A)
	// must not return tenant A's row.
	_, err = getById(1)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.Error(t, err, "tenant B must not see tenant A's row by id")
}

func TestCharacterProvider_GetForAccount_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newCharsDB(t)
	rows, err := getForAccount(7)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rows, 1, "account 7 has overlapping characters across tenants — only tenant A's should return")
	assert.Equal(t, byte(1), rows[0].Level)
}

func TestCharacterProvider_GetForAccountInWorld_FiltersByTenant(t *testing.T) {
	db, _, tidB := newCharsDB(t)
	rows, err := getForAccountInWorld(7, world.Id(0))(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, byte(200), rows[0].Level)
}

func TestCharacterProvider_GetForName_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newCharsDB(t)
	rows, err := getForName("Hero")(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, byte(1), rows[0].Level)
}

func TestCharacterProvider_GetAll_FiltersByTenant(t *testing.T) {
	db, _, tidB := newCharsDB(t)
	rows, err := getAll()(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rows, 1, "GetAll must not leak across tenants")
	assert.Equal(t, byte(200), rows[0].Level)
}
