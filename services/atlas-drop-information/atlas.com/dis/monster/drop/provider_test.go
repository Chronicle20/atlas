package drop

import (
	"testing"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newDropsDB seeds two monster-drop rows in two tenants that overlap on
// MonsterId so a leak across the tenant boundary is observable. The primary
// key is autoincrement and sqlite enforces global uniqueness, so the two rows
// use different ids (1 and 2).
func newDropsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity{ID: 1, TenantId: tidA, MonsterId: 100, ItemId: 2000000, Chance: 1000}).Error)
	require.NoError(t, db.Create(&entity{ID: 2, TenantId: tidB, MonsterId: 100, ItemId: 2000001, Chance: 2000}).Error)
	return db, tidA, tidB
}

func TestMonsterDropProvider_GetByMonsterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newDropsDB(t)

	rowsA, err := getByMonsterId(100)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rowsA, 1, "monster 100 has drops in both tenants — only tenant A's row should return")
	assert.Equal(t, tidA, rowsA[0].TenantId)
	assert.Equal(t, uint32(1000), rowsA[0].Chance)

	rowsB, err := getByMonsterId(100)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rowsB, 1)
	assert.Equal(t, tidB, rowsB[0].TenantId)
	assert.Equal(t, uint32(2000), rowsB[0].Chance)
}

func TestMonsterDropAdministrator_BulkCreate_RowsLandUnderContextTenant(t *testing.T) {
	// drop has no Update/Delete administrator — only BulkCreate. The
	// tenantCreateCallback's injection-from-context behaviour is exercised by
	// the F6 regression in libs/atlas-database, so this is the plan-prescribed
	// no-op assertion: BulkCreate succeeds under a tenant context and the rows
	// reflect that tenant.
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA := uuid.New()

	mdl, err := NewMonsterDropBuilder(tidA, 0).
		SetMonsterId(100).
		SetItemId(2000000).
		SetChance(1000).
		Build()
	require.NoError(t, err)

	err = BulkCreateMonsterDrop(db.WithContext(databasetest.TenantContext(tidA)), []Model{mdl})
	require.NoError(t, err)

	var rows []entity
	require.NoError(t, db.Unscoped().Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, tidA, rows[0].TenantId)
}
